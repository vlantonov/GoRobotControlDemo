package grid_test

import (
	"context"
	"testing"
	"time"

	"github.com/vlantonov/GoRobotControlDemo/internal/grid"
)

func TestNewGrid(t *testing.T) {
	tests := []struct {
		name  string
		w, h  int
		wantW int
		wantH int
	}{
		{"small", 3, 3, 3, 3},
		{"rectangular", 10, 5, 10, 5},
		{"wide", 20, 10, 20, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := grid.NewGrid(tt.w, tt.h)
			if g.Width() != tt.wantW {
				t.Errorf("Width = %d, want %d", g.Width(), tt.wantW)
			}
			if g.Height() != tt.wantH {
				t.Errorf("Height = %d, want %d", g.Height(), tt.wantH)
			}
		})
	}
}

func TestInitialSnapshot(t *testing.T) {
	g := grid.NewGrid(3, 3)
	grid.InitialPlace(g, 1, 0, 0)
	grid.InitialPlace(g, 2, 2, 2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go g.Run(ctx)

	select {
	case snap := <-g.SnapshotCh():
		if snap.Width != 3 || snap.Height != 3 {
			t.Errorf("snapshot dimensions = %dx%d, want 3x3", snap.Width, snap.Height)
		}
		if snap.Cells[0][0] != 1 {
			t.Errorf("Cells[0][0] = %d, want 1", snap.Cells[0][0])
		}
		if snap.Cells[2][2] != 2 {
			t.Errorf("Cells[2][2] = %d, want 2", snap.Cells[2][2])
		}
		if snap.ActiveRobots != 2 {
			t.Errorf("ActiveRobots = %d, want 2", snap.ActiveRobots)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for initial snapshot")
	}
}

func TestRun_GrantFreeCell(t *testing.T) {
	g := grid.NewGrid(5, 5)
	grid.InitialPlace(g, 1, 0, 0)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go g.Run(ctx)

	// drain initial snapshot so the post-move snapshot can fit
	select {
	case <-g.SnapshotCh():
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for initial snapshot")
	}

	respCh := make(chan bool, 1)
	g.ClaimCh() <- grid.ClaimRequest{
		RobotID: 1,
		From:    grid.Pos{X: 0, Y: 0},
		To:      grid.Pos{X: 1, Y: 0},
		ReplyTo: respCh,
	}

	select {
	case granted := <-respCh:
		if !granted {
			t.Error("expected grant for free cell, got deny")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for coordinator response")
	}
}

func TestRun_DenyOccupiedCell(t *testing.T) {
	g := grid.NewGrid(5, 5)
	grid.InitialPlace(g, 1, 1, 0)
	grid.InitialPlace(g, 2, 0, 0)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go g.Run(ctx)

	respCh := make(chan bool, 1)
	// robot 2 tries to enter robot 1's cell
	g.ClaimCh() <- grid.ClaimRequest{
		RobotID: 2,
		From:    grid.Pos{X: 0, Y: 0},
		To:      grid.Pos{X: 1, Y: 0},
		ReplyTo: respCh,
	}

	select {
	case granted := <-respCh:
		if granted {
			t.Error("expected deny for occupied cell, got grant")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for coordinator response")
	}
}

func TestRun_TickIncrements(t *testing.T) {
	g := grid.NewGrid(5, 5)
	grid.InitialPlace(g, 1, 0, 0)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go g.Run(ctx)

	// drain initial snapshot (tick == 0)
	select {
	case snap := <-g.SnapshotCh():
		if snap.Tick != 0 {
			t.Errorf("initial Tick = %d, want 0", snap.Tick)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for initial snapshot")
	}

	respCh := make(chan bool, 1)
	g.ClaimCh() <- grid.ClaimRequest{
		RobotID: 1,
		From:    grid.Pos{X: 0, Y: 0},
		To:      grid.Pos{X: 1, Y: 0},
		ReplyTo: respCh,
	}
	<-respCh

	select {
	case snap := <-g.SnapshotCh():
		if snap.Tick != 1 {
			t.Errorf("Tick after move = %d, want 1", snap.Tick)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for post-move snapshot")
	}
}

func TestRun_ExitsOnCancel(t *testing.T) {
	g := grid.NewGrid(3, 3)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		g.Run(ctx)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("coordinator did not exit after cancel")
	}
}
