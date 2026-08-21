package robot_test

import (
	"context"
	"testing"
	"time"

	"github.com/vlantonov/GoRobotControlDemo/internal/grid"
	"github.com/vlantonov/GoRobotControlDemo/internal/robot"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name  string
		id    int
		start grid.Pos
	}{
		{"zero pos", 1, grid.Pos{X: 0, Y: 0}},
		{"mid grid", 3, grid.Pos{X: 5, Y: 4}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := robot.New(tt.id, tt.start)
			if r == nil {
				t.Fatal("New returned nil")
			}
			if r.ID != tt.id {
				t.Errorf("ID = %d, want %d", r.ID, tt.id)
			}
		})
	}
}

func TestRun_ExitsOnCancel(t *testing.T) {
	g := grid.NewGrid(5, 5)
	grid.InitialPlace(g, 1, 2, 2)

	ctx, cancel := context.WithCancel(context.Background())
	go g.Run(ctx)

	tickCh := make(chan time.Time, 1)
	r := robot.New(1, grid.Pos{X: 2, Y: 2})

	done := make(chan struct{})
	go func() {
		defer close(done)
		r.Run(ctx, g, tickCh)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("robot did not exit within 1s of cancel")
	}
}

func TestRun_MovesOnTick(t *testing.T) {
	g := grid.NewGrid(5, 5)
	grid.InitialPlace(g, 1, 2, 2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go g.Run(ctx)

	// drain initial snapshot so the movement snapshot can occupy the buffer
	select {
	case <-g.SnapshotCh():
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for initial snapshot")
	}

	tickCh := make(chan time.Time, 1)
	r := robot.New(1, grid.Pos{X: 2, Y: 2})
	go r.Run(ctx, g, tickCh)

	tickCh <- time.Now()

	select {
	case snap := <-g.SnapshotCh():
		// robot must have vacated (2,2)
		if snap.Cells[2][2] != 0 {
			t.Error("robot did not leave starting cell after tick")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for movement snapshot")
	}
}

func TestRun_StaysPutWhenSurrounded(t *testing.T) {
	// Place robot 1 at (1,1) and surround all four neighbours.
	g := grid.NewGrid(3, 3)
	grid.InitialPlace(g, 1, 1, 1) // centre
	grid.InitialPlace(g, 2, 1, 0) // north
	grid.InitialPlace(g, 3, 1, 2) // south
	grid.InitialPlace(g, 4, 2, 1) // east
	grid.InitialPlace(g, 5, 0, 1) // west

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go g.Run(ctx)

	// drain initial snapshot
	select {
	case <-g.SnapshotCh():
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for initial snapshot")
	}

	tickCh := make(chan time.Time, 1)
	r := robot.New(1, grid.Pos{X: 1, Y: 1})

	done := make(chan struct{})
	go func() {
		defer close(done)
		r.Run(ctx, g, tickCh)
	}()

	tickCh <- time.Now()
	// give robot time to attempt all directions and stay put
	time.Sleep(50 * time.Millisecond)

	// no snapshot should appear (all denies, no grant → no snapshot push)
	select {
	case <-g.SnapshotCh():
		// a snapshot would mean a grant happened — unexpected
		t.Error("unexpected snapshot: surrounded robot should not move")
	default:
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("robot goroutine did not exit after cancel")
	}
}

// TestTryMove_CancelWhileWaitingForReply exercises the ctx.Done() branch inside
// tryMove that fires while the robot is waiting on respCh. We achieve this by
// NOT starting the coordinator (so no one ever writes to the response channel),
// then cancelling the context to unblock the robot.
func TestTryMove_CancelWhileWaitingForReply(t *testing.T) {
	// 2×1 grid: robot at (0,0) with one valid neighbour at (1,0).
	// claimCh buffer = 2×1 = 2, so the send succeeds without a reader.
	g := grid.NewGrid(2, 1)
	grid.InitialPlace(g, 1, 0, 0)
	// grid.Run is NOT started — nobody replies to respCh.

	ctx, cancel := context.WithCancel(context.Background())

	tickCh := make(chan time.Time, 1)
	r := robot.New(1, grid.Pos{X: 0, Y: 0})

	done := make(chan struct{})
	go func() {
		defer close(done)
		r.Run(ctx, g, tickCh)
	}()

	// Send a tick so the robot enters tryMove, enqueues its ClaimRequest, and
	// then blocks on respCh waiting for a reply that will never arrive.
	tickCh <- time.Now()
	// Give the robot time to reach the respCh select.
	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("robot did not exit after cancel while waiting for reply")
	}
}
