package tui_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/vlantonov/GoRobotControlDemo/internal/grid"
	"github.com/vlantonov/GoRobotControlDemo/internal/tui"
)

func TestRenderer_ExitsOnCancel(t *testing.T) {
	var buf bytes.Buffer
	r := tui.NewRendererWriter(&buf, 30)
	snapCh := make(chan grid.GridSnapshot, 1)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		r.Run(ctx, snapCh)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("renderer did not exit after context cancel")
	}
}

func TestRenderer_ExitsOnClosedChannel(t *testing.T) {
	var buf bytes.Buffer
	r := tui.NewRendererWriter(&buf, 30)
	snapCh := make(chan grid.GridSnapshot)

	done := make(chan struct{})
	go func() {
		defer close(done)
		r.Run(context.Background(), snapCh)
	}()

	close(snapCh)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("renderer did not exit after snapshot channel closed")
	}
}

func TestRenderer_DrawsSnapshot(t *testing.T) {
	var buf bytes.Buffer
	r := tui.NewRendererWriter(&buf, 100) // fast FPS so the ticker fires quickly

	snapCh := make(chan grid.GridSnapshot, 1)

	cells := [][]int{
		{1, 0, 0},
		{0, 2, 0},
		{0, 0, 0},
	}
	snapCh <- grid.GridSnapshot{Width: 3, Height: 3, Cells: cells, Tick: 7, ActiveRobots: 2}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		r.Run(ctx, snapCh)
	}()

	// Wait long enough for the FPS ticker to fire and draw.
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done // wait for renderer goroutine to stop before touching buf

	out := buf.String()
	if !strings.Contains(out, "Tick:") {
		t.Errorf("rendered output missing 'Tick:' line; got: %q", out)
	}
	if !strings.Contains(out, "Robots:") {
		t.Errorf("rendered output missing 'Robots:' line; got: %q", out)
	}
}
