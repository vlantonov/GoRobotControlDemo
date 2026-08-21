package grid

import "context"

// Pos is a zero-indexed (column, row) coordinate.
type Pos struct {
	X, Y int
}

// ClaimRequest is sent by a robot to the coordinator to atomically release
// its current cell (From) and claim a new cell (To).
type ClaimRequest struct {
	RobotID int
	From    Pos
	To      Pos
	// ReplyTo must be created with make(chan bool, 1) so the coordinator
	// can respond without blocking if the robot has already cancelled.
	ReplyTo chan bool
}

// GridSnapshot is an immutable copy of the grid state at one instant.
// Cells[y][x] holds 0 (empty) or the robot ID of the current occupant.
type GridSnapshot struct {
	Width        int
	Height       int
	Cells        [][]int
	Tick         uint64
	ActiveRobots int
}

// Grid owns the cell-occupation map and the coordination channels.
// The only goroutine that writes to cells is the one running Run.
type Grid struct {
	width      int
	height     int
	cells      map[Pos]int
	claimCh    chan ClaimRequest
	snapshotCh chan GridSnapshot
	tick       uint64
}

// NewGrid constructs a Grid with the given dimensions.
// The claim channel is buffered to width×height to absorb a full round
// of simultaneous requests without blocking robots.
func NewGrid(w, h int) *Grid {
	return &Grid{
		width:      w,
		height:     h,
		cells:      make(map[Pos]int),
		claimCh:    make(chan ClaimRequest, w*h),
		snapshotCh: make(chan GridSnapshot, 1),
	}
}

// Width returns the number of columns.
func (g *Grid) Width() int { return g.width }

// Height returns the number of rows.
func (g *Grid) Height() int { return g.height }

// ClaimCh returns the send-only end of the claim channel.
func (g *Grid) ClaimCh() chan<- ClaimRequest { return g.claimCh }

// SnapshotCh returns the receive-only end of the snapshot channel.
func (g *Grid) SnapshotCh() <-chan GridSnapshot { return g.snapshotCh }

// InitialPlace records robotID at position (x, y) before Run is started.
// Must not be called concurrently with Run.
func InitialPlace(g *Grid, robotID, x, y int) {
	g.cells[Pos{X: x, Y: y}] = robotID
}

func (g *Grid) buildSnapshot() GridSnapshot {
	cells := make([][]int, g.height)
	for row := 0; row < g.height; row++ {
		cells[row] = make([]int, g.width)
		for col := 0; col < g.width; col++ {
			cells[row][col] = g.cells[Pos{X: col, Y: row}]
		}
	}
	return GridSnapshot{
		Width:        g.width,
		Height:       g.height,
		Cells:        cells,
		Tick:         g.tick,
		ActiveRobots: len(g.cells),
	}
}

func (g *Grid) pushSnapshot() {
	snap := g.buildSnapshot()
	select {
	case g.snapshotCh <- snap:
	default: // renderer busy; latest snapshot already waiting
	}
}

// Run is the coordinator goroutine. It processes ClaimRequests serially until
// ctx is cancelled, then drains any buffered requests (denying all) before returning.
func (g *Grid) Run(ctx context.Context) {
	g.pushSnapshot() // send initial state to renderer
	for {
		select {
		case req := <-g.claimCh:
			if g.cells[req.To] == 0 {
				// free cell: atomic release-old + claim-new
				delete(g.cells, req.From)
				g.cells[req.To] = req.RobotID
				g.tick++
				req.ReplyTo <- true
				g.pushSnapshot()
			} else {
				req.ReplyTo <- false
			}
		case <-ctx.Done():
			// drain in-flight requests so no robot blocks forever on ReplyTo
			for {
				select {
				case req := <-g.claimCh:
					req.ReplyTo <- false
				default:
					return
				}
			}
		}
	}
}
