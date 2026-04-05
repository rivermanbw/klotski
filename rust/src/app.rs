use std::sync::mpsc;
use std::thread;

use crate::board::*;
use crate::league::*;
use crate::presets::presets;
use crate::save::*;
use crate::solver::*;
use crate::sound::SoundEngine;

/// Game modes.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum GameMode {
    FreePlay,
    Editor,
    NameInput,
    League,
    LeaguePlay,
    Leaderboard,
}

/// Messages from background threads.
pub enum BgMsg {
    BoardReady {
        board: Board,
        optimal: i32,
        diff: Difficulty,
    },
    HintReady {
        hint: Option<Hint>,
        seq: i32,
    },
    EditorSolve {
        optimal: i32,
    },
}

pub const LEAGUE_VISIBLE: usize = 15;

/// The main application state.
pub struct App {
    pub mode: GameMode,

    pub board: Board,
    pub cursor_x: i32,
    pub cursor_y: i32,
    pub selected: i32, // index of selected piece, or -1
    pub pre_select_board: Option<Board>,
    pub history: Vec<Board>,
    pub won: bool,
    pub difficulty: Difficulty,
    pub optimal: i32,
    pub loading: bool,
    pub show_coords: bool,

    pub cheat_mode: bool,
    pub hint: Option<Hint>,
    pub hint_loading: bool,
    pub hint_seq: i32,

    pub edit_piece: PieceKind,
    pub edit_error: String,
    pub edit_solving: bool,
    pub custom: bool,

    // League fields.
    pub nickname: String,
    pub name_input: String,
    pub save_data: SaveData,
    pub league_idx: usize,
    pub league_scroll: usize,
    pub league_score: i32,
    pub league_new_best: bool,

    // Presets cache.
    pub presets: Vec<PresetEntry>,

    // Background message channel.
    pub bg_rx: mpsc::Receiver<BgMsg>,
    pub bg_tx: mpsc::Sender<BgMsg>,

    // Sound engine.
    pub sound: SoundEngine,
}

impl App {
    pub fn new() -> Self {
        let (tx, rx) = mpsc::channel();
        let app = Self {
            mode: GameMode::FreePlay,
            board: Board::new(vec![]),
            cursor_x: 0,
            cursor_y: 0,
            selected: -1,
            pre_select_board: None,
            history: Vec::new(),
            won: false,
            difficulty: Difficulty::Easy,
            optimal: 0,
            loading: true,
            show_coords: false,
            cheat_mode: false,
            hint: None,
            hint_loading: false,
            hint_seq: 0,
            edit_piece: PieceKind::Large,
            edit_error: String::new(),
            edit_solving: false,
            custom: false,
            nickname: String::new(),
            name_input: String::new(),
            save_data: SaveData::new(),
            league_idx: 0,
            league_scroll: 0,
            league_score: 0,
            league_new_best: false,
            presets: presets(),
            bg_rx: rx,
            bg_tx: tx,
            sound: SoundEngine::new(),
        };
        app.generate_board_bg(Difficulty::Easy);
        app
    }

