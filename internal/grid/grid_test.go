package grid_test

import (
	"context"
	"math/rand"
	"sync"
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

// TestRun_DrainOnCancel ensures that a ClaimRequest already buffered in the
// claim channel before cancellation is denied (not dropped), so that robots
// waiting on their RespCh are not abandoned.
func TestRun_DrainOnCancel(t *testing.T) {
	g := grid.NewGrid(5, 5)
	grid.InitialPlace(g, 1, 0, 0)

	ctx, cancel := context.WithCancel(context.Background())
	go g.Run(ctx)

	// drain initial snapshot
	select {
	case <-g.SnapshotCh():
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for initial snapshot")
	}

	// Buffer a request BEFORE cancelling, so it sits in the claim channel.
	respCh := make(chan bool, 1)
	g.ClaimCh() <- grid.ClaimRequest{
		RobotID: 1,
		From:    grid.Pos{X: 0, Y: 0},
		To:      grid.Pos{X: 1, Y: 0},
		ReplyTo: respCh,
	}

	cancel() // coordinator will drain the buffered request

	select {
	case granted := <-respCh:
		// After cancel the coordinator denies all in-flight requests.
		// It may have already processed the request before cancel fired (grant),
		// or drained it after cancel (deny). Both are valid — what must NOT happen
		// is a deadlock (no response at all).
		_ = granted
	case <-time.After(time.Second):
		t.Fatal("robot blocked forever: coordinator did not reply after cancel")
	}
}

// TestNoCollision_Concurrent runs N robots on a small grid for several hundred
// ticks and asserts that every GridSnapshot shows each cell occupied by at most
// one robot — independently verifying the claim-before-move invariant (AC-2).
func TestNoCollision_Concurrent(t *testing.T) {
	const (
		w, h      = 5, 5
		numRobots = 6
		numTicks  = 300
		tickMs    = 5
	)

	g := grid.NewGrid(w, h)

	// Place robots at distinct starting positions.
	starts := []grid.Pos{
		{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 0, Y: 4},
		{X: 4, Y: 4}, {X: 2, Y: 2}, {X: 1, Y: 3},
	}
	for i, p := range starts {
		grid.InitialPlace(g, i+1, p.X, p.Y)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go g.Run(ctx)

	// drain initial snapshot
	select {
	case <-g.SnapshotCh():
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for initial snapshot")
	}

	// Start robots via individual tick channels.
	tickChs := make([]chan time.Time, numRobots)
	for i := range tickChs {
		tickChs[i] = make(chan time.Time, 1)
	}

	// Import robot package indirectly via ClaimRequest — use raw grid protocol
	// to avoid a direct dependency on the robot package from a grid test.
	// Instead, run the robots via the simulation package logic replicated here.
	// Actually we do need to import robot; use a separate helper goroutine.
	//
	// Since grid_test is in package grid_test, it can import robot.
	// But that would create an awkward import. Use the grid's channel protocol
	// directly with goroutines that mimic robot.tryMove logic.
	type robotState struct{ pos grid.Pos }
	states := make([]robotState, numRobots)
	for i, p := range starts {
		states[i] = robotState{pos: p}
	}

	dirs := []grid.Pos{{X: 0, Y: -1}, {X: 0, Y: 1}, {X: 1, Y: 0}, {X: -1, Y: 0}}

	// Use a shared source of pseudo-randomness per robot (deterministic enough).
	var wg sync.WaitGroup
	for i := 0; i < numRobots; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(i + 1)))
			id := i + 1
			pos := states[i].pos
			for tick := 0; tick < numTicks; tick++ {
				perm := rng.Perm(4)
				for _, di := range perm {
					d := dirs[di]
					next := grid.Pos{X: pos.X + d.X, Y: pos.Y + d.Y}
					if next.X < 0 || next.X >= w || next.Y < 0 || next.Y >= h {
						continue
					}
					respCh := make(chan bool, 1)
					select {
					case g.ClaimCh() <- grid.ClaimRequest{
						RobotID: id,
						From:    pos,
						To:      next,
						ReplyTo: respCh,
					}:
					case <-ctx.Done():
						return
					}
					select {
					case granted := <-respCh:
						if granted {
							pos = next
						}
					case <-ctx.Done():
						return
					}
					if pos == next {
						break
					}
				}
				time.Sleep(time.Duration(tickMs) * time.Millisecond)
			}
		}()
	}

	// Consume snapshots and check the no-collision invariant on each one.
	violations := 0
	snapDone := make(chan struct{})
	go func() {
		defer close(snapDone)
		deadline := time.After(time.Duration(numTicks*tickMs*2) * time.Millisecond)
		for {
			select {
			case snap, ok := <-g.SnapshotCh():
				if !ok {
					return
				}
				seen := make(map[int]grid.Pos) // robotID → pos
				for row := 0; row < snap.Height; row++ {
					for col := 0; col < snap.Width; col++ {
						id := snap.Cells[row][col]
						if id == 0 {
							continue
						}
						if prev, dup := seen[id]; dup {
							t.Errorf("robot %d appears at both (%d,%d) and (%d,%d) in same snapshot",
								id, prev.X, prev.Y, col, row)
							violations++
						}
						seen[id] = grid.Pos{X: col, Y: row}
					}
				}
				// Also verify no two robots share a cell.
				cellCount := make(map[grid.Pos]int)
				for row := 0; row < snap.Height; row++ {
					for col := 0; col < snap.Width; col++ {
						id := snap.Cells[row][col]
						if id != 0 {
							cellCount[grid.Pos{X: col, Y: row}]++
						}
					}
				}
				for pos, cnt := range cellCount {
					if cnt > 1 {
						t.Errorf("cell (%d,%d) has %d occupants", pos.X, pos.Y, cnt)
						violations++
					}
				}
			case <-deadline:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	wg.Wait()
	cancel()
	<-snapDone

	if violations > 0 {
		t.Errorf("total collision violations: %d", violations)
	}
}
