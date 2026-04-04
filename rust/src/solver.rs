use std::collections::{HashMap, VecDeque};

use crate::board::*;

/// Difficulty represents a game difficulty level.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Difficulty {
    Easy,
    Medium,
    Hard,
    Custom,
}

impl Difficulty {
    pub fn as_str(self) -> &'static str {
        match self {
            Difficulty::Easy => "Easy",
            Difficulty::Medium => "Medium",
            Difficulty::Hard => "Hard",
            Difficulty::Custom => "Custom",
        }
    }
}

/// Returns (min, max) move range for a difficulty.
pub fn difficulty_range(d: Difficulty) -> (i32, i32) {
    match d {
        Difficulty::Easy => (1, 40),
        Difficulty::Medium => (40, 80),
        Difficulty::Hard => (80, 9999),
        Difficulty::Custom => (0, 0),
    }
}

/// Canonicalize sorts pieces by (kind, x, y) so that interchangeable
/// same-kind pieces always appear in a deterministic order.
fn canonicalize(pieces: &[Piece]) -> Vec<Piece> {
    let mut sorted: Vec<Piece> = pieces.to_vec();
    sorted.sort_by(|a, b| a.kind.cmp(&b.kind).then(a.x.cmp(&b.x)).then(a.y.cmp(&b.y)));
    sorted
}

/// Produces a compact byte key for a set of (already canonical) pieces.
fn encode_state(pieces: &[Piece]) -> Vec<u8> {
    let mut buf = Vec::with_capacity(pieces.len() * 3);
    for p in pieces {
        buf.push(p.kind as u8);
        buf.push(p.x as u8);
        buf.push(p.y as u8);
    }
    buf
}

/// Solve returns the minimum number of moves to reach the win state using BFS,
/// or -1 if the board is unsolvable.
pub fn solve(b: &Board) -> i32 {
    let initial = canonicalize(&b.pieces);
    let key = encode_state(&initial);

    let mut visited: HashMap<Vec<u8>, ()> = HashMap::new();
    visited.insert(key, ());

    let mut queue: VecDeque<(Vec<Piece>, i32)> = VecDeque::new();
    queue.push_back((initial, 0));

    while let Some((cur_pieces, depth)) = queue.pop_front() {
        // Check win.
        for p in &cur_pieces {
            if p.kind == PieceKind::Large && p.x == 1 && p.y == 3 {
                return depth;
            }
        }

        // Build occupancy grid.
        let mut grid = [[-1i32; BOARD_H]; BOARD_W];
        for (i, p) in cur_pieces.iter().enumerate() {
            for (cx, cy) in p.cells() {
                grid[cx as usize][cy as usize] = i as i32;
            }
        }

        // Try moving each piece in each direction.
        for (i, p) in cur_pieces.iter().enumerate() {
            for dir in &ALL_DIRS {
                let (dx, dy) = dir.delta();
                let mut can_move = true;
                for (cx, cy) in p.cells() {
                    let nx = cx + dx;
                    let ny = cy + dy;
                    if nx < 0 || nx >= BOARD_W as i32 || ny < 0 || ny >= BOARD_H as i32 {
                        can_move = false;
                        break;
                    }
                    let occ = grid[nx as usize][ny as usize];
                    if occ != -1 && occ != i as i32 {
                        can_move = false;
                        break;
                    }
                }
                if !can_move {
                    continue;
                }

                let mut new_pieces = cur_pieces.clone();
                new_pieces[i] = Piece::new(p.kind, p.x + dx, p.y + dy);
                let new_pieces = canonicalize(&new_pieces);
                let k = encode_state(&new_pieces);

                if !visited.contains_key(&k) {
                    visited.insert(k, ());
                    queue.push_back((new_pieces, depth + 1));
                }
            }
        }
    }

    -1 // unsolvable
}

/// Hint represents a suggested next move.
#[derive(Debug, Clone)]
pub struct Hint {
    pub piece_index: usize,
    pub dir: Direction,
}

