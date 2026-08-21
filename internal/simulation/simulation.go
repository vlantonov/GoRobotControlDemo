package simulation

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/vlantonov/GoRobotControlDemo/internal/config"
	"github.com/vlantonov/GoRobotControlDemo/internal/grid"
	"github.com/vlantonov/GoRobotControlDemo/internal/robot"
	"github.com/vlantonov/GoRobotControlDemo/internal/tui"
)

// Simulation wires the grid coordinator, robots, renderer, and tick loop together.
type Simulation struct {
	cfg      config.Config
	g        *grid.Grid
	robots   []*robot.Robot
	renderer *tui.Renderer
	tickChs  []chan time.Time
}

// New validates cfg, assigns non-overlapping starting positions via Fisher-Yates
// shuffle, and allocates all channels and components.
// The caller must ensure cfg.NumRobots <= cfg.GridWidth*cfg.GridHeight.
func New(cfg config.Config) *Simulation {
	g := grid.NewGrid(cfg.GridWidth, cfg.GridHeight)

	// build a flat list of all positions and shuffle it
	positions := make([]grid.Pos, 0, cfg.GridWidth*cfg.GridHeight)
	for y := 0; y < cfg.GridHeight; y++ {
		for x := 0; x < cfg.GridWidth; x++ {
			positions = append(positions, grid.Pos{X: x, Y: y})
		}
	}
	rand.Shuffle(len(positions), func(i, j int) {
		positions[i], positions[j] = positions[j], positions[i]
	})

	robots := make([]*robot.Robot, cfg.NumRobots)
	tickChs := make([]chan time.Time, cfg.NumRobots)
	for i := 0; i < cfg.NumRobots; i++ {
		// robot IDs are 1-indexed so that 0 encodes "empty" in the cells map
		id := i + 1
		p := positions[i]
		grid.InitialPlace(g, id, p.X, p.Y)
		robots[i] = robot.New(id, p)
		tickChs[i] = make(chan time.Time, 1)
	}

	return &Simulation{
		cfg:      cfg,
		g:        g,
		robots:   robots,
		renderer: tui.NewRenderer(),
		tickChs:  tickChs,
	}
}

// Run starts all goroutines (coordinator, renderer, N robots, tick fan-out)
// and blocks until ctx is cancelled. Every goroutine is joined before Run returns.
func (s *Simulation) Run(ctx context.Context) {
	var wg sync.WaitGroup

	// coordinator — single writer to the cells map
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.g.Run(ctx)
	}()

	// renderer — reads snapshots, writes ANSI to stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.renderer.Run(ctx, s.g.SnapshotCh())
	}()

	// robot goroutines
	for i, r := range s.robots {
		r := r
		ch := s.tickChs[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Run(ctx, s.g, ch)
		}()
	}

	// tick fan-out — distributes ticks from a single Ticker to every robot
	ticker := time.NewTicker(time.Duration(s.cfg.TickMs) * time.Millisecond)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer ticker.Stop()
		for {
			select {
			case t := <-ticker.C:
				for _, ch := range s.tickChs {
					select {
					case ch <- t: // deliver tick
					default:     // robot still on last tick; skip (no backpressure)
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	<-ctx.Done()
	wg.Wait()
}
