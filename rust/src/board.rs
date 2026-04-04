use rand::seq::SliceRandom;
use rand::thread_rng;

/// Board dimensions.
pub const BOARD_W: usize = 4;
pub const BOARD_H: usize = 5;

/// Direction represents a movement direction.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Direction {
    Up,
    Down,
    Left,
    Right,
}

impl Direction {
    pub fn delta(self) -> (i32, i32) {
        match self {
            Direction::Up => (0, -1),
            Direction::Down => (0, 1),
            Direction::Left => (-1, 0),
            Direction::Right => (1, 0),
        }
    }

    pub fn arrow(self) -> &'static str {
        match self {
            Direction::Up => "\u{2191}",
            Direction::Down => "\u{2193}",
            Direction::Left => "\u{2190}",
            Direction::Right => "\u{2192}",
        }
    }
}

pub const ALL_DIRS: [Direction; 4] = [
    Direction::Up,
    Direction::Down,
    Direction::Left,
    Direction::Right,
];

/// PieceKind distinguishes the four piece types.
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub enum PieceKind {
    Small,      // 1x1
    Vertical,   // 1x2 (width=1, height=2)
    Horizontal, // 2x1 (width=2, height=1)
    Large,      // 2x2
}

/// Piece represents a block on the board by its top-left corner and kind.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Piece {
    pub kind: PieceKind,
    pub x: i32,
    pub y: i32,
}

impl Piece {
    pub fn new(kind: PieceKind, x: i32, y: i32) -> Self {
        Self { kind, x, y }
    }

    pub fn width(self) -> i32 {
        match self.kind {
            PieceKind::Horizontal | PieceKind::Large => 2,
            _ => 1,
        }
    }

    pub fn height(self) -> i32 {
        match self.kind {
            PieceKind::Vertical | PieceKind::Large => 2,
            _ => 1,
        }
    }

    /// Returns all (col, row) pairs occupied by this piece.
    pub fn cells(self) -> Vec<(i32, i32)> {
        let mut cells = Vec::new();
        for dx in 0..self.width() {
            for dy in 0..self.height() {
                cells.push((self.x + dx, self.y + dy));
            }
        }
        cells
    }
}

/// Manhattan distance between two pieces' positions.
pub fn piece_dist(a: &Piece, b: &Piece) -> i32 {
    (a.x - b.x).abs() + (a.y - b.y).abs()
}

/// Board holds the full game state.
#[derive(Debug, Clone)]
pub struct Board {
    pub pieces: Vec<Piece>,
    pub moves: i32,
}

impl Board {
    pub fn new(pieces: Vec<Piece>) -> Self {
        Self { pieces, moves: 0 }
    }

    /// Builds a grid where each cell holds the piece index or -1 if empty.
    pub fn occupancy(&self) -> [[i32; BOARD_H]; BOARD_W] {
        let mut grid = [[-1i32; BOARD_H]; BOARD_W];
        for (i, p) in self.pieces.iter().enumerate() {
            for (cx, cy) in p.cells() {
                if cx >= 0 && (cx as usize) < BOARD_W && cy >= 0 && (cy as usize) < BOARD_H {
                    grid[cx as usize][cy as usize] = i as i32;
                }
            }
        }
        grid
    }

    /// Returns the index of the piece at (col, row), or -1 if empty.
    pub fn piece_at(&self, x: i32, y: i32) -> i32 {
        if x < 0 || x >= BOARD_W as i32 || y < 0 || y >= BOARD_H as i32 {
            return -1;
        }
        let grid = self.occupancy();
        grid[x as usize][y as usize]
    }

    /// Checks if piece at index i can move in the given direction.
    pub fn can_move(&self, i: usize, dir: Direction) -> bool {
        if i >= self.pieces.len() {
            return false;
        }
        let grid = self.occupancy();
        let p = &self.pieces[i];
        let (dx, dy) = dir.delta();

        for (cx, cy) in p.cells() {
            let nx = cx + dx;
            let ny = cy + dy;
            if nx < 0 || nx >= BOARD_W as i32 || ny < 0 || ny >= BOARD_H as i32 {
                return false;
            }
            let occupant = grid[nx as usize][ny as usize];
            if occupant != -1 && occupant != i as i32 {
                return false;
            }
        }
        true
    }

    /// Moves piece i in the given direction. Returns true if the move was made.
    pub fn move_piece(&mut self, i: usize, dir: Direction) -> bool {
        if !self.can_move(i, dir) {
            return false;
        }
        let (dx, dy) = dir.delta();
        self.pieces[i].x += dx;
        self.pieces[i].y += dy;
        true
    }

    /// Returns true if the large 2x2 block is at the bottom-center (col 1, row 3).
    pub fn is_won(&self) -> bool {
        for p in &self.pieces {
            if p.kind == PieceKind::Large {
                return p.x == 1 && p.y == 3;
            }
        }
        false
    }

    /// Checks if a piece can be placed on the board without overlap or out of bounds.
    pub fn can_place(&self, p: &Piece) -> bool {
        let grid = self.occupancy();
        for (cx, cy) in p.cells() {
            if cx < 0 || cx >= BOARD_W as i32 || cy < 0 || cy >= BOARD_H as i32 {
                return false;
            }
            if grid[cx as usize][cy as usize] != -1 {
                return false;
            }
        }
        true
    }

