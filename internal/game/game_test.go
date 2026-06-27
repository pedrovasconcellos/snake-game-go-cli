package game

import "testing"

func TestMoveRightByDefault(t *testing.T) {
	g := newTestGame(t, Config{Width: 20, Height: 12, Mode: ModeClassic, Seed: 1})

	before := g.State().Snake[0]
	g.Tick()
	after := g.State().Snake[0]

	if after != (Point{X: before.X + 1, Y: before.Y}) {
		t.Fatalf("expected head to move right, got %+v from %+v", after, before)
	}
}

func TestRejectsDirectReverse(t *testing.T) {
	g := newTestGame(t, Config{Width: 20, Height: 12, Mode: ModeClassic, Seed: 1})

	g.ChangeDirection(Left)
	g.Tick()

	state := g.State()
	if state.Direction != Right {
		t.Fatalf("expected direction to remain right, got %v", state.Direction)
	}
	if state.GameOver {
		t.Fatal("reverse input should not end the game")
	}
}

func TestAllowsOnlyOneDirectionChangePerTick(t *testing.T) {
	g := newTestGame(t, Config{Width: 20, Height: 12, Mode: ModeClassic, Seed: 1})
	before := g.State().Snake[0]

	g.ChangeDirection(Up)
	g.ChangeDirection(Left)
	g.Tick()

	state := g.State()
	expected := Point{X: before.X, Y: before.Y - 1}
	if state.Snake[0] != expected {
		t.Fatalf("expected first queued direction to win and move to %+v, got %+v", expected, state.Snake[0])
	}
	if state.Direction != Up {
		t.Fatalf("expected direction up, got %v", state.Direction)
	}
	if state.GameOver {
		t.Fatal("rapid direction sequence should not cause immediate self collision")
	}
}

func TestAllowsNewDirectionChangeAfterTick(t *testing.T) {
	g := newTestGame(t, Config{Width: 20, Height: 12, Mode: ModeClassic, Seed: 1})

	g.ChangeDirection(Up)
	g.Tick()
	g.ChangeDirection(Left)
	g.Tick()

	state := g.State()
	if state.Direction != Left {
		t.Fatalf("expected direction left after second tick, got %v", state.Direction)
	}
}

func TestSnakeGrowsWhenEatingFood(t *testing.T) {
	g := newTestGame(t, Config{Width: 20, Height: 12, Mode: ModeClassic, Seed: 1})
	state := g.State()
	g.state.Food = Point{X: state.Snake[0].X + 1, Y: state.Snake[0].Y}

	g.Tick()

	after := g.State()
	if after.Score != 1 {
		t.Fatalf("expected score 1, got %d", after.Score)
	}
	if len(after.Snake) != len(state.Snake)+1 {
		t.Fatalf("expected snake to grow from %d to %d, got %d", len(state.Snake), len(state.Snake)+1, len(after.Snake))
	}
}

func TestWallCollisionEndsGame(t *testing.T) {
	g := newTestGame(t, Config{Width: 12, Height: 8, Mode: ModeClassic, Seed: 1})
	for !g.State().Recovering {
		g.Tick()
	}

	state := g.State()
	if state.Message != "hit a wall" {
		t.Fatalf("expected wall collision message, got %q", state.Message)
	}
	if state.GameOver {
		t.Fatal("expected first wall collision to recover instead of ending game")
	}
	if state.Lives != 2 {
		t.Fatalf("expected 2 lives after first collision, got %d", state.Lives)
	}
}

func TestSelfCollisionEndsGame(t *testing.T) {
	g := newTestGame(t, Config{Width: 20, Height: 12, Mode: ModeClassic, Seed: 1})
	g.state.Snake = []Point{
		{X: 5, Y: 5},
		{X: 5, Y: 6},
		{X: 4, Y: 6},
		{X: 4, Y: 5},
		{X: 4, Y: 4},
	}
	g.state.Direction = Left
	g.ChangeDirection(Down)
	g.Tick()

	state := g.State()
	if state.GameOver {
		t.Fatal("expected first self collision to recover instead of ending game")
	}
	if state.Message != "hit itself" {
		t.Fatalf("expected self collision message, got %q", state.Message)
	}
	if !state.Recovering {
		t.Fatal("expected self collision to enter recovering state")
	}
}

func TestObstacleCollisionEndsGame(t *testing.T) {
	g := newTestGame(t, Config{Width: 20, Height: 12, Mode: ModeObstacles, Seed: 1})
	head := g.State().Snake[0]
	g.state.Obstacles = map[Point]bool{{X: head.X + 1, Y: head.Y}: true}

	g.Tick()

	state := g.State()
	if state.GameOver {
		t.Fatal("expected first obstacle collision to recover instead of ending game")
	}
	if state.Message != "hit an obstacle" {
		t.Fatalf("expected obstacle collision message, got %q", state.Message)
	}
	if state.Lives != 2 {
		t.Fatalf("expected 2 lives after obstacle collision, got %d", state.Lives)
	}
}

