use ratatui::{
    style::{Color, Modifier, Style},
    text::{Line, Span},
};

use crate::app::*;
use crate::board::*;
use crate::league::*;
use crate::solver::*;

// --- Color constants (256-color) ---
fn color256(n: u8) -> Color {
    Color::Indexed(n)
}

const fn c(n: u8) -> Color {
    Color::Indexed(n)
}

// Piece foreground colors.
const COLOR_SMALL: Color = c(214); // orange
const COLOR_MED_VERT: Color = c(39); // blue
const COLOR_MED_HORIZ: Color = c(82); // green
const COLOR_LARGE: Color = c(196); // red
const COLOR_EMPTY: Color = c(236); // dark gray
const COLOR_CURSOR: Color = c(226); // yellow
const COLOR_WIN: Color = c(82); // bright green

const COLOR_EASY: Color = c(82); // green
const COLOR_MED_DIF: Color = c(214); // orange
const COLOR_HARD: Color = c(196); // red
const COLOR_HINT_BG: Color = c(53); // dark purple
const COLOR_HINT_FG: Color = c(213); // bright pink
const COLOR_CUSTOM: Color = c(45); // cyan
const COLOR_GHOST: Color = c(240); // dim gray
const COLOR_EDITOR: Color = c(177); // light purple
const COLOR_ERROR: Color = c(196); // red
const COLOR_LEAGUE: Color = c(220); // gold
const COLOR_LOCKED: Color = c(240); // dim
const COLOR_SCORED: Color = c(82); // green
const COLOR_RANK: Color = c(220); // gold

// Piece background colors (subtle dark tints).
const BG_SMALL: Color = c(58); // dark amber
const BG_VERT: Color = c(17); // dark navy
const BG_HORIZ: Color = c(22); // dark green
const BG_LARGE: Color = c(52); // dark red

// Selected piece backgrounds (brighter).
const BG_SEL_SMALL: Color = c(94); // medium amber
const BG_SEL_VERT: Color = c(18); // medium blue
const BG_SEL_HORIZ: Color = c(28); // medium green
const BG_SEL_LARGE: Color = c(88); // medium red

// Win background.
const BG_WIN_LARGE: Color = c(22); // dark green

const DIM: Color = c(245);
const WHITE: Color = c(255);
const BLACK: Color = c(0);

// Score gradient colors (1-10).
const SCORE_COLORS: [Color; 11] = [
    c(0),   // 0 unused
    c(196), // 1 red
    c(202), // 2 orange-red
    c(208), // 3 dark orange
    c(214), // 4 orange
    c(220), // 5 gold
    c(226), // 6 yellow
    c(190), // 7 yellow-green
    c(154), // 8 light green
    c(118), // 9 green
    c(82),  // 10 bright green
];

fn score_color(score: i32) -> Color {
    let s = score.clamp(1, 10) as usize;
    SCORE_COLORS[s]
}

fn diff_color(d: Difficulty) -> Color {
    match d {
        Difficulty::Easy => COLOR_EASY,
        Difficulty::Medium => COLOR_MED_DIF,
        Difficulty::Hard => COLOR_HARD,
        Difficulty::Custom => COLOR_CUSTOM,
    }
}

fn piece_fg(kind: PieceKind) -> Color {
    match kind {
        PieceKind::Small => COLOR_SMALL,
        PieceKind::Vertical => COLOR_MED_VERT,
        PieceKind::Horizontal => COLOR_MED_HORIZ,
        PieceKind::Large => COLOR_LARGE,
    }
}

fn piece_bg(kind: PieceKind) -> Color {
    match kind {
        PieceKind::Small => BG_SMALL,
        PieceKind::Vertical => BG_VERT,
        PieceKind::Horizontal => BG_HORIZ,
        PieceKind::Large => BG_LARGE,
    }
}

fn piece_sel_bg(kind: PieceKind) -> Color {
    match kind {
        PieceKind::Small => BG_SEL_SMALL,
        PieceKind::Vertical => BG_SEL_VERT,
        PieceKind::Horizontal => BG_SEL_HORIZ,
        PieceKind::Large => BG_SEL_LARGE,
    }
}

