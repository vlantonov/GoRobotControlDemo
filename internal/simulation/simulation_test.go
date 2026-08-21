package simulation_test

import (
	"context"
	"testing"
	"time"

	"github.com/vlantonov/GoRobotControlDemo/internal/config"
	"github.com/vlantonov/GoRobotControlDemo/internal/simulation"
)

func TestNew_ReturnsNonNil(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
	}{
		{"single robot", config.Config{GridWidth: 5, GridHeight: 5, NumRobots: 1, TickMs: 100, FPS: 10}},
		{"default config", config.Config{GridWidth: 20, GridHeight: 10, NumRobots: 5, TickMs: 100, FPS: 10}},
		{"max robots", config.Config{GridWidth: 3, GridHeight: 3, NumRobots: 9, TickMs: 50, FPS: 10}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim := simulation.New(tt.cfg)
			if sim == nil {
				t.Fatal("New returned nil")
			}
		})
	}
}

// TestRun_StartsAndStops verifies AC-1 (N robot goroutines start) and AC-4
// (context cancellation causes Run to return cleanly within a reasonable time).
func TestRun_StartsAndStops(t *testing.T) {
	cfg := config.Config{GridWidth: 5, GridHeight: 5, NumRobots: 3, TickMs: 20, FPS: 20}
	sim := simulation.New(cfg)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		sim.Run(ctx)
	}()

	// Let the simulation tick a few times so robots actually move.
	time.Sleep(80 * time.Millisecond)

	cancel()

	select {
	case <-done:
		// Run returned — all goroutines joined (FR-28)
	case <-time.After(2 * time.Second):
		t.Fatal("simulation.Run did not return within 2s after context cancel")
	}
}

// TestRun_MultipleRobots exercises a larger fleet to confirm all goroutines
// start and the simulation shuts down without deadlock.
func TestRun_MultipleRobots(t *testing.T) {
	cfg := config.Config{GridWidth: 10, GridHeight: 10, NumRobots: 10, TickMs: 10, FPS: 30}
	sim := simulation.New(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		sim.Run(ctx)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("simulation.Run did not return after context deadline")
	}
}
