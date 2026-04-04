use crate::board::Piece;

/// A pre-generated puzzle with its known optimal solution.
#[derive(Debug, Clone)]
pub struct PresetEntry {
    pub optimal: i32,
    pub pieces: Vec<Piece>,
}

/// Returns the score (1-10) for completing a puzzle.
/// 10 points for optimal solution, decreasing proportionally for more moves.
/// Minimum 1 point for any solve.
pub fn calc_score(optimal: i32, actual: i32) -> i32 {
    if actual <= 0 || optimal <= 0 {
        return 0;
    }
    // Integer rounding of 10 * optimal / actual.
    let s = (10 * optimal + actual / 2) / actual;
    s.clamp(1, 10)
}

/// Returns the maximum possible total score across all presets.
pub fn max_league_score(num_presets: usize) -> i32 {
    num_presets as i32 * 10
}
