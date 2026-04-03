package main

// presetEntry holds a pre-generated puzzle with its known optimal solution.
type presetEntry struct {
	Optimal int
	Pieces  []Piece
}

// calcScore returns the score (1-10) for completing a puzzle.
// 10 points for optimal solution, decreasing proportionally for more moves.
// Minimum 1 point for any solve.
func calcScore(optimal, actual int) int {
	if actual <= 0 || optimal <= 0 {
		return 0
	}
	// Integer rounding of 10 * optimal / actual.
	s := (10*optimal + actual/2) / actual
	if s > 10 {
		s = 10
	}
	if s < 1 {
		s = 1
	}
	return s
}

// maxLeagueScore returns the maximum possible total score across all presets.
func maxLeagueScore() int {
	return len(presets) * 10
}
