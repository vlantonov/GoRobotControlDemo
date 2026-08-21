# Software Requirements Specification — GoRobotControlDemo

**Version:** 1.0.0
**Date:** 2026-08-22
**Status:** Draft — awaiting System Architect review

---

## 1. Overview

GoRobotControlDemo is a single-binary Go service that simulates a fleet of N robots navigating a shared two-dimensional warehouse grid. The project exists as a portfolio demonstration of idiomatic Go concurrency: each robot runs as an independent goroutine, and all coordination is done through Go channels. A terminal-based real-time visualization renders the grid and robot positions. No external infrastructure (Docker, databases, network peers, or real hardware) is required.

---

## 2. Functional Requirements

### 2.1 Grid

| ID | Requirement |
|----|-------------|
| FR-01 | The service shall represent the warehouse as a finite rectangular grid of W columns × H rows, where W and H are positive integers greater than zero. |
| FR-02 | The grid dimensions W and H shall be configurable via CLI flags (`--width` and `--height`). Default values shall be used when the flags are omitted. |
| FR-03 | Each cell in the grid shall be uniquely identified by a (column, row) coordinate pair, zero-indexed from the top-left corner. |
| FR-04 | The grid shall track, for every cell, whether it is currently claimed and which robot ID holds the claim. |

### 2.2 Robots

| ID | Requirement |
|----|-------------|
| FR-05 | The service shall spawn exactly N robots, where N is configurable via a CLI flag (`--robots`). |
| FR-06 | Each robot shall run as a dedicated goroutine for its entire lifetime. |
| FR-07 | Each robot shall be assigned a unique integer ID in the range [0, N). |
| FR-08 | Each robot shall start at a distinct cell. If N exceeds the total number of cells (W × H), the service shall exit with a non-zero status code and a descriptive error message before starting any goroutine. |
| FR-09 | Each robot shall move to an adjacent cell (up, down, left, or right — no diagonals) on every movement tick. |
| FR-10 | A robot that reaches a grid boundary shall not move through it; it shall choose a different valid direction or remain stationary for that tick. |
| FR-11 | Each robot shall independently determine its next target cell. The movement strategy (e.g., random walk, goal-directed) is unspecified at this stage and is left to the System Architect to define; however, robots must eventually move rather than stay permanently stationary. |

### 2.3 Cell-Claiming Protocol

| ID | Requirement |
|----|-------------|
| FR-12 | Before entering a cell, a robot shall send a **claim request** for that cell via a dedicated coordination channel. |
| FR-13 | A central coordinator (goroutine) shall process claim requests serially and respond with a **grant** or **deny** message. |
| FR-14 | A claim request for a cell that is currently unclaimed shall always result in a **grant**. |
| FR-15 | A claim request for a cell that is currently claimed by another robot shall always result in a **deny**. |
| FR-16 | Upon receiving a **grant**, the robot shall move to the claimed cell and release the claim on its previous cell atomically from the coordinator's perspective (i.e., the release message is processed by the coordinator before the old cell is considered free to other robots). |
| FR-17 | Upon receiving a **deny**, the robot shall not move to that cell during the current tick. It shall retry or choose a different target cell on the next tick. |
| FR-18 | A robot shall release the claim on its current cell only after successfully claiming its next cell (claim-before-move semantics), ensuring no cell is ever occupied by two robots simultaneously. |
| FR-19 | The channel protocol shall use only Go's built-in channel primitives (no mutexes, sync primitives, or third-party concurrency libraries for cell coordination). Stdlib `sync` types may be used exclusively for visualization/rendering synchronization where channel-based coordination would be impractical. |

### 2.4 Visualization

| ID | Requirement |
|----|-------------|
| FR-20 | The service shall render the current state of the grid to the terminal in real time. |
| FR-21 | The render shall refresh at a configurable rate, with a default of at least 4 frames per second. The refresh rate shall be configurable via a CLI flag (`--fps`). |
| FR-22 | Each occupied cell shall display the robot's ID (or a distinguishing symbol when IDs exceed a single character). |
| FR-23 | Each empty cell shall display a distinct empty-cell character. |
| FR-24 | The visualization shall use in-place terminal overwriting (ANSI escape sequences or equivalent) rather than scrolling output, so the grid appears to animate in place. |
| FR-25 | The visualization shall display at minimum: grid state, current tick count, and number of active robots. |

### 2.5 Lifecycle and Shutdown

| ID | Requirement |
|----|-------------|
| FR-26 | The service shall intercept SIGINT (Ctrl-C) and SIGTERM signals. |
| FR-27 | Upon receiving a termination signal, the service shall: (a) stop issuing new movement ticks, (b) allow all in-flight claim/release messages to drain, (c) terminate all robot goroutines cleanly, (d) restore the terminal to its original state, and (e) exit with status code 0. |
| FR-28 | The service shall not leave orphaned goroutines after shutdown; every goroutine started by the service must have a defined termination path triggered by the shutdown sequence. |

---

## 3. Non-Functional Requirements