fn edit_piece_short(k: PieceKind) -> &'static str {
    match k {
        PieceKind::Large => "L",
        PieceKind::Vertical => "V",
        PieceKind::Horizontal => "H",
        PieceKind::Small => "s",
    }
}

fn edit_piece_label(k: PieceKind) -> &'static str {
    match k {
        PieceKind::Large => "Large 2x2",
        PieceKind::Vertical => "Vertical 1x2",
        PieceKind::Horizontal => "Horizontal 2x1",
        PieceKind::Small => "Small 1x1",
    }
}

/// Computes the effective background color for a piece cell.
fn cell_bg(app: &App, idx: i32) -> Option<Color> {
    if idx == -1 {
        return None;
    }
    let p = &app.board.pieces[idx as usize];
    if idx == app.selected {
        return Some(piece_sel_bg(p.kind));
    }
    if app.cheat_mode {
        if let Some(ref hint) = app.hint {
            if idx as usize == hint.piece_index && !app.won {
                return Some(COLOR_HINT_BG);
            }
        }
    }
    if p.kind == PieceKind::Large && app.won {
        return Some(BG_WIN_LARGE);
    }
    Some(piece_bg(p.kind))
}

/// The main rendering function. Returns lines of styled text.
pub fn render(app: &App) -> Vec<Line<'static>> {
    match app.mode {
        GameMode::FreePlay => render_free_play(app),
        GameMode::Editor => render_editor(app),
        GameMode::NameInput => render_name_input(app),
        GameMode::League => render_league(app),
        GameMode::LeaguePlay => render_league_play(app),
        GameMode::Leaderboard => render_leaderboard(app),
    }
}

fn render_free_play(app: &App) -> Vec<Line<'static>> {
    let mut lines: Vec<Line<'static>> = Vec::new();

    // Title line.
    let mut title_spans: Vec<Span<'static>> = vec![
        Span::styled(
            "KLOTSKI PUZZLE",
            Style::default().fg(WHITE).add_modifier(Modifier::BOLD),
        ),
        Span::raw("  "),
        Span::styled(
            format!(" {} ", app.difficulty.as_str()),
            Style::default()
                .fg(BLACK)
                .bg(diff_color(app.difficulty))
                .add_modifier(Modifier::BOLD),
        ),
    ];

    if app.optimal > 0 {
        title_spans.push(Span::styled(
            format!("  Best: {} moves", app.optimal),
            Style::default().fg(DIM),
        ));
    }

    if app.cheat_mode {
        title_spans.push(Span::raw("  "));
        title_spans.push(Span::styled(
            " CHEAT ",
            Style::default()
                .fg(BLACK)
                .bg(COLOR_HINT_FG)
                .add_modifier(Modifier::BOLD),
        ));
    }

    lines.push(Line::from(title_spans));
    lines.push(Line::raw(""));

    if app.loading {
        lines.push(Line::from(Span::styled(
            "  Generating puzzle...",
            Style::default().fg(DIM),
        )));
        return lines;
    }

    render_board_lines(app, &mut lines);
    lines.push(Line::raw(""));

    // Status: moves.
    let mut moves_str = format!("  Moves: {}", app.board.moves);
    let mut pending = 0;
    if app.selected != -1 {
        if let Some(ref pre) = app.pre_select_board {
            pending = piece_dist(
                &app.board.pieces[app.selected as usize],
                &pre.pieces[app.selected as usize],
            );
        }
    }
    if pending > 0 {
        moves_str += &format!(" (+{})", pending);
    }
    if !app.history.is_empty() || pending > 0 {
        moves_str += "  (u: undo  U: restart)";
    }
    lines.push(Line::from(Span::styled(
        moves_str,
        Style::default().fg(DIM),
    )));

    // Hint display.
    if app.cheat_mode && !app.won {
        if app.hint_loading {
            lines.push(Line::from(Span::styled(
                "  Computing hint...",
                Style::default()
                    .fg(COLOR_HINT_FG)
                    .add_modifier(Modifier::BOLD),
            )));
        } else if let Some(ref hint) = app.hint {
            lines.push(Line::from(Span::styled(
                format!("  Hint: {}", hint.dir.arrow()),
                Style::default()
                    .fg(COLOR_HINT_FG)
                    .add_modifier(Modifier::BOLD),
            )));
        }
    }

    if app.won {
        lines.push(Line::raw(""));
        let mut win_spans: Vec<Span<'static>> = vec![Span::styled(
            format!("  YOU WIN in {} moves!", app.board.moves),
            Style::default().fg(COLOR_WIN).add_modifier(Modifier::BOLD),
        )];
        if app.board.moves == app.optimal {
            win_spans.push(Span::styled(
                "  PERFECT!",
                Style::default().fg(COLOR_WIN).add_modifier(Modifier::BOLD),
            ));
        }
        lines.push(Line::from(win_spans));
        lines.push(Line::from(Span::styled(
            "  u: undo  U: restart  n: new game  1/2/3: change difficulty  q: quit",
            Style::default().fg(DIM),
        )));
    } else {
        lines.push(Line::raw(""));
        if app.selected != -1 {
            lines.push(Line::from(Span::styled(
                "  Piece selected \u{2014} arrows: move  enter: accept  esc: cancel",
                Style::default().fg(color256(46)),
            )));
        } else {
            let mute_label = if app.sound.is_muted() {
                "m: unmute"
            } else {
                "m: mute"
            };
            lines.push(Line::from(Span::styled(
                format!("  Arrows/hjkl: move  Enter/Space: select  n: new  e: editor  g: league  ?: cheat  1/2/3: difficulty  {}  q: quit", mute_label),
                Style::default().fg(DIM),
            )));
        }
    }

    lines
}

