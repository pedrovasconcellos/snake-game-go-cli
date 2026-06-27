package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"snake/internal/game"
)

type terminalMode struct {
	state syscall.Termios
}

const (
	colorGreen  = "\033[32m"
	colorOrange = "\033[38;5;208m"
	colorReset  = "\033[0m"

	recoveryDuration = 3 * time.Second
	blinkDuration    = 250 * time.Millisecond
)

func main() {
	width := flag.Int("width", 30, "board width")
	height := flag.Int("height", 18, "board height")
	mode := flag.String("mode", string(game.ModeClassic), "mode: classic or obstacles")
	flag.Parse()

	config := game.Config{
		Width:  *width,
		Height: *height,
		Mode:   game.Mode(*mode),
	}
	g, err := game.New(config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	terminal, err := enableRawMode()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error configuring terminal:", err)
		os.Exit(1)
	}
	defer terminal.restore()
	defer showCursor()

	input := make(chan game.Direction, 8)
	commands := make(chan rune, 8)
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go readInput(input, commands)
	clearScreen()
	hideCursor()

	paused := false
	recoveringUntil := time.Time{}
	showRecoveringSnake := true
	timer := time.NewTimer(g.TickDuration())
	defer timer.Stop()
	for {
		render(g.State(), paused, showRecoveringSnake)

		select {
		case direction := <-input:
			g.ChangeDirection(direction)
		case command := <-commands:
			switch command {
			case 'p':
				if !g.State().GameOver {
					paused = !paused
				}
			case 'q':
				return
			case 'r':
				next, err := game.New(config)
				if err != nil {
					fmt.Fprintln(os.Stderr, "error restarting:", err)
					return
				}
				g = next
				paused = false
				recoveringUntil = time.Time{}
				showRecoveringSnake = true
				resetTimer(timer, g.TickDuration())
			}
		case <-timer.C:
			state := g.State()
			switch {
			case state.Recovering:
				if !time.Now().Before(recoveringUntil) {
					g.Resume()
					recoveringUntil = time.Time{}
					showRecoveringSnake = true
					resetTimer(timer, g.TickDuration())
					continue
				}
				showRecoveringSnake = !showRecoveringSnake
				resetTimer(timer, blinkDuration)
			case !paused && !state.GameOver:
				g.Tick()
				if g.State().Recovering {
					recoveringUntil = time.Now().Add(recoveryDuration)
					showRecoveringSnake = false
					resetTimer(timer, blinkDuration)
				} else {
					resetTimer(timer, g.TickDuration())
				}
			default:
				resetTimer(timer, g.TickDuration())
			}
		case <-done:
			return
		}
	}
}

func render(state game.State, paused bool, showSnake bool) {
	var builder strings.Builder
	builder.WriteString("\033[2J\033[H")
	builder.WriteString("Snake - mode ")
	builder.WriteString(stringMode(state.Obstacles))
	builder.WriteString(fmt.Sprintf(" | HP: %s | score: %d | length: %d | P pause | R restart | Q quit\n", hearts(state.Lives, state.MaxLives), state.Score, len(state.Snake)))

	for x := 0; x < state.Width+2; x++ {
		writeWall(&builder)
	}
	builder.WriteByte('\n')

	head := state.Snake[0]
	body := map[game.Point]byte{}
	for index, point := range state.Snake {
		if index == 0 {
			body[point] = '@'
			continue
		}
		body[point] = 'o'
	}

	for y := 0; y < state.Height; y++ {
		writeWall(&builder)
		for x := 0; x < state.Width; x++ {
			point := game.Point{X: x, Y: y}
			switch {
			case showSnake && point == head:
				builder.WriteByte('@')
			case showSnake && body[point] != 0:
				builder.WriteByte('o')
			case point == state.Food:
				builder.WriteString(colorOrange)
				builder.WriteByte('*')
				builder.WriteString(colorReset)
			case state.Obstacles[point]:
				builder.WriteByte('X')
			default:
				builder.WriteByte(' ')
			}
		}
		writeWall(&builder)
		builder.WriteByte('\n')
	}

	for x := 0; x < state.Width+2; x++ {
		writeWall(&builder)
	}
	builder.WriteByte('\n')

	if paused {
		builder.WriteString("Paused\n")
	} else if state.Recovering {
		builder.WriteString(fmt.Sprintf("You lost a heart: %s. Reviving...\n", state.Message))
	} else if state.GameOver {
		builder.WriteString(fmt.Sprintf("Game over: %s. Final score: %d. Press R to restart or Q to quit.\n", state.Message, state.Score))
	} else {
		builder.WriteString("Use arrow keys or WASD to move.\n")
	}

	fmt.Print(builder.String())
}

func hearts(lives int, maxLives int) string {
	var builder strings.Builder
	for i := 0; i < maxLives; i++ {
		if i < lives {
			builder.WriteRune('♥')
		} else {
			builder.WriteRune('♡')
		}
	}
	return builder.String()
}

func writeWall(builder *strings.Builder) {
	builder.WriteString(colorGreen)
	builder.WriteByte('#')
	builder.WriteString(colorReset)
}

func stringMode(obstacles map[game.Point]bool) string {
	if len(obstacles) > 0 {
		return string(game.ModeObstacles)
	}
	return string(game.ModeClassic)
}

func readInput(directions chan<- game.Direction, commands chan<- rune) {
	reader := bufio.NewReader(os.Stdin)
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return
		}
		switch b {
		case 'w', 'W':
			directions <- game.Up
		case 'd', 'D':
			directions <- game.Right
		case 's', 'S':
			directions <- game.Down
		case 'a', 'A':
			directions <- game.Left
		case 'p', 'P':
			commands <- 'p'
		case 'q', 'Q':
			commands <- 'q'
		case 'r', 'R':
			commands <- 'r'
		case 0x1b:
			first, err := reader.ReadByte()
			if err != nil || first != '[' {
				continue
			}
			second, err := reader.ReadByte()
			if err != nil {
				continue
			}
			switch second {
			case 'A':
				directions <- game.Up
			case 'B':
				directions <- game.Down
			case 'C':
				directions <- game.Right
			case 'D':
				directions <- game.Left
			}
		}
	}
}

func resetTimer(timer *time.Timer, duration time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(duration)
}

func enableRawMode() (*terminalMode, error) {
	fd := int(os.Stdin.Fd())
	var old syscall.Termios
	if _, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TIOCGETA), uintptr(unsafePointer(&old)), 0, 0, 0); errno != 0 {
		return nil, errno
	}

	next := old
	next.Iflag &^= syscall.IXON | syscall.ICRNL
	next.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.ISIG
	next.Cc[syscall.VMIN] = 1
	next.Cc[syscall.VTIME] = 0

	if _, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TIOCSETA), uintptr(unsafePointer(&next)), 0, 0, 0); errno != 0 {
		return nil, errno
	}
	return &terminalMode{state: old}, nil
}

func (t *terminalMode) restore() {
	fd := int(os.Stdin.Fd())
	_, _, _ = syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TIOCSETA), uintptr(unsafePointer(&t.state)), 0, 0, 0)
}

func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

func hideCursor() {
	fmt.Print("\033[?25l")
}

func showCursor() {
	fmt.Print("\033[?25h")
}

func unsafePointer(value *syscall.Termios) unsafe.Pointer {
	return unsafe.Pointer(value)
}