    /// Process any pending background messages. Returns true if something changed.
    pub fn poll_bg(&mut self) -> bool {
        let mut changed = false;
        while let Ok(msg) = self.bg_rx.try_recv() {
            changed = true;
            match msg {
                BgMsg::BoardReady {
                    board,
                    optimal,
                    diff,
                } => {
                    self.board = board;
                    self.optimal = optimal;
                    self.difficulty = diff;
                    self.loading = false;
                    self.won = false;
                    self.selected = -1;
                    self.pre_select_board = None;
                    self.history.clear();
                    self.cursor_x = 0;
                    self.cursor_y = 0;
                    self.hint = None;
                    self.hint_loading = false;
                    self.custom = false;
                    if self.cheat_mode {
                        self.compute_hint_bg();
                    }
                }
                BgMsg::HintReady { hint, seq } => {
                    if seq == self.hint_seq && self.cheat_mode {
                        self.hint = hint;
                        self.hint_loading = false;
                    }
                }
                BgMsg::EditorSolve { optimal } => {
                    self.edit_solving = false;
                    if optimal == -1 {
                        self.edit_error = "Unsolvable! Adjust pieces and try again.".to_string();
                    } else {
                        self.mode = GameMode::FreePlay;
                        self.optimal = optimal;
                        self.difficulty = Difficulty::Custom;
                        self.custom = true;
                        self.won = false;
                        self.selected = -1;
                        self.pre_select_board = None;
                        self.history.clear();
                        self.cursor_x = 0;
                        self.cursor_y = 0;
                        self.board.moves = 0;
                        self.edit_error.clear();
                        self.hint = None;
                        self.hint_loading = false;
                        if self.cheat_mode {
                            self.compute_hint_bg();
                        }
                    }
                }
            }
        }
        changed
    }

    /// Spawn background board generation.
    pub fn generate_board_bg(&self, diff: Difficulty) {
        let tx = self.bg_tx.clone();
        let (lo, hi) = difficulty_range(diff);
        thread::spawn(move || {
            let (board, opt) = new_random_board(lo, hi, solve);
            let _ = tx.send(BgMsg::BoardReady {
                board,
                optimal: opt,
                diff,
            });
        });
    }

    /// Spawn background hint computation.
    pub fn compute_hint_bg(&mut self) {
        self.hint_seq += 1;
        self.hint = None;
        self.hint_loading = true;
        let tx = self.bg_tx.clone();
        let board = self.board.clone();
        let seq = self.hint_seq;
        thread::spawn(move || {
            let hint = solve_next_move(&board);
            let _ = tx.send(BgMsg::HintReady { hint, seq });
        });
    }

    /// Spawn background editor solve check.
    pub fn editor_solve_bg(&mut self) {
        self.edit_solving = true;
        self.edit_error.clear();
        let tx = self.bg_tx.clone();
        let board = self.board.clone();
        thread::spawn(move || {
            let optimal = solve(&board);
            let _ = tx.send(BgMsg::EditorSolve { optimal });
        });
    }

    // --- Input handling ---

    /// Handle a key event. Returns true if the app should quit.
    pub fn handle_key(&mut self, key: &str) -> bool {
        // Quit — always available.
        if key == "q" || key == "ctrl+c" {
            return true;
        }

        // Mute toggle — always available except during text input.
        if key == "m" && self.mode != GameMode::NameInput {
            self.sound.toggle_mute();
            return false;
        }

        match self.mode {
            GameMode::Editor => self.update_editor(key),
            GameMode::NameInput => self.update_name_input(key),
            GameMode::League => self.update_league(key),
            GameMode::LeaguePlay => self.update_league_play(key),
            GameMode::Leaderboard => self.update_leaderboard(key),
            GameMode::FreePlay => self.update_free_play(key),
        }
        false
    }

    fn update_free_play(&mut self, key: &str) {
        if self.loading {
            return;
        }

        // Enter league mode.
        if key == "g" {
            self.enter_league();
            return;
        }

        // Enter editor mode.
        if key == "e" && !self.won {
            self.mode = GameMode::Editor;
            self.board = Board::new(vec![]);
            self.cursor_x = 0;
            self.cursor_y = 0;
            self.selected = -1;
            self.pre_select_board = None;
            self.edit_piece = PieceKind::Large;
            self.edit_error.clear();
            self.edit_solving = false;
            self.hint = None;
            self.hint_loading = false;
            return;
        }

        // Cycle difficulty: 1/2/3.
        if key == "1" || key == "2" || key == "3" {
            let d = match key {
                "1" => Difficulty::Easy,
                "2" => Difficulty::Medium,
                "3" => Difficulty::Hard,
                _ => unreachable!(),
            };
            self.difficulty = d;
            self.loading = true;
            self.custom = false;
            self.generate_board_bg(d);
            return;
        }

        // New game.
        if key == "n" {
            if self.custom {
                self.difficulty = Difficulty::Easy;
                self.custom = false;
            }
            self.loading = true;
            self.generate_board_bg(self.difficulty);
            return;
        }

        // Toggle coordinates.
        if key == "c" {
            self.show_coords = !self.show_coords;
            return;
        }

        // Toggle cheat mode.
        if key == "?" {
            self.cheat_mode = !self.cheat_mode;
            if self.cheat_mode && !self.won {
                self.compute_hint_bg();
            } else {
                self.hint = None;
                self.hint_loading = false;
            }
            return;
        }

        self.update_play(key);
    }