func TestRecoveringRespawnsInCenterUntilResume(t *testing.T) {
	g := newTestGame(t, Config{Width: 12, Height: 8, Mode: ModeClassic, Seed: 1})
	g.state.Snake = []Point{
		{X: 11, Y: 4},
		{X: 10, Y: 4},
		{X: 9, Y: 4},
	}
	g.state.Direction = Right
	before := g.State()
	center := Point{X: before.Width / 2, Y: before.Height / 2}

	g.Tick()
	afterHit := g.State()
	g.Tick()
	afterRecoveringTick := g.State()

	if !afterHit.Recovering {
		t.Fatal("expected recovering after collision")
	}
	if len(afterHit.Snake) != len(before.Snake) {
		t.Fatalf("expected respawn to keep size %d, got %d", len(before.Snake), len(afterHit.Snake))
	}
	if afterHit.Snake[0] != center {
		t.Fatalf("expected head to respawn at center %+v, got %+v", center, afterHit.Snake[0])
	}
	assertValidSnake(t, afterHit)
	if afterRecoveringTick.Snake[0] != center {
		t.Fatalf("expected recovering tick not to move snake, got %+v", afterRecoveringTick.Snake[0])
	}

	g.ChangeDirection(Up)
	g.Resume()
	g.Tick()

	afterResume := g.State()
	if afterResume.Recovering {
		t.Fatal("expected resume to leave recovering state")
	}
	if afterResume.Snake[0] != (Point{X: center.X + 1, Y: center.Y}) {
		t.Fatalf("expected snake to continue safely from center, got %+v", afterResume.Snake[0])
	}
}

func TestRespawnAvoidsObstaclesAndSelfCollision(t *testing.T) {
	g := newTestGame(t, Config{Width: 20, Height: 12, Mode: ModeObstacles, Seed: 1})
	center := Point{X: g.state.Width / 2, Y: g.state.Height / 2}
	g.state.Snake = []Point{
		{X: 19, Y: 6},
		{X: 18, Y: 6},
		{X: 17, Y: 6},
		{X: 16, Y: 6},
		{X: 15, Y: 6},
		{X: 14, Y: 6},
	}
	g.state.Direction = Right
	g.state.Obstacles[Point{X: center.X - 1, Y: center.Y}] = true

	g.Tick()

	state := g.State()
	if state.GameOver {
		t.Fatalf("expected safe respawn, got game over: %s", state.Message)
	}
	if !state.Recovering {
		t.Fatal("expected recovering after collision")
	}
	if state.Snake[0] != center {
		t.Fatalf("expected head to respawn at center %+v, got %+v", center, state.Snake[0])
	}
	if len(state.Snake) != 6 {
		t.Fatalf("expected respawn to keep size 6, got %d", len(state.Snake))
	}
	assertValidSnake(t, state)
}

func TestRespawnMovesFoodAwayFromRespawnedSnake(t *testing.T) {
	g := newTestGame(t, Config{Width: 12, Height: 8, Mode: ModeClassic, Seed: 1})
	g.state.Snake = []Point{
		{X: 11, Y: 4},
		{X: 10, Y: 4},
		{X: 9, Y: 4},
	}
	g.state.Direction = Right
	g.state.Food = Point{X: 6, Y: 4}

	g.Tick()

	state := g.State()
	if contains(state.Snake, state.Food) {
		t.Fatalf("expected food to move away from respawned snake, got food %+v snake %+v", state.Food, state.Snake)
	}
}

func TestGameEndsAfterThreeCollisions(t *testing.T) {
	g := newTestGame(t, Config{Width: 12, Height: 8, Mode: ModeClassic, Seed: 1})

	for collision := 0; collision < 3; collision++ {
		g.state.Snake = []Point{
			{X: 11, Y: 4},
			{X: 10, Y: 4},
			{X: 9, Y: 4},
		}
		g.state.Direction = Right
		g.Tick()
		if collision < 2 {
			if g.State().GameOver {
				t.Fatalf("collision %d ended game too early", collision+1)
			}
			g.Resume()
		}
	}

	state := g.State()
	if !state.GameOver {
		t.Fatal("expected game over after third collision")
	}
	if state.Lives != 0 {
		t.Fatalf("expected 0 lives, got %d", state.Lives)
	}
}

func TestFoodIsNotPlacedOnSnakeOrObstacle(t *testing.T) {
	g := newTestGame(t, Config{Width: 20, Height: 12, Mode: ModeObstacles, Seed: 1})
	state := g.State()

	if contains(state.Snake, state.Food) {
		t.Fatalf("food was placed on snake: %+v", state.Food)
	}
	if state.Obstacles[state.Food] {
		t.Fatalf("food was placed on obstacle: %+v", state.Food)
	}
}

func TestTickDurationIncreasesDifficulty(t *testing.T) {
	g := newTestGame(t, Config{Width: 20, Height: 12, Mode: ModeClassic, Seed: 1})
	initial := g.TickDuration()
	g.state.Score = 12
	later := g.TickDuration()

	if later >= initial {
		t.Fatalf("expected duration to shrink as score grows, initial %s later %s", initial, later)
	}
}

func newTestGame(t *testing.T, config Config) *Game {
	t.Helper()
	g, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func assertValidSnake(t *testing.T, state State) {
	t.Helper()

	seen := map[Point]bool{}
	for index, point := range state.Snake {
		if point.X < 0 || point.Y < 0 || point.X >= state.Width || point.Y >= state.Height {
			t.Fatalf("snake point %d is out of bounds: %+v", index, point)
		}
		if state.Obstacles[point] {
			t.Fatalf("snake point %d overlaps obstacle: %+v", index, point)
		}
		if seen[point] {
			t.Fatalf("snake overlaps itself at point %d: %+v", index, point)
		}
		seen[point] = true
		if index > 0 && manhattanDistance(state.Snake[index-1], point) != 1 {
			t.Fatalf("snake points %d and %d are not adjacent: %+v %+v", index-1, index, state.Snake[index-1], point)
		}
	}
}

func manhattanDistance(a Point, b Point) int {
	dx := a.X - b.X
	if dx < 0 {
		dx = -dx
	}
	dy := a.Y - b.Y
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}
