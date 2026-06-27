package game

import (
	"errors"
	"math/rand"
	"time"
)

type Direction int

const (
	Up Direction = iota
	Right
	Down
	Left
)

type Point struct {
	X int
	Y int
}

type Mode string

const (
	ModeClassic   Mode = "classic"
	ModeObstacles Mode = "obstacles"
)

type Config struct {
	Width    int
	Height   int
	Mode     Mode
	Seed     int64
	MaxLives int
}

type State struct {
	Width      int
	Height     int
	Snake      []Point
	Food       Point
	Obstacles  map[Point]bool
	Direction  Direction
	Score      int
	Lives      int
	MaxLives   int
	Recovering bool
	GameOver   bool
	Message    string
}

type Game struct {
	state              State
	rng                *rand.Rand
	queuedDirection    Direction
	hasQueuedDirection bool
}

func New(config Config) (*Game, error) {
	if config.Width < 12 || config.Height < 8 {
		return nil, errors.New("board must be at least 12x8")
	}
	if config.Mode == "" {
		config.Mode = ModeClassic
	}
	if config.Mode != ModeClassic && config.Mode != ModeObstacles {
		return nil, errors.New("invalid mode")
	}
	if config.Seed == 0 {
		config.Seed = time.Now().UnixNano()
	}
	if config.MaxLives <= 0 {
		config.MaxLives = 3
	}

	mid := Point{X: config.Width / 2, Y: config.Height / 2}
	snake := []Point{
		mid,
		{X: mid.X - 1, Y: mid.Y},
		{X: mid.X - 2, Y: mid.Y},
	}

	g := &Game{
		state: State{
			Width:     config.Width,
			Height:    config.Height,
			Snake:     snake,
			Obstacles: buildObstacles(config.Width, config.Height, config.Mode, snake),
			Direction: Right,
			Lives:     config.MaxLives,
			MaxLives:  config.MaxLives,
		},
		rng: rand.New(rand.NewSource(config.Seed)),
	}
	g.state.Food = g.randomFreePoint()
	return g, nil
}

func (g *Game) State() State {
	state := g.state
	state.Snake = append([]Point(nil), g.state.Snake...)
	state.Obstacles = make(map[Point]bool, len(g.state.Obstacles))
	for point, blocked := range g.state.Obstacles {
		state.Obstacles[point] = blocked
	}
	return state
}

func (g *Game) ChangeDirection(next Direction) {
	if g.state.GameOver || g.state.Recovering || g.hasQueuedDirection || isOpposite(g.state.Direction, next) {
		return
	}
	g.queuedDirection = next
	g.hasQueuedDirection = true
}

func (g *Game) Tick() {
	if g.state.GameOver || g.state.Recovering {
		return
	}
	if g.hasQueuedDirection {
		g.state.Direction = g.queuedDirection
		g.hasQueuedDirection = false
	}

	head := g.state.Snake[0]
	next := move(head, g.state.Direction)
	if g.hitsWall(next) {
		g.hit("hit a wall")
		return
	}
	if g.state.Obstacles[next] {
		g.hit("hit an obstacle")
		return
	}

	eats := next == g.state.Food
	bodyToCheck := g.state.Snake
	if !eats {
		bodyToCheck = g.state.Snake[:len(g.state.Snake)-1]
	}
	if contains(bodyToCheck, next) {
		g.hit("hit itself")
		return
	}

	g.state.Snake = append([]Point{next}, g.state.Snake...)
	if eats {
		g.state.Score++
		g.state.Food = g.randomFreePoint()
		return
	}
	g.state.Snake = g.state.Snake[:len(g.state.Snake)-1]
}

func (g *Game) TickDuration() time.Duration {
	score := g.state.Score
	duration := 180*time.Millisecond - time.Duration(score/3)*15*time.Millisecond
	if duration < 70*time.Millisecond {
		return 70 * time.Millisecond
	}
	return duration
}

func (g *Game) end(message string) {
	g.state.GameOver = true
	g.state.Message = message
}

func (g *Game) hit(message string) {
	g.state.Lives--
	g.state.Message = message
	g.hasQueuedDirection = false
	if g.state.Lives <= 0 {
		g.state.Lives = 0
		g.end(message)
		return
	}
	if !g.respawnSnake() {
		g.end("no safe space to revive")
		return
	}
	g.state.Recovering = true
}

func (g *Game) Resume() {
	if g.state.GameOver {
		return
	}
	g.state.Recovering = false
	g.state.Message = ""
}

func (g *Game) hitsWall(point Point) bool {
	return point.X < 0 || point.Y < 0 || point.X >= g.state.Width || point.Y >= g.state.Height
}

func (g *Game) respawnSnake() bool {
	length := len(g.state.Snake)
	center := Point{X: g.state.Width / 2, Y: g.state.Height / 2}
	if g.hitsWall(center) || g.state.Obstacles[center] {
		return false
	}

	snake, ok := g.buildRespawnSnake(center, length)
	if !ok {
		return false
	}
	direction, ok := g.safeRespawnDirection(snake)
	if !ok {
		return false
	}

	g.state.Snake = snake
	g.state.Direction = direction
	if contains(g.state.Snake, g.state.Food) {
		g.state.Food = g.randomFreePoint()
	}
	return true
}