    fn update_play(&mut self, key: &str) {
        // Undo.
        if key == "u" {
            if self.selected != -1 {
                if let Some(pre) = self.pre_select_board.take() {
                    self.board = pre;
                }
                self.selected = -1;
                if self.cheat_mode {
                    self.compute_hint_bg();
                }
                return;
            }
            if let Some(prev) = self.history.pop() {
                self.board = prev;
                self.won = false;
                if self.cheat_mode {
                    self.compute_hint_bg();
                }
            }
            return;
        }

        // Reset to starting state.
        if key == "U" {
            if !self.history.is_empty() {
                self.board = self.history[0].clone();
                self.history.clear();
            } else if let Some(pre) = self.pre_select_board.take() {
                self.board = pre;
            } else {
                return;
            }
            self.pre_select_board = None;
            self.selected = -1;
            self.won = false;
            self.hint = None;
            self.hint_loading = false;
            if self.cheat_mode {
                self.compute_hint_bg();
            }
            return;
        }

        if self.won {
            return;
        }

        // Deselect / cancel.
        if key == "esc" {
            if self.selected != -1 {
                if let Some(pre) = self.pre_select_board.take() {
                    self.board = pre;
                }
                self.selected = -1;
                if self.cheat_mode {
                    self.compute_hint_bg();
                }
                return;
            }
            if self.mode == GameMode::LeaguePlay {
                self.enter_league_browser();
            }
            return;
        }

        // Select / confirm.
        if key == "enter" || key == " " {
            if self.selected != -1 {
                // Confirm: compute net displacement and commit.
                if let Some(ref pre) = self.pre_select_board {
                    let disp = piece_dist(
                        &self.board.pieces[self.selected as usize],
                        &pre.pieces[self.selected as usize],
                    );
                    if disp > 0 {
                        self.board.moves += disp;
                        self.history.push(pre.clone());
                    }
                }
                self.pre_select_board = None;
                self.selected = -1;
            } else {
                let idx = self.board.piece_at(self.cursor_x, self.cursor_y);
                if idx != -1 {
                    self.selected = idx;
                    self.pre_select_board = Some(self.board.clone());
                }
            }
            return;
        }

        // Directional input.
        let dir = match key {
            "up" | "k" => Some(Direction::Up),
            "down" | "j" => Some(Direction::Down),
            "left" | "h" => Some(Direction::Left),
            "right" | "l" => Some(Direction::Right),
            _ => None,
        };

        if let Some(dir) = dir {
            if self.selected != -1 {
                let idx = self.selected as usize;
                if self.board.move_piece(idx, dir) {
                    let p = &self.board.pieces[idx];
                    self.cursor_x = p.x;
                    self.cursor_y = p.y;
                    if self.board.is_won() {
                        // Auto-confirm on win.
                        if let Some(ref pre) = self.pre_select_board {
                            let disp = piece_dist(&self.board.pieces[idx], &pre.pieces[idx]);
                            self.board.moves += disp;
                            self.history.push(pre.clone());
                        }
                        self.pre_select_board = None;
                        self.won = true;
                        self.selected = -1;
                        self.hint = None;
                        self.hint_loading = false;
                        self.sound.play_success();
                        // In league play, auto-save the score.
                        if self.mode == GameMode::LeaguePlay {
                            self.league_score = calc_score(self.optimal, self.board.moves);
                            self.league_new_best = false;
                            let pd = self.save_data.player(&self.nickname);
                            let prev = pd.scores.get(&self.league_idx).copied();
                            if prev.is_none() || self.league_score > prev.unwrap() {
                                pd.scores.insert(self.league_idx, self.league_score);
                                self.league_new_best = true;
                                let _ = self.save_data.save();
                            }
                        }
                    } else if self.cheat_mode {
                        self.compute_hint_bg();
                    }
                }
            } else {
                let (dx, dy) = dir.delta();
                let nx = self.cursor_x + dx;
                let ny = self.cursor_y + dy;
                if nx >= 0 && nx < BOARD_W as i32 && ny >= 0 && ny < BOARD_H as i32 {
                    self.cursor_x = nx;
                    self.cursor_y = ny;
                }
            }
        }
    }

