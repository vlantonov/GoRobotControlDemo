package robot

import (
	"context"
	"math/rand"
	"time"

	"github.com/vlantonov/GoRobotControlDemo/internal/grid"
)

// Robot is a warehouse robot that runs as an independent goroutine.
type Robot struct {
	ID  int
	pos grid.Pos
}

// New constructs a Robot with the given ID and starting position.
func New(id int, startPos grid.Pos) *Robot {
	return &Robot{ID: id, pos: startPos}
}

// directions holds the four orthogonal offsets used for random-walk movement.
var directions = [4]grid.Pos{
	{X: 0, Y: -1}, // north
	{X: 0, Y: 1},  // south
	{X: 1, Y: 0},  // east
	{X: -1, Y: 0}, // west
}

// Run is the robot goroutine body. It blocks until ctx is done.
// On each tick it shuffles the four cardinal directions and tries each in turn,
// stopping on the first grant. If all are denied or out-of-bounds, it stays put.
func (r *Robot) Run(ctx context.Context, g *grid.Grid, tickCh <-chan time.Time) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-tickCh:
			r.tryMove(ctx, g)
		}
	}
}

// tryMove attempts to move to one of the four orthogonal neighbours chosen in
// random order. It returns after the first successful grant or after exhausting
// all valid neighbours (robot stays put for this tick, per FR-17).
func (r *Robot) tryMove(ctx context.Context, g *grid.Grid) {
	dirs := directions
	rand.Shuffle(len(dirs), func(i, j int) { dirs[i], dirs[j] = dirs[j], dirs[i] })

	for _, d := range dirs {
		next := grid.Pos{X: r.pos.X + d.X, Y: r.pos.Y + d.Y}
		// skip out-of-bounds neighbours (FR-10)
		if next.X < 0 || next.X >= g.Width() || next.Y < 0 || next.Y >= g.Height() {
			continue
		}

		respCh := make(chan bool, 1)
		select {
		case g.ClaimCh() <- grid.ClaimRequest{
			RobotID: r.ID,
			From:    r.pos,
			To:      next,
			ReplyTo: respCh,
		}:
		case <-ctx.Done():
			return
		}

		select {
		case granted := <-respCh:
			if granted {
				r.pos = next
				return
			}
		case <-ctx.Done():
			return
		}
	}
	// all neighbours denied — stay put until next tick (FR-17)
}
