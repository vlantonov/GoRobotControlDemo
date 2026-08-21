package config

import "flag"

// Config holds all runtime parameters resolved from CLI flags.
type Config struct {
	GridWidth  int
	GridHeight int
	NumRobots  int
	TickMs     int
	FPS        int
}

// Parse parses os.Args flags and returns a Config with defaults applied.
func Parse() Config {
	var cfg Config
	flag.IntVar(&cfg.GridWidth, "width", 20, "grid width in cells")
	flag.IntVar(&cfg.GridHeight, "height", 10, "grid height in cells")
	flag.IntVar(&cfg.NumRobots, "robots", 5, "number of robots")
	flag.IntVar(&cfg.TickMs, "tick-ms", 100, "movement tick in milliseconds")
	flag.IntVar(&cfg.FPS, "fps", 10, "render frames per second")
	flag.Parse()
	return cfg
}