    fn update_name_input(&mut self, key: &str) {
        match key {
            "esc" => {
                self.mode = GameMode::FreePlay;
            }
            "enter" => {
                let name = self.name_input.trim().to_string();
                if name.is_empty() {
                    return;
                }
                self.nickname = name;
                self.save_data.last_player = self.nickname.clone();
                let _ = self.save_data.save();
                self.enter_league_browser();
            }
            "backspace" => {
                self.name_input.pop();
            }
            _ => {
                if key.len() == 1 && self.name_input.len() < 20 {
                    self.name_input.push_str(key);
                }
            }
        }
    }

    fn update_league(&mut self, key: &str) {
        let num_presets = self.presets.len();
        match key {
            "esc" => {
                self.mode = GameMode::FreePlay;
            }
            "tab" => {
                self.mode = GameMode::Leaderboard;
            }
            "@" => {
                self.mode = GameMode::NameInput;
                self.name_input.clear();
            }
            "up" | "k" => {
                if self.league_idx > 0 {
                    self.league_idx -= 1;
                    self.adjust_league_scroll();
                }
            }
            "down" | "j" => {
                if self.league_idx < num_presets - 1 {
                    self.league_idx += 1;
                    self.adjust_league_scroll();
                }
            }
            "ctrl+u" => {
                self.league_idx = self.league_idx.saturating_sub(LEAGUE_VISIBLE);
                self.adjust_league_scroll();
            }
            "ctrl+d" => {
                self.league_idx = (self.league_idx + LEAGUE_VISIBLE).min(num_presets - 1);
                self.adjust_league_scroll();
            }
            "home" | "g" => {
                self.league_idx = 0;
                self.adjust_league_scroll();
            }
            "end" | "G" => {
                self.league_idx = num_presets - 1;
                self.adjust_league_scroll();
            }
            "enter" | " " => {
                let preset = &self.presets[self.league_idx];
                self.mode = GameMode::LeaguePlay;
                self.board = Board::new(preset.pieces.clone());
                self.optimal = preset.optimal;
                self.won = false;
                self.selected = -1;
                self.pre_select_board = None;
                self.history.clear();
                self.cursor_x = 0;
                self.cursor_y = 0;
                self.league_score = 0;
                self.league_new_best = false;
                self.cheat_mode = false;
                self.hint = None;
                self.hint_loading = false;
            }
            _ => {}
        }
    }

    fn update_league_play(&mut self, key: &str) {
        // Post-win keys.
        if self.won {
            match key {
                "esc" => {
                    self.enter_league_browser();
                    return;
                }
                "enter" | " " => {
                    let next = self.league_idx + 1;
                    if next < self.presets.len() {
                        self.league_idx = next;
                        let preset = &self.presets[next];
                        self.board = Board::new(preset.pieces.clone());
                        self.optimal = preset.optimal;
                        self.won = false;
                        self.selected = -1;
                        self.pre_select_board = None;
                        self.history.clear();
                        self.cursor_x = 0;
                        self.cursor_y = 0;
                        self.league_score = 0;
                        self.league_new_best = false;
                        self.cheat_mode = false;
                        self.hint = None;
                        self.hint_loading = false;
                        return;
                    }
                    self.enter_league_browser();
                    return;
                }
                _ => {}
            }
        }

        if key == "c" {
            self.show_coords = !self.show_coords;
            return;
        }

        self.update_play(key);
    }

