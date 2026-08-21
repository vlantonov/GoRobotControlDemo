package simulation_test

import (
	"testing"

	"github.com/vlantonov/GoRobotControlDemo/internal/config"
	"github.com/vlantonov/GoRobotControlDemo/internal/simulation"
)

func TestNew_ReturnsNonNil(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
	}{
		{"single robot", config.Config{GridWidth: 5, GridHeight: 5, NumRobots: 1, TickMs: 100}},
		{"default config", config.Config{GridWidth: 20, GridHeight: 10, NumRobots: 5, TickMs: 100}},
		{"max robots", config.Config{GridWidth: 3, GridHeight: 3, NumRobots: 9, TickMs: 50}},
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
