package main

import (
	"math/rand"
)

// Board dimensions.
const (
	BoardW = 4
	BoardH = 5
)

// Direction represents a movement direction.
type Direction int

const (
	Up Direction = iota
	Down
	Left
	Right
)

// PieceKind distinguishes the four piece types.
type PieceKind int

const (
	Small      PieceKind = iota // 1x1
	Vertical                    // 1x2 (width=1, height=2)
	Horizontal                  // 2x1 (width=2, height=1)
	Large                       // 2x2
)

// Piece represents a block on the board by its top-left corner and kind.
type Piece struct {
	Kind PieceKind
	X, Y int // top-left corner (col, row), 0-indexed
}

// Width returns the piece width.
func (p Piece) Width() int {
	switch p.Kind {
	case Horizontal:
		return 2
	case Large:
		return 2
	default:
		return 1
	}
}

// Height returns the piece height.
func (p Piece) Height() int {
	switch p.Kind {
	case Vertical:
		return 2
	case Large:
		return 2
	default:
		return 1
	}
}

// Cells returns all (col, row) pairs occupied by this piece.
func (p Piece) Cells() [][2]int {
	var cells [][2]int
	for dx := 0; dx < p.Width(); dx++ {
		for dy := 0; dy < p.Height(); dy++ {
			cells = append(cells, [2]int{p.X + dx, p.Y + dy})
		}
	}
	return cells
}

// Board holds the full game state.
type Board struct {
	Pieces []Piece
	Moves  int
}

// occupancy builds a grid where each cell holds the piece index or -1 if empty.
func (b *Board) occupancy() [BoardW][BoardH]int {
	var grid [BoardW][BoardH]int
	for x := range BoardW {
		for y := range BoardH {
			grid[x][y] = -1
		}
	}
	for i, p := range b.Pieces {
		for _, c := range p.Cells() {
			grid[c[0]][c[1]] = i
		}
	}
	return grid
}

// PieceAt returns the index of the piece at (col, row), or -1 if empty.
func (b *Board) PieceAt(x, y int) int {
	grid := b.occupancy()
	if x < 0 || x >= BoardW || y < 0 || y >= BoardH {
		return -1
	}
	return grid[x][y]
}

// CanMove checks if piece at index i can move in the given direction.
func (b *Board) CanMove(i int, dir Direction) bool {
	if i < 0 || i >= len(b.Pieces) {
		return false
	}
	grid := b.occupancy()
	p := b.Pieces[i]
	dx, dy := dirDelta(dir)

	for _, c := range p.Cells() {
		nx, ny := c[0]+dx, c[1]+dy
		if nx < 0 || nx >= BoardW || ny < 0 || ny >= BoardH {
			return false
		}
		occupant := grid[nx][ny]
		if occupant != -1 && occupant != i {
			return false
		}
	}
	return true
}

// Move moves piece i in the given direction. Returns true if the move was made.
func (b *Board) Move(i int, dir Direction) bool {
	if !b.CanMove(i, dir) {
		return false
	}
	dx, dy := dirDelta(dir)
	b.Pieces[i].X += dx
	b.Pieces[i].Y += dy
	return true
}

// pieceDist returns the Manhattan distance between two pieces' positions.
func pieceDist(a, b Piece) int {
	dx := a.X - b.X
	if dx < 0 {
		dx = -dx
	}
	dy := a.Y - b.Y
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}

// IsWon returns true if the large 2x2 block is at the bottom-center (col 1, row 3).
func (b *Board) IsWon() bool {
	for _, p := range b.Pieces {
		if p.Kind == Large {
			return p.X == 1 && p.Y == 3
		}
	}
	return false
}

// Clone returns a deep copy of the board.
func (b *Board) Clone() *Board {
	pieces := make([]Piece, len(b.Pieces))
	copy(pieces, b.Pieces)
	return &Board{Pieces: pieces, Moves: b.Moves}
}

func dirDelta(dir Direction) (int, int) {
	switch dir {
	case Up:
		return 0, -1
	case Down:
		return 0, 1
	case Left:
		return -1, 0
	case Right:
		return 1, 0
	}
	return 0, 0
}

