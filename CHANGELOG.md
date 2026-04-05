# Changelog

## Unreleased

### Sound
- Background theme music: Korobeiniki (Tetris Theme A) with chiptune melody and pulsing bass, looping continuously
- Victory chime: ascending C5-E5-G5-C6 success sound on puzzle completion
- All audio generated programmatically at runtime — no embedded assets
- `m` key to toggle mute/unmute (global, works in all modes except name input)
- Help text dynamically shows `m: mute` or `m: unmute` based on current state
- Go: audio playback via `gopxl/beep` with `ebitengine/oto` backend; fails silently if audio is unavailable
- Rust: audio playback via `rodio` with ALSA backend; fails silently if audio is unavailable
- `go run ./cmd/gensound` tool to export sounds as WAV files for previewing or tweaking

### Rust Port
- Full rewrite of the game in Rust using `ratatui` + `crossterm`
- Feature parity with the Go version: all 6 game modes (FreePlay, Editor, NameInput, League, LeaguePlay, Leaderboard)
- BFS solver with canonical state encoding for interchangeable same-kind pieces
- Background thread for board generation, hint computation, and editor solvability checks via `mpsc::channel`
- All 620 league presets ported
- JSON save/load persistence compatible with the Go version
- Full TUI rendering with 256-color styling, box-drawing borders, ghost piece preview, and hint arrows
- Release binary: 1.1MB (vs Go's 5.1MB)

### Project Reorganization
- Separated Go and Rust codebases into `go/` and `rust/` subdirectories
- Updated `.gitignore` for the new layout

### League Mode
- New league mode (`g` key from free play) with 620 pre-generated puzzles sorted by increasing difficulty (1-179 optimal moves)
- All puzzles unlocked from the start — no linear progression or locking
- Puzzle browser with scroll, showing puzzle number, optimal move count, and score
- Cursor starts at the first unscored puzzle when entering the browser
- Scoring: 10 points for optimal solution, scaling proportionally down to minimum 1 point for any solve
- Score colors: 10-shade gradient from red (1) through orange/yellow to bright green (10)
- Browser navigation: Arrows/jk for single step, `Ctrl+u`/`Ctrl+d` for page jump (15 items), `g`/`G` for home/end
- Nickname entry for player identity, with `@` to switch players
- Persistent save data at `~/.klotski-puzzle/save.json` — tracks scores per player, remembers last player
- Leaderboard (`Tab` in league browser) — all players ranked by total score
- League play uses same controls as free play but restricts mode-switching keys; Esc returns to browser
- On win, score auto-saved if it's a new best; "NEW BEST!" indicator shown
- After winning, `Enter` advances to the next puzzle; `Esc` returns to the browser; `u`/`U` to undo/restart for a better score
- Cheat mode disabled in league play — no hints allowed

### Board Editor
- New editor mode (`e` key) for creating custom puzzles from scratch
- Place pieces on an empty 4x5 grid: Large (2x2), Vertical (1x2), Horizontal (2x1), Small (1x1)
- Cycle piece types with `Tab`, place with `Enter`/`Space`, remove with `x`/`Backspace`/`Delete`
- Ghost preview shows a dim outline of the piece before placing it
- Cursor displays piece-type indicator (`[L]`, `[V]`, `[H]`, `[s]`) on empty cells
- Piece count display (L/V/H/S/Empty) updates in real time
- `r` to clear the board, `c` to toggle coordinate labels
- `p` to validate and play: checks for exactly 1 Large piece, at least 2 empty cells, and runs BFS to confirm solvability
- Unsolvable boards show an error message so you can adjust and retry
- Custom boards play with a "Custom" badge and computed optimal move count
- `n` exits custom mode and returns to random puzzle generation
- `Esc` cancels the editor and generates a new random board

### Visual
- Colorized game stones: each piece type now has a subtle dark-tinted background matching its color (amber for small, navy for vertical, green for horizontal, red for large)
- Selected pieces glow brighter in their own color instead of a generic green highlight
- Multi-cell pieces (vertical, horizontal, large) render as seamless colored tiles with merged internal borders
- Win state: large piece background transitions from red to green on victory
- Target cell indicator: empty cells in the 2x2 goal area (cols 1-2, rows 3-4) show a dim "L"
- Score colors: 10-shade red-to-green gradient matching score value

### Cheat Mode
- Toggle with `?` to show the optimal next move
- Hinted piece highlighted with purple background and direction arrow
- "CHEAT" badge displayed in the title bar
- Hints recomputed automatically after each move, undo, or new game
- Async BFS computation prevents UI blocking

### Undo / Restart
- `u` undo and `U` restart now work in the win state
- `U` resets the board to its starting position in a single keypress
