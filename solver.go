package main

import "sort"

// Difficulty represents a game difficulty level.
type Difficulty int

const (
	Easy Difficulty = iota
	Medium
	Hard
	Custom
)

func (d Difficulty) String() string {
	switch d {
	case Easy:
		return "Easy"
	case Medium:
		return "Medium"
	case Hard:
		return "Hard"
	case Custom:
		return "Custom"
	}
	return "?"
}

// Minimum optimal-move thresholds per difficulty.
// These define the [min, max) range of BFS-optimal moves.
const (
	easyMin   = 1
	easyMax   = 40
	mediumMin = 40
	mediumMax = 80
	hardMin   = 80
	hardMax   = 9999
)

func difficultyRange(d Difficulty) (int, int) {
	switch d {
	case Easy:
		return easyMin, easyMax
	case Medium:
		return mediumMin, mediumMax
	case Hard:
		return hardMin, hardMax
	}
	return 0, 0
}

// Solve returns the minimum number of moves to reach the win state using BFS,
// or -1 if the board is unsolvable. It uses a canonical state encoding that
// treats same-kind pieces as interchangeable for efficient state deduplication.
func Solve(b *Board) int {
	type state struct {
		pieces []Piece
		depth  int
	}

	initial := canonicalize(b.Pieces)
	key := encodeState(initial)

	visited := map[string]bool{key: true}
	queue := []state{{pieces: initial, depth: 0}}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		// Check win.
		for _, p := range cur.pieces {
			if p.Kind == Large && p.X == 1 && p.Y == 3 {
				return cur.depth
			}
		}

		// Build occupancy grid for this state.
		var grid [BoardW][BoardH]int
		for x := range BoardW {
			for y := range BoardH {
				grid[x][y] = -1
			}
		}
		for i, p := range cur.pieces {
			for _, c := range p.Cells() {
				grid[c[0]][c[1]] = i
			}
		}

		// Try moving each piece in each direction.
		dirs := [4]Direction{Up, Down, Left, Right}
		for i, p := range cur.pieces {
			for _, dir := range dirs {
				dx, dy := dirDelta(dir)
				canMove := true
				for _, c := range p.Cells() {
					nx, ny := c[0]+dx, c[1]+dy
					if nx < 0 || nx >= BoardW || ny < 0 || ny >= BoardH {
						canMove = false
						break
					}
					occ := grid[nx][ny]
					if occ != -1 && occ != i {
						canMove = false
						break
					}
				}
				if !canMove {
					continue
				}

				// Apply move.
				newPieces := make([]Piece, len(cur.pieces))
				copy(newPieces, cur.pieces)
				newPieces[i] = Piece{Kind: p.Kind, X: p.X + dx, Y: p.Y + dy}
				newPieces = canonicalize(newPieces)

				k := encodeState(newPieces)
				if !visited[k] {
					visited[k] = true
					queue = append(queue, state{pieces: newPieces, depth: cur.depth + 1})
				}
			}
		}
	}

	return -1 // unsolvable
}

// canonicalize sorts pieces by (kind, x, y) so that interchangeable
// same-kind pieces always appear in a deterministic order.
func canonicalize(pieces []Piece) []Piece {
	sorted := make([]Piece, len(pieces))
	copy(sorted, pieces)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Kind != sorted[j].Kind {
			return sorted[i].Kind < sorted[j].Kind
		}
		if sorted[i].X != sorted[j].X {
			return sorted[i].X < sorted[j].X
		}
		return sorted[i].Y < sorted[j].Y
	})
	return sorted
}

// encodeState produces a compact string key for a set of (already canonical) pieces.
func encodeState(pieces []Piece) string {
	// Each piece encoded as 3 bytes: kind, x, y.
	buf := make([]byte, len(pieces)*3)
	for i, p := range pieces {
		buf[i*3] = byte(p.Kind)
		buf[i*3+1] = byte(p.X)
		buf[i*3+2] = byte(p.Y)
	}
	return string(buf)
}

// Hint represents a suggested next move: which piece to move and in which direction.
type Hint struct {
	PieceIndex int
	Dir        Direction
}