// CanPlace checks if a piece can be placed on the board without going out of
// bounds or overlapping existing pieces.
func (b *Board) CanPlace(p Piece) bool {
	grid := b.occupancy()
	for _, c := range p.Cells() {
		if c[0] < 0 || c[0] >= BoardW || c[1] < 0 || c[1] >= BoardH {
			return false
		}
		if grid[c[0]][c[1]] != -1 {
			return false
		}
	}
	return true
}

// RemovePieceAt removes the piece occupying (x, y) and returns true,
// or returns false if the cell is empty.
func (b *Board) RemovePieceAt(x, y int) bool {
	idx := b.PieceAt(x, y)
	if idx == -1 {
		return false
	}
	b.Pieces = append(b.Pieces[:idx], b.Pieces[idx+1:]...)
	return true
}

// NewRandomBoard generates a random valid starting position that matches
// the given difficulty. It repeatedly generates random layouts, solves each
// with BFS, and returns the first one whose optimal move count falls within
// the difficulty's range.
func NewRandomBoard(diff Difficulty) (*Board, int) {
	lo, hi := difficultyRange(diff)
	for {
		b := tryGenerateBoard()
		if b == nil {
			continue
		}
		opt := Solve(b)
		if opt >= lo && opt < hi {
			return b, opt
		}
	}
}

func tryGenerateBoard() *Board {
	// Pieces: 1 Large (2x2)=4, 5 Medium (1x2 or 2x1)=10, 4 Small (1x1)=4 => 18 cells.
	// Board: 4x5 = 20 cells => 2 empty cells.

	var grid [BoardW][BoardH]bool
	var pieces []Piece

	// Place the large piece first (2x2). It can go at x=0..2, y=0..3.
	// Avoid the win position (1,3).
	largePositions := [][2]int{}
	for x := 0; x <= BoardW-2; x++ {
		for y := 0; y <= BoardH-2; y++ {
			if x == 1 && y == 3 {
				continue // skip win position
			}
			largePositions = append(largePositions, [2]int{x, y})
		}
	}
	rand.Shuffle(len(largePositions), func(i, j int) {
		largePositions[i], largePositions[j] = largePositions[j], largePositions[i]
	})

	lp := largePositions[0]
	large := Piece{Kind: Large, X: lp[0], Y: lp[1]}
	for _, c := range large.Cells() {
		grid[c[0]][c[1]] = true
	}
	pieces = append(pieces, large)

	// Place 5 medium pieces. Each can be vertical (1x2) or horizontal (2x1),
	// chosen randomly per piece.
	type medCandidate struct {
		x, y int
		kind PieceKind
	}
	medPlaced := 0
	candidates := []medCandidate{}
	for x := range BoardW {
		for y := 0; y <= BoardH-2; y++ {
			candidates = append(candidates, medCandidate{x, y, Vertical})
		}
	}
	for x := 0; x <= BoardW-2; x++ {
		for y := range BoardH {
			candidates = append(candidates, medCandidate{x, y, Horizontal})
		}
	}
	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})
	for _, c := range candidates {
		if medPlaced >= 5 {
			break
		}
		p := Piece{Kind: c.kind, X: c.x, Y: c.y}
		fits := true
		for _, cell := range p.Cells() {
			if cell[0] < 0 || cell[0] >= BoardW || cell[1] < 0 || cell[1] >= BoardH {
				fits = false
				break
			}
			if grid[cell[0]][cell[1]] {
				fits = false
				break
			}
		}
		if fits {
			for _, cell := range p.Cells() {
				grid[cell[0]][cell[1]] = true
			}
			pieces = append(pieces, p)
			medPlaced++
		}
	}
	if medPlaced < 5 {
		return nil
	}

	// Place 4 small pieces (1x1).
	smallPlaced := 0
	smallPositions := [][2]int{}
	for x := range BoardW {
		for y := range BoardH {
			smallPositions = append(smallPositions, [2]int{x, y})
		}
	}
	rand.Shuffle(len(smallPositions), func(i, j int) {
		smallPositions[i], smallPositions[j] = smallPositions[j], smallPositions[i]
	})
	for _, pos := range smallPositions {
		if smallPlaced >= 4 {
			break
		}
		x, y := pos[0], pos[1]
		if !grid[x][y] {
			grid[x][y] = true
			pieces = append(pieces, Piece{Kind: Small, X: x, Y: y})
			smallPlaced++
		}
	}
	if smallPlaced < 4 {
		return nil
	}

	return &Board{Pieces: pieces, Moves: 0}
}