fn render_league_play(app: &App) -> Vec<Line<'static>> {
    let mut lines: Vec<Line<'static>> = Vec::new();

    // Title.
    let mut title_spans: Vec<Span<'static>> = vec![
        Span::styled(
            "KLOTSKI PUZZLE",
            Style::default().fg(WHITE).add_modifier(Modifier::BOLD),
        ),
        Span::raw("  "),
        Span::styled(
            format!(" LEAGUE #{} ", app.league_idx + 1),
            Style::default()
                .fg(BLACK)
                .bg(COLOR_LEAGUE)
                .add_modifier(Modifier::BOLD),
        ),
    ];
    if app.optimal > 0 {
        title_spans.push(Span::styled(
            format!("  Best: {} moves", app.optimal),
            Style::default().fg(DIM),
        ));
    }
    lines.push(Line::from(title_spans));
    lines.push(Line::raw(""));

    render_board_lines(app, &mut lines);
    lines.push(Line::raw(""));

    // Status: moves.
    let mut moves_str = format!("  Moves: {}", app.board.moves);
    let mut pending = 0;
    if app.selected != -1 {
        if let Some(ref pre) = app.pre_select_board {
            pending = piece_dist(
                &app.board.pieces[app.selected as usize],
                &pre.pieces[app.selected as usize],
            );
        }
    }
    if pending > 0 {
        moves_str += &format!(" (+{})", pending);
    }
    if !app.history.is_empty() || pending > 0 {
        moves_str += "  (u: undo  U: restart)";
    }
    lines.push(Line::from(Span::styled(
        moves_str,
        Style::default().fg(DIM),
    )));

    // Current best score.
    if let Some(pd) = app.save_data.player_ref(&app.nickname) {
        if let Some(&prev_score) = pd.scores.get(&app.league_idx) {
            let spans: Vec<Span<'static>> = vec![
                Span::styled("  Current best: ", Style::default().fg(DIM)),
                Span::styled(
                    format!("{}/10", prev_score),
                    Style::default()
                        .fg(score_color(prev_score))
                        .add_modifier(Modifier::BOLD),
                ),
            ];
            lines.push(Line::from(spans));
        }
    }

    if app.won {
        lines.push(Line::raw(""));
        let mut win_spans: Vec<Span<'static>> = vec![Span::styled(
            format!("  YOU WIN in {} moves!", app.board.moves),
            Style::default().fg(COLOR_WIN).add_modifier(Modifier::BOLD),
        )];
        if app.board.moves == app.optimal {
            win_spans.push(Span::styled(
                "  PERFECT!",
                Style::default().fg(COLOR_WIN).add_modifier(Modifier::BOLD),
            ));
        }
        lines.push(Line::from(win_spans));

        // Score line.
        let mut score_spans: Vec<Span<'static>> = vec![
            Span::raw("  Score: "),
            Span::styled(
                format!("{}/10", app.league_score),
                Style::default()
                    .fg(score_color(app.league_score))
                    .add_modifier(Modifier::BOLD),
            ),
        ];
        if app.league_new_best {
            score_spans.push(Span::styled(
                "  NEW BEST!",
                Style::default().fg(COLOR_WIN).add_modifier(Modifier::BOLD),
            ));
        }
        lines.push(Line::from(score_spans));
        lines.push(Line::from(Span::styled(
            "  Enter: next puzzle  u: undo  U: restart  Esc: back to league  q: quit",
            Style::default().fg(DIM),
        )));
    } else {
        lines.push(Line::raw(""));
        if app.selected != -1 {
            lines.push(Line::from(Span::styled(
                "  Piece selected \u{2014} arrows: move  enter: accept  esc: cancel",
                Style::default().fg(color256(46)),
            )));
        } else {
            let mute_label = if app.sound.is_muted() {
                "m: unmute"
            } else {
                "m: mute"
            };
            lines.push(Line::from(Span::styled(
                format!("  Arrows/hjkl: move  Enter/Space: select  c: coords  {}  Esc: back to league  q: quit", mute_label),
                Style::default().fg(DIM),
            )));
        }
    }

    lines
}

