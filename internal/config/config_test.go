package config_test

import (
	"testing"

	"github.com/vlantonov/GoRobotControlDemo/internal/config"
)

func TestConfigDefaults(t *testing.T) {
	tests := []struct {
		name  string
		field int
		want  int
	}{
		{"GridWidth", config.Config{GridWidth: 20}.GridWidth, 20},
		{"GridHeight", config.Config{GridHeight: 10}.GridHeight, 10},
		{"NumRobots", config.Config{NumRobots: 5}.NumRobots, 5},
		{"TickMs", config.Config{TickMs: 100}.TickMs, 100},
		{"FPS", config.Config{FPS: 10}.FPS, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.field != tt.want {
				t.Errorf("got %d, want %d", tt.field, tt.want)
			}
		})
	}
}

func TestConfigZeroValue(t *testing.T) {
	var cfg config.Config
	if cfg.GridWidth != 0 || cfg.GridHeight != 0 || cfg.NumRobots != 0 || cfg.TickMs != 0 || cfg.FPS != 0 {
		t.Error("zero-value Config should have all-zero fields")
	}
}