/// SolveNextMove finds the optimal next move from the current board state.
/// Returns the first move on an optimal solution path, mapped back to the actual
/// piece indices in the given board. Returns None if already won or unsolvable.
pub fn solve_next_move(b: &Board) -> Option<Hint> {
    if b.is_won() {
        return None;
    }

    let initial = canonicalize(&b.pieces);
    let init_key = encode_state(&initial);
    let mut visited: HashMap<Vec<u8>, ()> = HashMap::new();
    visited.insert(init_key, ());

    // Build occupancy for initial canonical state.
    let mut grid0 = [[-1i32; BOARD_H]; BOARD_W];
    for (i, p) in initial.iter().enumerate() {
        for (cx, cy) in p.cells() {
            grid0[cx as usize][cy as usize] = i as i32;
        }
    }

    // Expand initial state to depth-1 states.
    // Each is tagged with its own key (the first-move key).
    let mut queue: VecDeque<(Vec<Piece>, Vec<u8>)> = VecDeque::new();

    for (i, p) in initial.iter().enumerate() {
        for dir in &ALL_DIRS {
            let (dx, dy) = dir.delta();
            let mut can_move = true;
            for (cx, cy) in p.cells() {
                let nx = cx + dx;
                let ny = cy + dy;
                if nx < 0 || nx >= BOARD_W as i32 || ny < 0 || ny >= BOARD_H as i32 {
                    can_move = false;
                    break;
                }
                let occ = grid0[nx as usize][ny as usize];
                if occ != -1 && occ != i as i32 {
                    can_move = false;
                    break;
                }
            }
            if !can_move {
                continue;
            }

            let mut new_pieces = initial.clone();
            new_pieces[i] = Piece::new(p.kind, p.x + dx, p.y + dy);
            let new_pieces = canonicalize(&new_pieces);
            let k = encode_state(&new_pieces);

            if visited.contains_key(&k) {
                continue;
            }
            visited.insert(k.clone(), ());

            // Check if this single move wins.
            for pp in &new_pieces {
                if pp.kind == PieceKind::Large && pp.x == 1 && pp.y == 3 {
                    return map_to_original_board(b, &k);
                }
            }
            queue.push_back((new_pieces, k));
        }
    }

    // BFS from depth-1 onward.
    while let Some((cur_pieces, first_move_key)) = queue.pop_front() {
        let mut grid = [[-1i32; BOARD_H]; BOARD_W];
        for (i, p) in cur_pieces.iter().enumerate() {
            for (cx, cy) in p.cells() {
                grid[cx as usize][cy as usize] = i as i32;
            }
        }

        for (i, p) in cur_pieces.iter().enumerate() {
            for dir in &ALL_DIRS {
                let (dx, dy) = dir.delta();
                let mut can_move = true;
                for (cx, cy) in p.cells() {
                    let nx = cx + dx;
                    let ny = cy + dy;
                    if nx < 0 || nx >= BOARD_W as i32 || ny < 0 || ny >= BOARD_H as i32 {
                        can_move = false;
                        break;
                    }
                    let occ = grid[nx as usize][ny as usize];
                    if occ != -1 && occ != i as i32 {
                        can_move = false;
                        break;
                    }
                }
                if !can_move {
                    continue;
                }

                let mut new_pieces = cur_pieces.clone();
                new_pieces[i] = Piece::new(p.kind, p.x + dx, p.y + dy);
                let new_pieces = canonicalize(&new_pieces);
                let k = encode_state(&new_pieces);

                if visited.contains_key(&k) {
                    continue;
                }
                visited.insert(k, ());

                for pp in &new_pieces {
                    if pp.kind == PieceKind::Large && pp.x == 1 && pp.y == 3 {
                        return map_to_original_board(b, &first_move_key);
                    }
                }
                queue.push_back((new_pieces, first_move_key.clone()));
            }
        }
    }

    None // unsolvable
}

/// Maps a canonical depth-1 key back to an actual piece index and direction
/// in the original board.
fn map_to_original_board(b: &Board, depth1_key: &[u8]) -> Option<Hint> {
    for i in 0..b.pieces.len() {
        for dir in &ALL_DIRS {
            if !b.can_move(i, *dir) {
                continue;
            }
            let (dx, dy) = dir.delta();
            let mut new_pieces = b.pieces.clone();
            new_pieces[i] = Piece::new(b.pieces[i].kind, b.pieces[i].x + dx, b.pieces[i].y + dy);
            let canonical = canonicalize(&new_pieces);
            if encode_state(&canonical) == depth1_key {
                return Some(Hint {
                    piece_index: i,
                    dir: *dir,
                });
            }
        }
    }
    None
}