fn render_name_input(app: &App) -> Vec<Line<'static>> {
    let mut lines: Vec<Line<'static>> = Vec::new();

    lines.push(Line::from(vec![
        Span::styled(
            "KLOTSKI PUZZLE",
            Style::default().fg(WHITE).add_modifier(Modifier::BOLD),
        ),
        Span::raw("  "),
        Span::styled(
            " LEAGUE ",
            Style::default()
                .fg(BLACK)
                .bg(COLOR_LEAGUE)
                .add_modifier(Modifier::BOLD),
        ),
    ]));
    lines.push(Line::raw(""));

    lines.push(Line::from(Span::styled(
        "  Enter your nickname:",
        Style::default().fg(WHITE).add_modifier(Modifier::BOLD),
    )));
    lines.push(Line::raw(""));

    lines.push(Line::from(vec![
        Span::raw("  > "),
        Span::styled(
            app.name_input.clone(),
            Style::default()
                .fg(color256(220))
                .add_modifier(Modifier::BOLD),
        ),
        Span::styled("_", Style::default().fg(DIM)),
    ]));
    lines.push(Line::raw(""));

    lines.push(Line::from(Span::styled(
        "  Enter: confirm  Esc: cancel  q: quit",
        Style::default().fg(DIM),
    )));

    lines
}

