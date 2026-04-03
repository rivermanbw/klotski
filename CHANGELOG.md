# Changelog

## Unreleased

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
