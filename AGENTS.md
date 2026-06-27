# Agent Instructions

## Project context

This repository contains a terminal Snake game written in Go.

- `main.go` owns CLI flags, terminal input, rendering, pause/restart/quit commands, and raw terminal mode.
- `internal/game` owns deterministic game rules and tests.
- The current terminal implementation uses Darwin/BSD-style `syscall.Termios` constants, so do not claim Linux support unless the code has been changed and verified there.
- Keep documentation concise and practical for GitHub readers.

## Development guidelines

- Keep changes scoped to the requested behavior.
- Follow the existing Go style and run `gofmt` on any Go files you edit.
- Prefer deterministic tests in `internal/game` for gameplay-rule changes.
- Do not add new dependencies unless they are needed for the requested change.

## Required validation

Whenever you complete an implementation request in this project:

1. Run `go test ./...`.
2. Generate the executable at the project root only if the tests pass:
   `go build -o ./snake .`
3. If the tests fail, fix the problem before generating the executable.
4. Report in the final summary which tests were run and whether the executable was generated.