fn render_league(app: &App) -> Vec<Line<'static>> {
    let mut lines: Vec<Line<'static>> = Vec::new();
    let num_presets = app.presets.len();

    lines.push(Line::from(vec![
        Span::styled(
            "KLOTSKI PUZZLE",
            Style::default().fg(WHITE).add_modifier(Modifier::BOLD),
        ),
        Span::raw("  "),
        Span::styled(
            " LEAGUE ",
            Style::default()
                .fg(BLACK)
                .bg(COLOR_LEAGUE)
                .add_modifier(Modifier::BOLD),
        ),
        Span::raw("  "),
        Span::styled(
            app.nickname.clone(),
            Style::default()
                .fg(color256(220))
                .add_modifier(Modifier::BOLD),
        ),
    ]));
    lines.push(Line::raw(""));

    // Summary.
    let pd_total;
    let pd_completed;
    if let Some(pd) = app.save_data.player_ref(&app.nickname) {
        pd_total = pd.total_score();
        pd_completed = pd.completed();
    } else {
        pd_total = 0;
        pd_completed = 0;
    }
    lines.push(Line::from(Span::styled(
        format!(
            "  Score: {}/{}  Completed: {}/{}",
            pd_total,
            max_league_score(num_presets),
            pd_completed,
            num_presets,
        ),
        Style::default().fg(DIM),
    )));
    lines.push(Line::raw(""));

    // Puzzle list.
    let end = (app.league_scroll + LEAGUE_VISIBLE).min(num_presets);
    for i in app.league_scroll..end {
        let preset = &app.presets[i];
        let selected = i == app.league_idx;

        let score_opt = app
            .save_data
            .player_ref(&app.nickname)
            .and_then(|pd| pd.scores.get(&i).copied());

        let num_str = format!("{:3}.", i + 1);
        let opt_str = format!("({} moves)", preset.optimal);

        let mut spans: Vec<Span<'static>> = Vec::new();

        if selected {
            spans.push(Span::styled(
                "> ",
                Style::default()
                    .fg(COLOR_CURSOR)
                    .add_modifier(Modifier::BOLD),
            ));
        } else {
            spans.push(Span::raw("  "));
        }

        if let Some(score) = score_opt {
            spans.push(Span::styled(
                format!("  {}  ", num_str),
                Style::default().fg(WHITE),
            ));
            spans.push(Span::styled(
                format!("{}/10", score),
                Style::default().fg(score_color(score)),
            ));
            spans.push(Span::styled(
                format!("  {}", opt_str),
                Style::default().fg(DIM),
            ));
        } else {
            spans.push(Span::styled(
                format!("  {}  ", num_str),
                Style::default().fg(WHITE).add_modifier(Modifier::BOLD),
            ));
            spans.push(Span::styled(" --  ", Style::default().fg(DIM)));
            spans.push(Span::styled(
                format!("  {}", opt_str),
                Style::default().fg(DIM),
            ));
        }

        lines.push(Line::from(spans));
    }

    // Scroll indicators.
    if app.league_scroll > 0 {
        lines.push(Line::from(Span::styled(
            format!("  ... {} more above", app.league_scroll),
            Style::default().fg(DIM),
        )));
    }
    if end < num_presets {
        lines.push(Line::from(Span::styled(
            format!("  ... {} more below", num_presets - end),
            Style::default().fg(DIM),
        )));
    }

    lines.push(Line::raw(""));
    lines.push(Line::from(Span::styled(
        "  Arrows/jk: browse  Ctrl+u/d: page  g/G: home/end  Enter: play  Tab: leaderboard  @: switch player  Esc: back  q: quit",
        Style::default().fg(DIM),
    )));

    lines
}

fn render_leaderboard(app: &App) -> Vec<Line<'static>> {
    let mut lines: Vec<Line<'static>> = Vec::new();
    let num_presets = app.presets.len();

    lines.push(Line::from(vec![
        Span::styled(
            "KLOTSKI PUZZLE",
            Style::default().fg(WHITE).add_modifier(Modifier::BOLD),
        ),
        Span::raw("  "),
        Span::styled(
            " LEADERBOARD ",
            Style::default()
                .fg(BLACK)
                .bg(COLOR_LEAGUE)
                .add_modifier(Modifier::BOLD),
        ),
    ]));
    lines.push(Line::raw(""));

    let entries = app.save_data.leaderboard();

    if entries.is_empty() {
        lines.push(Line::from(Span::styled(
            "  No players yet.",
            Style::default().fg(DIM),
        )));
    } else {
        lines.push(Line::from(Span::styled(
            format!(
                "  {:<4}  {:<20}  {:>6}  {:>9}",
                "Rank", "Player", "Score", "Completed"
            ),
            Style::default().fg(DIM).add_modifier(Modifier::BOLD),
        )));
        lines.push(Line::from(Span::styled(
            format!("  {}", "-".repeat(45)),
            Style::default().fg(DIM).add_modifier(Modifier::BOLD),
        )));

        for (i, e) in entries.iter().enumerate() {
            let is_current = e.name == app.nickname;
            let name_style = if is_current {
                Style::default()
                    .fg(color256(220))
                    .add_modifier(Modifier::BOLD)
            } else {
                Style::default().fg(WHITE)
            };

            lines.push(Line::from(vec![
                Span::styled(
                    format!("  {:<4}", format!("#{}", i + 1)),
                    Style::default().fg(COLOR_RANK).add_modifier(Modifier::BOLD),
                ),
                Span::raw("  "),
                Span::styled(format!("{:<20}", e.name), name_style),
                Span::raw("  "),
                Span::styled(format!("{:>6}", e.total), Style::default().fg(COLOR_SCORED)),
                Span::raw("  "),
                Span::styled(
                    format!("{:>5}/{}", e.completed, num_presets),
                    Style::default().fg(DIM),
                ),
            ]));
        }
    }

    lines.push(Line::raw(""));
    lines.push(Line::from(Span::styled(
        "  Esc/Tab: back to league  q: quit",
        Style::default().fg(DIM),
    )));

    lines
}