    /// Removes the piece occupying (x, y). Returns true if removed.
    pub fn remove_piece_at(&mut self, x: i32, y: i32) -> bool {
        let idx = self.piece_at(x, y);
        if idx == -1 {
            return false;
        }
        self.pieces.remove(idx as usize);
        true
    }
}

/// Generates a random valid starting position that matches the given difficulty.
/// Returns (board, optimal_move_count).
pub fn new_random_board(lo: i32, hi: i32, solve_fn: impl Fn(&Board) -> i32) -> (Board, i32) {
    loop {
        if let Some(b) = try_generate_board() {
            let opt = solve_fn(&b);
            if opt >= lo && opt < hi {
                return (b, opt);
            }
        }
    }
}

fn try_generate_board() -> Option<Board> {
    let mut rng = thread_rng();
    let mut grid = [[false; BOARD_H]; BOARD_W];
    let mut pieces = Vec::new();

    // Place the large piece first (2x2). Avoid the win position (1,3).
    let mut large_positions: Vec<(i32, i32)> = Vec::new();
    for x in 0..=(BOARD_W as i32 - 2) {
        for y in 0..=(BOARD_H as i32 - 2) {
            if x == 1 && y == 3 {
                continue;
            }
            large_positions.push((x, y));
        }
    }
    large_positions.shuffle(&mut rng);

    let (lx, ly) = large_positions[0];
    let large = Piece::new(PieceKind::Large, lx, ly);
    for (cx, cy) in large.cells() {
        grid[cx as usize][cy as usize] = true;
    }
    pieces.push(large);

    // Place 5 medium pieces. Each can be vertical (1x2) or horizontal (2x1).
    struct MedCandidate {
        x: i32,
        y: i32,
        kind: PieceKind,
    }

    let mut candidates: Vec<MedCandidate> = Vec::new();
    for x in 0..BOARD_W as i32 {
        for y in 0..=(BOARD_H as i32 - 2) {
            candidates.push(MedCandidate {
                x,
                y,
                kind: PieceKind::Vertical,
            });
        }
    }
    for x in 0..=(BOARD_W as i32 - 2) {
        for y in 0..BOARD_H as i32 {
            candidates.push(MedCandidate {
                x,
                y,
                kind: PieceKind::Horizontal,
            });
        }
    }
    candidates.shuffle(&mut rng);

    let mut med_placed = 0;
    for c in &candidates {
        if med_placed >= 5 {
            break;
        }
        let p = Piece::new(c.kind, c.x, c.y);
        let mut fits = true;
        for (cx, cy) in p.cells() {
            if cx < 0 || cx >= BOARD_W as i32 || cy < 0 || cy >= BOARD_H as i32 {
                fits = false;
                break;
            }
            if grid[cx as usize][cy as usize] {
                fits = false;
                break;
            }
        }
        if fits {
            for (cx, cy) in p.cells() {
                grid[cx as usize][cy as usize] = true;
            }
            pieces.push(p);
            med_placed += 1;
        }
    }
    if med_placed < 5 {
        return None;
    }

    // Place 4 small pieces (1x1).
    let mut small_positions: Vec<(i32, i32)> = Vec::new();
    for x in 0..BOARD_W as i32 {
        for y in 0..BOARD_H as i32 {
            small_positions.push((x, y));
        }
    }
    small_positions.shuffle(&mut rng);

    let mut small_placed = 0;
    for (sx, sy) in &small_positions {
        if small_placed >= 4 {
            break;
        }
        if !grid[*sx as usize][*sy as usize] {
            grid[*sx as usize][*sy as usize] = true;
            pieces.push(Piece::new(PieceKind::Small, *sx, *sy));
            small_placed += 1;
        }
    }
    if small_placed < 4 {
        return None;
    }

    Some(Board::new(pieces))
}

/// Count pieces by kind.
pub fn count_pieces(b: &Board) -> (i32, i32, i32, i32) {
    let (mut large, mut vert, mut horiz, mut small) = (0, 0, 0, 0);
    for p in &b.pieces {
        match p.kind {
            PieceKind::Large => large += 1,
            PieceKind::Vertical => vert += 1,
            PieceKind::Horizontal => horiz += 1,
            PieceKind::Small => small += 1,
        }
    }
    (large, vert, horiz, small)
}

/// Validates editor board for play. Returns error message or empty string.
pub fn validate_editor(b: &Board) -> &'static str {
    let (l, _, _, _) = count_pieces(b);
    if l == 0 {
        return "Need exactly 1 Large (2x2) piece.";
    }
    if l > 1 {
        return "Too many Large pieces (max 1).";
    }
    if b.is_won() {
        return "Large piece is already at the goal!";
    }
    let occupied: usize = b.pieces.iter().map(|p| p.cells().len()).sum();
    if BOARD_W * BOARD_H - occupied < 2 {
        return "Need at least 2 empty cells.";
    }
    ""
}
