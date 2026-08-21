# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-08-22

### Added

- Initial simulation service: `internal/simulation` wires grid coordinator, robot goroutines, tick fan-out, and renderer behind a single `simulation.Run(ctx)` call.
- `internal/grid`: grid coordinator goroutine with channel-based cell-claiming protocol (`ClaimRequest` / `GridSnapshot`); claim-before-move semantics guarantee mutual exclusion without mutexes.
- `internal/robot`: goroutine-per-robot with random-walk movement and collision avoidance; robots that receive a deny stay put and retry on the next tick.
- `internal/tui`: ANSI terminal renderer; redraws the grid in place at configurable FPS using escape sequences; robot IDs coloured by index.
- `internal/config`: CLI flag parsing (`--width`, `--height`, `--robots`, `--tick-ms`, `--fps`) with validation in `cmd/robotdemo/main.go`.
- `cmd/robotdemo`: entry point — signal handling via `signal.NotifyContext` (SIGINT / SIGTERM), graceful shutdown with full goroutine join.
- CI workflow (`.github/workflows/ci.yml`): runs `go vet`, `go test -race`, and `go build` on every push and pull request to `main`.
- Release workflow (`.github/workflows/release.yml`): builds a static binary (`CGO_ENABLED=0`) and uploads it as a GitHub Release asset on `v*.*.*` tags.
- `Makefile` with `build`, `test`, `vet`, `clean`, and `run` targets.

[Unreleased]: https://github.com/vlantonov/GoRobotControlDemo/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/vlantonov/GoRobotControlDemo/releases/tag/v0.1.0