| ID | Requirement |
|----|-------------|
| NFR-01 | The project shall target **Go 1.22 or later**. No language features introduced after Go 1.22 shall be used unless explicitly approved. |
| NFR-02 | **Race-detector clean:** the service shall produce zero data-race reports when run under `go test -race` and `go run -race`. |
| NFR-03 | **No external runtime dependencies for coordination:** the coordination layer shall use only the Go standard library. A minimal TUI or terminal-rendering helper (e.g., `golang.org/x/term`) is permitted solely for terminal setup/teardown, provided it introduces no coordination logic. |
| NFR-04 | The service shall compile to a **single self-contained binary** with `go build`. No C libraries, CGo, or dynamic dependencies shall be required at runtime. |
| NFR-05 | The binary shall run on **Linux, macOS, and Windows** without code changes (GOOS-agnostic). Platform-specific terminal escape sequences that do not work on Windows are acceptable provided a graceful fallback or documented limitation exists. |
| NFR-06 | Memory allocations per tick shall not grow unboundedly; the service shall be suitable for running uninterrupted for at least 10 minutes on a developer laptop without observable memory growth. |
| NFR-07 | The goroutine count shall remain bounded and proportional to N; no goroutine leaks shall occur during normal operation or shutdown. |
| NFR-08 | The project shall carry an OSI-approved open-source license (already present: see `LICENSE`). |

---

## 4. Constraints and Assumptions

| ID | Constraint / Assumption |
|----|-------------------------|
| CON-01 | No Docker, Kubernetes, or container runtime is required to build or run the service. |
| CON-02 | No database or persistent storage is used; all state is in-process and ephemeral. |
| CON-03 | No real network communication occurs; all inter-goroutine messaging uses in-process Go channels. |
| CON-04 | The terminal running the service must support ANSI escape sequences for visualization. If it does not, the service may fall back to plain-text output with degraded rendering quality. |
| CON-05 | Grid dimensions and robot count are fixed at startup; dynamic resizing of the grid or changing N at runtime is out of scope. |
| CON-06 | The project is a portfolio / demonstration artifact; production-grade observability (metrics endpoints, distributed tracing) is out of scope. |

---

## 5. Out of Scope

- Control of or communication with physical or simulated robotic hardware.
- Real network protocols (TCP, UDP, gRPC, HTTP, WebSocket).
- Persistent storage or any form of database.
- Multi-node / distributed deployment.
- Authentication, authorization, or any security controls.
- GUI or web-based visualization.
- Dynamic reconfiguration at runtime (adding/removing robots, resizing grid).
- Production observability (Prometheus metrics, OpenTelemetry traces, structured logging sinks).

---

## 6. Open Questions

| ID | Question | Impact if Unresolved |
|----|----------|----------------------|
| OQ-01 | What is the default grid size (W × H)? | Needed to set `--width` / `--height` defaults and for demonstration screenshots in the README. |
| OQ-02 | What movement strategy should robots use? Random walk is assumed, but a goal-directed (BFS/A\*) strategy would better showcase algorithmic depth as a portfolio piece. | Affects FR-11 and the complexity signal to reviewers. |
| OQ-03 | Should robots wrap around the grid edges (toroidal topology) or treat edges as hard walls? | Affects FR-10 and movement logic. |
| OQ-04 | Is there a maximum value of N for benchmarking/demo purposes, or should the service handle arbitrarily large N within grid bounds? | Affects NFR-06 (memory) and visualization legibility (FR-22). |
| OQ-05 | What TUI library (if any) is acceptable? `golang.org/x/term` for raw-mode setup only, or a richer library such as `bubbletea`? | Affects NFR-03 and the "no external runtime dependencies" constraint. |
| OQ-06 | Should the tick rate (movement speed) be configurable separately from the visualization refresh rate? | Affects FR-21 and the overall CLI surface. |
| OQ-07 | Is a `go test` suite with unit tests for the coordinator and movement logic required, or is integration/demo-run testing sufficient for portfolio purposes? | Affects acceptance criteria scope. |

---

## 7. Acceptance Criteria

| ID | Criterion |
|----|-----------|
| AC-01 | Given `--robots N --width W --height H`, the service starts N robot goroutines and renders a W×H grid without panicking. |
| AC-02 | At no point during a 60-second run shall two robots occupy the same grid cell simultaneously, verifiable by an in-process assertion or test harness. |
| AC-03 | Running `go run -race .` for 30 seconds produces zero data-race reports. |
| AC-04 | Pressing Ctrl-C causes the service to exit with status code 0, the terminal cursor and state are restored, and no goroutines remain blocked (verified by a post-shutdown goroutine count check or `goleak`). |
| AC-05 | The binary produced by `go build` is a single file with no external runtime dependencies (verified by `ldd` on Linux returning "not a dynamic executable" or equivalent). |
| AC-06 | `go build ./...` and `go test ./...` complete without errors on Go 1.22+. |
| AC-07 | The grid and robot positions visibly update in the terminal at the configured FPS for the duration of the run. |

---

*Requirements are ready for the System Architect agent to design against.*
