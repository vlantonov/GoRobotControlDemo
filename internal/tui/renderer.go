package tui

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/vlantonov/GoRobotControlDemo/internal/grid"
)

// Renderer renders grid snapshots to an ANSI terminal at a configurable FPS.
type Renderer struct {
	w   io.Writer
	fps int
}

// NewRenderer returns a Renderer that writes to stdout at the given frames-per-second.
func NewRenderer(fps int) *Renderer {
	return &Renderer{w: os.Stdout, fps: fps}
}

// NewRendererWriter returns a Renderer that writes to w at the given FPS.
// Useful in tests to capture output without a real terminal.
func NewRendererWriter(w io.Writer, fps int) *Renderer {
	return &Renderer{w: w, fps: fps}
}

// Run reads from snapCh and redraws the terminal at the configured FPS until
// ctx is cancelled. ANSI escape sequences are used for in-place overwriting.
func (r *Renderer) Run(ctx context.Context, snapCh <-chan grid.GridSnapshot) {
	ticker := time.NewTicker(time.Second / time.Duration(r.fps))
	defer ticker.Stop()
	fmt.Fprint(r.w, "\x1b[?25l\x1b[s") // hide cursor; save position
	defer r.restore()
	var latest *grid.GridSnapshot
	for {
		select {
		case <-ctx.Done():
			return
		case snap, ok := <-snapCh:
			if !ok {
				return
			}
			latest = &snap
		case <-ticker.C:
			if latest != nil {
				r.draw(*latest)
				latest = nil
			}
		}
	}
}

func (r *Renderer) restore() {
	fmt.Fprint(r.w, "\x1b[?25h\x1b[0m\n") // show cursor; reset colour
}

// ansiColors maps (robotID-1) mod 7 to ANSI foreground colour codes.
var ansiColors = [7]string{"31", "32", "33", "34", "35", "36", "37"}

func (r *Renderer) draw(snap grid.GridSnapshot) {
	var sb strings.Builder
	sb.WriteString("\x1b[u") // restore cursor to saved position

	for row := 0; row < snap.Height; row++ {
		for col := 0; col < snap.Width; col++ {
			id := snap.Cells[row][col]
			if id == 0 {
				sb.WriteByte('.')
			} else {
				color := ansiColors[(id-1)%len(ansiColors)]
				var ch byte
				if id < 10 {
					ch = byte('0' + id)
				} else {
					ch = '@' // distinguishing symbol for multi-digit IDs (FR-22)
				}
				fmt.Fprintf(&sb, "\x1b[%sm%c\x1b[0m", color, ch)
			}
		}
		sb.WriteByte('\n')
	}

	fmt.Fprintf(&sb, "Tick: %-10d  Robots: %d\n", snap.Tick, snap.ActiveRobots)
	fmt.Fprint(r.w, sb.String())
}