func (g *Game) buildRespawnSnake(center Point, length int) ([]Point, bool) {
	snake := []Point{center}
	used := map[Point]bool{center: true}
	if length == 1 {
		return snake, true
	}
	if g.extendRespawnSnake(&snake, used, length) {
		return snake, true
	}
	return nil, false
}

func (g *Game) extendRespawnSnake(snake *[]Point, used map[Point]bool, length int) bool {
	if len(*snake) == length {
		return true
	}

	current := (*snake)[len(*snake)-1]
	for _, next := range g.respawnCandidates(current, used) {
		used[next] = true
		*snake = append(*snake, next)
		if g.extendRespawnSnake(snake, used, length) {
			return true
		}
		*snake = (*snake)[:len(*snake)-1]
		delete(used, next)
	}
	return false
}

func (g *Game) respawnCandidates(point Point, used map[Point]bool) []Point {
	candidates := make([]Point, 0, 4)
	for _, direction := range []Direction{Left, Up, Down, Right} {
		next := move(point, direction)
		if g.hitsWall(next) || g.state.Obstacles[next] || used[next] {
			continue
		}
		candidates = append(candidates, next)
	}

	for i := 1; i < len(candidates); i++ {
		for j := i; j > 0 && g.availableNeighbors(candidates[j], used) < g.availableNeighbors(candidates[j-1], used); j-- {
			candidates[j], candidates[j-1] = candidates[j-1], candidates[j]
		}
	}
	return candidates
}

func (g *Game) availableNeighbors(point Point, used map[Point]bool) int {
	count := 0
	for _, direction := range []Direction{Up, Right, Down, Left} {
		next := move(point, direction)
		if !g.hitsWall(next) && !g.state.Obstacles[next] && !used[next] {
			count++
		}
	}
	return count
}

func (g *Game) safeRespawnDirection(snake []Point) (Direction, bool) {
	occupied := make(map[Point]bool, len(snake))
	for _, point := range snake {
		occupied[point] = true
	}

	for _, direction := range []Direction{Right, Down, Up, Left} {
		next := move(snake[0], direction)
		if g.hitsWall(next) || g.state.Obstacles[next] || occupied[next] {
			continue
		}
		return direction, true
	}
	return Right, false
}

func (g *Game) randomFreePoint() Point {
	free := make([]Point, 0, g.state.Width*g.state.Height)
	for y := 0; y < g.state.Height; y++ {
		for x := 0; x < g.state.Width; x++ {
			point := Point{X: x, Y: y}
			if contains(g.state.Snake, point) || g.state.Obstacles[point] {
				continue
			}
			free = append(free, point)
		}
	}
	if len(free) == 0 {
		g.end("you won")
		return Point{}
	}
	return free[g.rng.Intn(len(free))]
}

func move(point Point, direction Direction) Point {
	switch direction {
	case Up:
		return Point{X: point.X, Y: point.Y - 1}
	case Right:
		return Point{X: point.X + 1, Y: point.Y}
	case Down:
		return Point{X: point.X, Y: point.Y + 1}
	case Left:
		return Point{X: point.X - 1, Y: point.Y}
	default:
		return point
	}
}

func isOpposite(current Direction, next Direction) bool {
	return current == Up && next == Down ||
		current == Down && next == Up ||
		current == Left && next == Right ||
		current == Right && next == Left
}

func contains(points []Point, target Point) bool {
	for _, point := range points {
		if point == target {
			return true
		}
	}
	return false
}

func buildObstacles(width int, height int, mode Mode, snake []Point) map[Point]bool {
	obstacles := map[Point]bool{}
	if mode != ModeObstacles {
		return obstacles
	}

	centerX := width / 2
	centerY := height / 2
	for x := 2; x < width-2; x++ {
		if x%4 == 0 {
			point := Point{X: x, Y: 2}
			if !contains(snake, point) {
				obstacles[point] = true
			}
			point = Point{X: x, Y: height - 3}
			if !contains(snake, point) {
				obstacles[point] = true
			}
		}
	}
	for y := 2; y < height-2; y++ {
		if y%3 == 0 {
			point := Point{X: 2, Y: y}
			if !contains(snake, point) {
				obstacles[point] = true
			}
			point = Point{X: width - 3, Y: y}
			if !contains(snake, point) {
				obstacles[point] = true
			}
		}
	}
	for _, point := range []Point{
		{X: centerX - 4, Y: centerY},
		{X: centerX + 4, Y: centerY},
		{X: centerX, Y: centerY - 3},
		{X: centerX, Y: centerY + 3},
	} {
		if point.X >= 0 && point.Y >= 0 && point.X < width && point.Y < height && !contains(snake, point) {
			obstacles[point] = true
		}
	}
	return obstacles
}
