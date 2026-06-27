# Snake

A terminal Snake game written in Go, inspired by the classic Nokia-era version.

The snake grows when it eats food, the score increases, and the game gets faster as you improve. You can play the standard mode or enable obstacle mode for a tighter board.

## Features

- Classic terminal Snake gameplay.
- Optional obstacle mode.
- Score, snake length, and HP display.
- Three lives before game over.
- Temporary recovery after collisions.
- Increasing speed as the score rises.
- Keyboard controls with arrow keys or WASD.

## Requirements

- Go 1.24 or newer.
- A terminal that supports ANSI escape codes.
- macOS is the currently verified platform.

The current raw terminal implementation uses macOS/BSD terminal syscall constants. Linux support should be treated as unsupported until the terminal layer is made portable.

## Run

Start the default game:

```sh
go run .
```

Run with obstacles:

```sh
go run . -mode obstacles
```

Customize the board size:

```sh
go run . -width 40 -height 20
```

Combine options:

```sh
go run . -mode obstacles -width 40 -height 20
```

The board must be at least `12x8`.

## Controls

| Key | Action |
| --- | --- |
| Arrow keys | Move |
| WASD | Move |
| P | Pause or resume |
| R | Restart |
| Q | Quit |

## Build

Build the executable at the project root:

```sh
go build -o ./snake .
```

Run the built executable:

```sh
./snake
```

## Test

Run the full test suite:

```sh
go test ./...
```

The gameplay rules are covered in `internal/game`, including movement, growth, collisions, lives, recovery, obstacle handling, food placement, and difficulty scaling.

## Project layout

```text
.
|-- main.go
|-- go.mod
|-- internal/game
|   |-- game.go
|   `-- game_test.go
`-- memory-bank/GAME_DESCRIPTION.md
```

## License

No license file is included yet. Add one before publishing if you want to define how others may use, copy, or contribute to the project.