fn render_editor(app: &App) -> Vec<Line<'static>> {
    let mut lines: Vec<Line<'static>> = Vec::new();

    lines.push(Line::from(vec![
        Span::styled(
            "KLOTSKI PUZZLE",
            Style::default().fg(WHITE).add_modifier(Modifier::BOLD),
        ),
        Span::raw("  "),
        Span::styled(
            " EDITOR ",
            Style::default()
                .fg(BLACK)
                .bg(COLOR_EDITOR)
                .add_modifier(Modifier::BOLD),
        ),
    ]));
    lines.push(Line::raw(""));

    // Piece selector.
    let kinds = [
        PieceKind::Large,
        PieceKind::Vertical,
        PieceKind::Horizontal,
        PieceKind::Small,
    ];
    let mut sel_spans: Vec<Span<'static>> = vec![Span::raw("  Piece: ")];
    for (i, &k) in kinds.iter().enumerate() {
        let mut style = Style::default().fg(piece_fg(k));
        if k == app.edit_piece {
            style = style.add_modifier(Modifier::BOLD | Modifier::UNDERLINED);
        }
        sel_spans.push(Span::styled(edit_piece_label(k).to_string(), style));
        if i < kinds.len() - 1 {
            sel_spans.push(Span::styled("  ", Style::default().fg(DIM)));
        }
    }
    lines.push(Line::from(sel_spans));

    // Piece counts.
    let (l, v, h, s) = count_pieces(&app.board);
    let occupied: usize = app.board.pieces.iter().map(|p| p.cells().len()).sum();
    let empty = BOARD_W * BOARD_H - occupied;
    lines.push(Line::from(Span::styled(
        format!("  L:{}  V:{}  H:{}  S:{}  Empty:{}", l, v, h, s, empty),
        Style::default().fg(DIM),
    )));
    lines.push(Line::raw(""));

    render_board_lines(app, &mut lines);
    lines.push(Line::raw(""));

    // Error.
    if !app.edit_error.is_empty() {
        lines.push(Line::from(Span::styled(
            format!("  {}", app.edit_error),
            Style::default()
                .fg(COLOR_ERROR)
                .add_modifier(Modifier::BOLD),
        )));
    }

    if app.edit_solving {
        lines.push(Line::from(Span::styled(
            "  Checking solvability...",
            Style::default().fg(DIM),
        )));
    }

    lines.push(Line::raw(""));
    lines.push(Line::from(Span::styled(
        "  Arrows/hjkl: move cursor  Tab: cycle piece  Enter/Space: place",
        Style::default().fg(DIM),
    )));
    lines.push(Line::from(Span::styled(
        "  x/Backspace: remove  r: clear  c: coords  p: play  Esc: cancel  q: quit",
        Style::default().fg(DIM),
    )));

    lines
}

