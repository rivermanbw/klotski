# Changelog

## Unreleased

### League Mode
- New league mode (`g` key from free play) with 320 pre-generated puzzles sorted by increasing difficulty (1-173 optimal moves)
- Puzzle browser with scroll, showing puzzle number, optimal move count, score, and lock status
- Linear progression — completing puzzle N unlocks puzzle N+1; replay any scored puzzle to improve
- Scoring: 10 points for optimal solution, scaling proportionally down to minimum 1 point for any solve
- Nickname entry for player identity, with `@` to switch players
- Persistent save data at `~/.klotski-puzzle/save.json` — tracks scores per player, remembers last player
- Leaderboard (`Tab` in league browser) — all players ranked by total score
- League play uses same controls as free play but restricts mode-switching keys; Esc returns to browser
- On win, score auto-saved if it's a new best; "NEW BEST!" indicator shown
- After winning, `Enter` advances to the next puzzle; `Esc` returns to the browser
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

### Cheat Mode
- Toggle with `?` to show the optimal next move
- Hinted piece highlighted with purple background and direction arrow
- "CHEAT" badge displayed in the title bar
- Hints recomputed automatically after each move, undo, or new game
- Async BFS computation prevents UI blocking

### Undo / Restart
- `u` undo and `U` restart now work in the win state
- `U` resets the board to its starting position in a single keypress