    fn update_leaderboard(&mut self, key: &str) {
        if key == "esc" || key == "tab" {
            self.mode = GameMode::League;
        }
    }

    fn update_editor(&mut self, key: &str) {
        if self.edit_solving {
            return;
        }

        match key {
            "up" | "k" => {
                if self.cursor_y > 0 {
                    self.cursor_y -= 1;
                }
            }
            "down" | "j" => {
                if self.cursor_y < BOARD_H as i32 - 1 {
                    self.cursor_y += 1;
                }
            }
            "left" | "h" => {
                if self.cursor_x > 0 {
                    self.cursor_x -= 1;
                }
            }
            "right" | "l" => {
                if self.cursor_x < BOARD_W as i32 - 1 {
                    self.cursor_x += 1;
                }
            }
            "tab" => {
                self.edit_error.clear();
                self.edit_piece = match self.edit_piece {
                    PieceKind::Large => PieceKind::Vertical,
                    PieceKind::Vertical => PieceKind::Horizontal,
                    PieceKind::Horizontal => PieceKind::Small,
                    PieceKind::Small => PieceKind::Large,
                };
            }
            "enter" | " " => {
                self.edit_error.clear();
                let p = Piece::new(self.edit_piece, self.cursor_x, self.cursor_y);
                if self.board.can_place(&p) {
                    self.board.pieces.push(p);
                } else {
                    self.edit_error =
                        "Can't place here \u{2014} overlaps or out of bounds.".to_string();
                }
            }
            "x" | "backspace" | "delete" => {
                self.edit_error.clear();
                if !self.board.remove_piece_at(self.cursor_x, self.cursor_y) {
                    self.edit_error = "No piece here to remove.".to_string();
                }
            }
            "r" => {
                self.board = Board::new(vec![]);
                self.edit_error.clear();
            }
            "c" => {
                self.show_coords = !self.show_coords;
            }
            "p" => {
                self.edit_error.clear();
                let err = validate_editor(&self.board);
                if !err.is_empty() {
                    self.edit_error = err.to_string();
                    return;
                }
                self.editor_solve_bg();
            }
            "esc" => {
                self.mode = GameMode::FreePlay;
                self.edit_error.clear();
                self.loading = true;
                self.generate_board_bg(self.difficulty);
            }
            _ => {}
        }
    }

    fn enter_league(&mut self) {
        self.save_data = load_save();
        if !self.save_data.last_player.is_empty() {
            self.nickname = self.save_data.last_player.clone();
            self.enter_league_browser();
        } else {
            self.mode = GameMode::NameInput;
            self.name_input.clear();
        }
    }

    fn enter_league_browser(&mut self) {
        self.mode = GameMode::League;
        self.league_score = 0;
        self.league_new_best = false;
        let nickname = self.nickname.clone();
        let pd = self.save_data.player(&nickname);
        // Position cursor at the first unscored puzzle.
        self.league_idx = 0;
        let num_presets = self.presets.len();
        while self.league_idx < num_presets - 1 {
            if !pd.scores.contains_key(&self.league_idx) {
                break;
            }
            self.league_idx += 1;
        }
        self.league_scroll = 0;
        self.adjust_league_scroll();
    }

    fn adjust_league_scroll(&mut self) {
        if self.league_idx < self.league_scroll {
            self.league_scroll = self.league_idx;
        }
        if self.league_idx >= self.league_scroll + LEAGUE_VISIBLE {
            self.league_scroll = self.league_idx - LEAGUE_VISIBLE + 1;
        }
    }
}