/// Renders the board grid into a list of styled lines.
fn render_board_lines(app: &App, lines: &mut Vec<Line<'static>>) {
    let grid = app.board.occupancy();

    // Ghost piece for editor preview.
    let mut ghost: Option<Piece> = None;
    let mut ghost_grid = [[false; BOARD_H]; BOARD_W];
    if app.mode == GameMode::Editor && app.board.piece_at(app.cursor_x, app.cursor_y) == -1 {
        let candidate = Piece::new(app.edit_piece, app.cursor_x, app.cursor_y);
        if app.board.can_place(&candidate) {
            for (cx, cy) in candidate.cells() {
                ghost_grid[cx as usize][cy as usize] = true;
            }
            ghost = Some(candidate);
        }
    }

    // Column headers (coords mode).
    if app.show_coords {
        let mut spans: Vec<Span<'static>> = vec![Span::raw("   ")];
        for x in 0..BOARD_W {
            spans.push(Span::styled(format!("  {}  ", x), Style::default().fg(DIM)));
            if x < BOARD_W - 1 {
                spans.push(Span::raw(" "));
            }
        }
        lines.push(Line::from(spans));
    }

    // Top border.
    let mut top = String::from("  \u{250c}");
    for x in 0..BOARD_W {
        top.push_str("\u{2500}\u{2500}\u{2500}\u{2500}\u{2500}");
        if x < BOARD_W - 1 {
            top.push('\u{252c}');
        }
    }
    top.push('\u{2510}');
    lines.push(Line::raw(top));

    for y in 0..BOARD_H {
        // Two lines per cell.
        for line_num in 0..2 {
            let mut spans: Vec<Span<'static>> = Vec::new();

            if app.show_coords && line_num == 0 {
                spans.push(Span::styled(format!("{}", y), Style::default().fg(DIM)));
                spans.push(Span::raw(" \u{2502}"));
            } else {
                spans.push(Span::raw("  \u{2502}"));
            }

            for x in 0..BOARD_W {
                let idx = grid[x][y];
                let cell_spans =
                    render_cell(app, x as i32, y as i32, idx, line_num, &ghost, &ghost_grid);
                spans.extend(cell_spans);

                if x < BOARD_W - 1 {
                    let same_real = idx != -1 && (x + 1) < BOARD_W && grid[x + 1][y] == idx;
                    let same_ghost = ghost.is_some() && ghost_grid[x][y] && ghost_grid[x + 1][y];
                    if same_real || same_ghost {
                        if same_real {
                            let bg = cell_bg(app, idx);
                            let style = if let Some(bg_color) = bg {
                                Style::default().bg(bg_color)
                            } else {
                                Style::default()
                            };
                            spans.push(Span::styled(" ", style));
                        } else {
                            spans.push(Span::raw(" "));
                        }
                    } else {
                        spans.push(Span::raw("\u{2502}"));
                    }
                }
            }
            spans.push(Span::raw("\u{2502}"));
            lines.push(Line::from(spans));
        }

        // Horizontal border between rows.
        if y < BOARD_H - 1 {
            let mut spans: Vec<Span<'static>> = vec![Span::raw("  \u{251c}")];

            for x in 0..BOARD_W {
                let top_idx = grid[x][y];
                let bot_idx = grid[x][y + 1];
                let same_real = top_idx != -1 && top_idx == bot_idx;
                let same_ghost = ghost.is_some() && ghost_grid[x][y] && ghost_grid[x][y + 1];
                if same_real || same_ghost {
                    if same_real {
                        let bg = cell_bg(app, top_idx);
                        let style = if let Some(bg_color) = bg {
                            Style::default().bg(bg_color)
                        } else {
                            Style::default()
                        };
                        spans.push(Span::styled("     ", style));
                    } else {
                        spans.push(Span::raw("     "));
                    }
                } else {
                    spans.push(Span::raw("\u{2500}\u{2500}\u{2500}\u{2500}\u{2500}"));
                }

                if x < BOARD_W - 1 {
                    let tl = grid[x][y];
                    let tr = grid[x + 1][y];
                    let bl = grid[x][y + 1];
                    let br = grid[x + 1][y + 1];
                    if tl != -1 && tl == tr && tl == bl && tl == br {
                        let bg = cell_bg(app, tl);
                        let style = if let Some(bg_color) = bg {
                            Style::default().bg(bg_color)
                        } else {
                            Style::default()
                        };
                        spans.push(Span::styled(" ", style));
                    } else {
                        spans.push(Span::raw("\u{253c}"));
                    }
                }
            }
            spans.push(Span::raw("\u{2524}"));
            lines.push(Line::from(spans));
        }
    }

    // Bottom border.
    let mut bot = String::from("  \u{2514}");
    for x in 0..BOARD_W {
        bot.push_str("\u{2500}\u{2500}\u{2500}\u{2500}\u{2500}");
        if x < BOARD_W - 1 {
            bot.push('\u{2534}');
        }
    }
    bot.push('\u{2518}');
    lines.push(Line::raw(bot));
}