// SolveNextMove finds the optimal next move from the current board state using BFS.
// It returns the first move on an optimal solution path, mapped back to the actual
// piece indices in the given board. Returns nil if already won or unsolvable.
func SolveNextMove(b *Board) *Hint {
	if b.IsWon() {
		return nil
	}

	type entry struct {
		pieces       []Piece
		firstMoveKey string // canonical key of the depth-1 ancestor on this path
	}

	initial := canonicalize(b.Pieces)
	initKey := encodeState(initial)
	visited := map[string]bool{initKey: true}
	dirs := [4]Direction{Up, Down, Left, Right}

	// Build occupancy for the initial (canonical) state.
	var grid0 [BoardW][BoardH]int
	for x := range BoardW {
		for y := range BoardH {
			grid0[x][y] = -1
		}
	}
	for i, p := range initial {
		for _, c := range p.Cells() {
			grid0[c[0]][c[1]] = i
		}
	}

	// Expand the initial state to generate depth-1 states, each tagged with its own key.
	var queue []entry
	for i, p := range initial {
		for _, dir := range dirs {
			dx, dy := dirDelta(dir)
			canMove := true
			for _, c := range p.Cells() {
				nx, ny := c[0]+dx, c[1]+dy
				if nx < 0 || nx >= BoardW || ny < 0 || ny >= BoardH {
					canMove = false
					break
				}
				if occ := grid0[nx][ny]; occ != -1 && occ != i {
					canMove = false
					break
				}
			}
			if !canMove {
				continue
			}

			newPieces := make([]Piece, len(initial))
			copy(newPieces, initial)
			newPieces[i] = Piece{Kind: p.Kind, X: p.X + dx, Y: p.Y + dy}
			newPieces = canonicalize(newPieces)
			k := encodeState(newPieces)

			if visited[k] {
				continue
			}
			visited[k] = true

			// Check if this single move wins.
			for _, pp := range newPieces {
				if pp.Kind == Large && pp.X == 1 && pp.Y == 3 {
					return mapToOriginalBoard(b, k)
				}
			}
			queue = append(queue, entry{pieces: newPieces, firstMoveKey: k})
		}
	}

	// BFS from depth-1 onward; each node inherits its depth-1 ancestor's key.
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		var grid [BoardW][BoardH]int
		for x := range BoardW {
			for y := range BoardH {
				grid[x][y] = -1
			}
		}
		for i, p := range cur.pieces {
			for _, c := range p.Cells() {
				grid[c[0]][c[1]] = i
			}
		}

		for i, p := range cur.pieces {
			for _, dir := range dirs {
				dx, dy := dirDelta(dir)
				canMove := true
				for _, c := range p.Cells() {
					nx, ny := c[0]+dx, c[1]+dy
					if nx < 0 || nx >= BoardW || ny < 0 || ny >= BoardH {
						canMove = false
						break
					}
					if occ := grid[nx][ny]; occ != -1 && occ != i {
						canMove = false
						break
					}
				}
				if !canMove {
					continue
				}

				newPieces := make([]Piece, len(cur.pieces))
				copy(newPieces, cur.pieces)
				newPieces[i] = Piece{Kind: p.Kind, X: p.X + dx, Y: p.Y + dy}
				newPieces = canonicalize(newPieces)
				k := encodeState(newPieces)

				if visited[k] {
					continue
				}
				visited[k] = true

				for _, pp := range newPieces {
					if pp.Kind == Large && pp.X == 1 && pp.Y == 3 {
						return mapToOriginalBoard(b, cur.firstMoveKey)
					}
				}
				queue = append(queue, entry{pieces: newPieces, firstMoveKey: cur.firstMoveKey})
			}
		}
	}

	return nil // unsolvable
}

// mapToOriginalBoard finds the actual piece index and direction in the original
// board that produces the given canonical depth-1 state after one move.
func mapToOriginalBoard(b *Board, depth1Key string) *Hint {
	for i := range b.Pieces {
		for _, dir := range [4]Direction{Up, Down, Left, Right} {
			if !b.CanMove(i, dir) {
				continue
			}
			dx, dy := dirDelta(dir)
			newPieces := make([]Piece, len(b.Pieces))
			copy(newPieces, b.Pieces)
			newPieces[i] = Piece{
				Kind: b.Pieces[i].Kind,
				X:    b.Pieces[i].X + dx,
				Y:    b.Pieces[i].Y + dy,
			}
			if encodeState(canonicalize(newPieces)) == depth1Key {
				return &Hint{PieceIndex: i, Dir: dir}
			}
		}
	}
	return nil
}
