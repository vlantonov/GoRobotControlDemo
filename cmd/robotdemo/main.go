package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/vlantonov/GoRobotControlDemo/internal/config"
	"github.com/vlantonov/GoRobotControlDemo/internal/simulation"
)

func main() {
	cfg := config.Parse()

	if cfg.GridWidth <= 0 || cfg.GridHeight <= 0 {
		fmt.Fprintln(os.Stderr, "error: --width and --height must be positive integers")
		os.Exit(1)
	}
	if cfg.NumRobots <= 0 {
		fmt.Fprintln(os.Stderr, "error: --robots must be a positive integer")
		os.Exit(1)
	}
	if cfg.TickMs <= 0 {
		fmt.Fprintln(os.Stderr, "error: --tick-ms must be a positive integer")
		os.Exit(1)
	}
	if cfg.NumRobots > cfg.GridWidth*cfg.GridHeight {
		fmt.Fprintf(os.Stderr,
			"error: --robots (%d) exceeds total grid cells (%d×%d = %d)\n",
			cfg.NumRobots, cfg.GridWidth, cfg.GridHeight, cfg.GridWidth*cfg.GridHeight)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	simulation.New(cfg).Run(ctx)
}