/// Renders a single cell (5 chars wide), returning a list of spans.
fn render_cell(
    app: &App,
    x: i32,
    y: i32,
    idx: i32,
    line_num: usize,
    ghost: &Option<Piece>,
    ghost_grid: &[[bool; BOARD_H]; BOARD_W],
) -> Vec<Span<'static>> {
    let is_cursor = x == app.cursor_x && y == app.cursor_y;
    let is_hinted = app.cheat_mode
        && app.hint.is_some()
        && idx != -1
        && idx as usize == app.hint.as_ref().unwrap().piece_index
        && !app.won;
    let is_ghost = ghost.is_some() && ghost_grid[x as usize][y as usize];

    // Ghost preview cell.
    if is_ghost && idx == -1 {
        let ghost_piece = ghost.as_ref().unwrap();
        let label = match ghost_piece.kind {
            PieceKind::Small => {
                if line_num == 0 {
                    "  s  "
                } else {
                    "     "
                }
            }
            PieceKind::Vertical => {
                if line_num == 0 && y == ghost_piece.y {
                    "  m  "
                } else if line_num == 1 && y == ghost_piece.y + 1 {
                    "  m  "
                } else {
                    "     "
                }
            }
            PieceKind::Horizontal => {
                if line_num == 0 {
                    "  m  "
                } else {
                    "     "
                }
            }
            PieceKind::Large => {
                if line_num == 0 {
                    "  L  "
                } else {
                    "     "
                }
            }
        };

        if is_cursor && line_num == 0 {
            return vec![Span::styled(
                format!(" [{}] ", edit_piece_short(ghost_piece.kind)),
                Style::default()
                    .fg(COLOR_CURSOR)
                    .add_modifier(Modifier::BOLD),
            )];
        }
        return vec![Span::styled(
            label.to_string(),
            Style::default().fg(COLOR_GHOST),
        )];
    }

    let (label, fg) = if idx == -1 {
        // Empty cell. Show dim "L" on target cells.
        let is_target = x >= 1 && x <= 2 && y >= 3 && y <= 4;
        if is_target && line_num == 0 && !app.won {
            ("  L  ", COLOR_LOCKED)
        } else {
            ("     ", COLOR_EMPTY)
        }
    } else {
        let p = &app.board.pieces[idx as usize];
        let fg_color = if p.kind == PieceKind::Large && app.won {
            COLOR_WIN
        } else {
            piece_fg(p.kind)
        };
        let lbl = match p.kind {
            PieceKind::Small => {
                if line_num == 0 {
                    "  s  "
                } else {
                    "     "
                }
            }
            PieceKind::Vertical => {
                if line_num == 0 && y == p.y {
                    "  m  "
                } else if line_num == 1 && y == p.y + 1 {
                    "  m  "
                } else {
                    "     "
                }
            }
            PieceKind::Horizontal => {
                if line_num == 0 {
                    "  m  "
                } else {
                    "     "
                }
            }
            PieceKind::Large => {
                if line_num == 0 {
                    "  L  "
                } else {
                    "     "
                }
            }
        };
        (lbl, fg_color)
    };

    let mut style = Style::default().fg(fg);

    // Apply piece background.
    if idx != -1 {
        if let Some(bg_color) = cell_bg(app, idx) {
            style = style.bg(bg_color);
        }
    }

    // Show direction arrow on line 1 of the hinted piece's origin cell.
    if is_hinted && line_num == 1 {
        let p = &app.board.pieces[idx as usize];
        if x == p.x && y == p.y {
            let hint = app.hint.as_ref().unwrap();
            let mut arrow_style = Style::default()
                .fg(COLOR_HINT_FG)
                .add_modifier(Modifier::BOLD);
            if let Some(bg_color) = cell_bg(app, idx) {
                arrow_style = arrow_style.bg(bg_color);
            }
            return vec![Span::styled(
                format!("  {}  ", hint.dir.arrow()),
                arrow_style,
            )];
        }
    }

    if is_cursor && !app.won {
        if line_num == 0 {
            let mut cursor_style = Style::default()
                .fg(COLOR_CURSOR)
                .add_modifier(Modifier::BOLD);
            if idx != -1 {
                if let Some(bg_color) = cell_bg(app, idx) {
                    cursor_style = cursor_style.bg(bg_color);
                }
            }
            let cursor_label = if app.mode == GameMode::Editor && idx == -1 {
                format!("[{}]", edit_piece_short(app.edit_piece))
            } else {
                "[*]".to_string()
            };
            return vec![Span::styled(format!(" {} ", cursor_label), cursor_style)];
        }
        style = style.add_modifier(Modifier::BOLD);
    }

    vec![Span::styled(label.to_string(), style)]
}
