# GoRobotControlDemo

![CI](https://github.com/vlantonov/GoRobotControlDemo/actions/workflows/ci.yml/badge.svg)

GoRobotControlDemo is a single-binary Go service that simulates a fleet of robots navigating a shared two-dimensional warehouse grid in real time. Each robot runs as an independent goroutine; all coordination is done through typed Go channels — no mutexes in the hot path. A terminal-based renderer redraws the grid in place using ANSI escape sequences. The project is a portfolio demonstration of idiomatic Go concurrency: goroutine-per-actor design, channel-based cell claiming, bounded goroutine lifetimes, and clean shutdown via `context.Context`.

---

## Features

- **Goroutine-per-robot** — every robot is an independent, long-lived goroutine with a clear termination path.
- **Channel-based cell claiming** — robots negotiate exclusive cell ownership through a serial coordinator goroutine; no two robots can occupy the same cell simultaneously.
- **Collision avoidance** — claim-before-move semantics guarantee mutual exclusion; denied robots retry on the next tick.
- **ANSI terminal visualization** — the grid animates in place; each robot is rendered in a distinct colour.
- **Configurable parameters** — grid dimensions, robot count, tick speed, and render FPS are all CLI flags.
- **Race-detector clean** — the test suite is run under `-race` in CI.

---

## Quick start

**Prerequisites:** Go 1.22 or later.

```sh
# Clone
git clone https://github.com/vlantonov/GoRobotControlDemo.git
cd GoRobotControlDemo

# Build and run (default: 20×10 grid, 5 robots, 100 ms tick, 10 FPS)
make run

# Or build a binary first
make build
./robotdemo

# Custom parameters
./robotdemo --width 30 --height 15 --robots 10 --tick-ms 50 --fps 20

# Stop with Ctrl-C (SIGINT) or SIGTERM — the simulation drains cleanly.
```

---

## CLI flags

| Flag | Default | Description |
|------|---------|-------------|
| `--width` | `20` | Grid width in cells (columns). |
| `--height` | `10` | Grid height in cells (rows). |
| `--robots` | `5` | Number of robots to spawn. Must not exceed `width × height`. |
| `--tick-ms` | `100` | Movement tick interval in milliseconds. |
| `--fps` | `10` | Render frames per second. |

---

## How it works

```
┌─────────────────────────── single binary process ────────────────────────────┐
│                                                                               │
│  main()                                                                       │
│  ├─ parse CLI flags → Config                                                  │
│  └─ signal.NotifyContext(SIGINT/SIGTERM) → ctx                                │
│                                                                               │
│  tickLoop goroutine                                                           │
│  └─ time.Ticker(tick-ms) ──fan-out──► tickCh[0] tickCh[1] … tickCh[N-1]     │
│                                           │         │               │         │
│  robot.Run goroutines ◄───────────────────┘─────────┘───────────────┘         │
│  └─ on each tick: shuffle directions, try neighbours                          │
│     └─ send ClaimRequest{From, To, ReplyTo} ──────────────────────────────┐  │
│                                                                            │  │
│  coordinator (grid.Run goroutine) ◄────────────────────────────────────────┘  │
│  └─ serial claim processor                                                    │
│     ├─ grant: update cells map, reply true, push GridSnapshot to snapshotCh  │
│     └─ deny:  reply false (robot stays put for this tick)                    │
│                           │                                                   │
│  renderer (tui.Run goroutine) ◄──── snapshotCh (buf=1, non-blocking send)    │
│  └─ time.Ticker(1s/fps) → draw to stdout via ANSI escape sequences           │
│                                                                               │
└───────────────────────────────────────────────────────────────────────────────┘
```

### Key channel types

| Channel | Direction | Type | Buffer |
|---------|-----------|------|--------|
| `tickCh[i]` | tickLoop → robot | `chan time.Time` | 1 |
| `claimCh` | robot → coordinator | `chan ClaimRequest` | `width × height` |
| `ReplyTo` (per request) | coordinator → robot | `chan bool` | 1 |
| `snapshotCh` | coordinator → renderer | `chan GridSnapshot` | 1 |

### Shutdown sequence

1. `signal.NotifyContext` cancels `ctx` on SIGINT / SIGTERM.
2. The tick fan-out goroutine exits; no new ticks are delivered.
3. All robot goroutines exit when they next observe `ctx.Done()`.
4. The coordinator drains buffered claim requests (denying all) and exits.
5. The renderer exits; ANSI cursor is restored.
6. `simulation.Run` joins every goroutine via `sync.WaitGroup` and returns.

---

## Running tests

```sh
# Run all tests with the race detector (matches CI)
make test

# Equivalent go command
go test -race -count=1 ./...

# Static analysis
make vet
```

---

## Project structure

```
GoRobotControlDemo/
├── cmd/
│   └── robotdemo/
│       └── main.go          # Entry point: flag parsing, signal handling, simulation.Run
├── internal/
│   ├── config/
│   │   └── config.go        # Config struct and flag.Parse wrapper
│   ├── grid/
│   │   └── grid.go          # Grid, ClaimRequest, GridSnapshot, coordinator Run
│   ├── robot/
│   │   └── robot.go         # Robot struct, random-walk Run goroutine
│   ├── simulation/
│   │   └── simulation.go    # Wires all components; owns WaitGroup
│   └── tui/
│       └── renderer.go      # ANSI terminal renderer
├── docs/
│   ├── design/
│   │   └── DESIGN.md        # Architecture and design decisions
│   └── requirements/
│       └── SRS.md           # Software Requirements Specification
├── .github/
│   └── workflows/
│       ├── ci.yml           # CI: vet + race test + build on push/PR to main
│       └── release.yml      # Release: static binary upload on v*.*.* tags
├── Makefile                 # build / test / vet / clean / run targets
├── go.mod
├── LICENSE
├── CHANGELOG.md
└── VERSION
```

---

## License

See [LICENSE](LICENSE).
