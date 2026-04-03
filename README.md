# Klotski Puzzle

A terminal-based sliding block puzzle game written in Go.

## Game Overview

Klotski is a classic sliding block puzzle played on a **4-wide by 5-tall** grid.
The objective is to move the large 2x2 block to the bottom-center of the board
by sliding pieces horizontally or vertically. There are only **two empty cells**
on the board at any time, so space is tight and every move counts.

## Pieces

| Piece Type | Size      | Count | Symbol |
|------------|-----------|-------|--------|
| Small      | 1x1       | 4     | `s`    |
| Medium     | 1x2 / 2x1 | 5     | `m`    |
| Large      | 2x2       | 1     | `L`    |

**Total occupied cells:** 4(1) + 5(2) + 1(4) = **18 out of 20 cells**

This leaves exactly **2 empty cells** for maneuvering.

## Rules

1. Pieces can only be moved **horizontally or vertically** (no diagonal moves).
2. A piece can only move into **empty space** — pieces cannot overlap.
3. A piece moves **one cell at a time** in the chosen direction.
4. The game is won when the **large 2x2 block** occupies the bottom-center
   position: columns 1-2, rows 3-4 (0-indexed).

## Controls

| Key              | Action                          |
|------------------|---------------------------------|
| Arrow keys       | Move cursor to select a piece   |
| `h/j/k/l`       | Move cursor (vim-style)         |
| Enter / Space    | Select piece under cursor       |
| Arrow keys       | Move selected piece             |
| `h/j/k/l`       | Move selected piece (vim-style) |
| Escape           | Deselect piece                  |
| `n`              | New game (same difficulty)      |
| `1`              | New game — Easy                 |
| `2`              | New game — Medium               |
| `3`              | New game — Hard                 |
| `u`              | Undo last move                  |
| `q` / Ctrl+C     | Quit                            |

## Difficulty

Each generated puzzle is solved by the engine using BFS to determine the
minimum number of moves required. The difficulty level controls which range
of optimal moves is accepted:

| Level  | Optimal Moves | Description                          |
|--------|---------------|--------------------------------------|
| Easy   | 1 – 39        | Good for learning the mechanics      |
| Medium | 40 – 79       | Requires planning several moves ahead|
| Hard   | 80+           | Challenging; may need 100+ moves     |

The current difficulty and the optimal (best possible) move count are
displayed next to the title. When you win, the game tells you whether
you achieved a perfect solution.

## Board Layout

```
  0   1   2   3      <- columns
+---+---+---+---+
|   |   |   |   |  0  <- rows
+---+---+---+---+
|   |   |   |   |  1
+---+---+---+---+
|   |   |   |   |  2
+---+---+---+---+
|   |   |   |   |  3
+---+---+---+---+
|   |   |   |   |  4
+---+---+---+---+
```

### Win Position

The large 2x2 block must reach the bottom-center:

```
+---+---+---+---+
|   |   |   |   |
+---+---+---+---+
|   |   |   |   |
+---+---+---+---+
|   |   |   |   |
+---+---+---+---+
|   | L | L |   |
+---+---+---+---+
|   | L | L |   |
+---+---+---+---+
```

## Building and Running

```bash
go build -o puzzle .
./puzzle
```

Or run directly:

```bash
go run .
```

## How to Play

1. Launch the game. A randomized starting layout is generated.
2. Use the arrow keys (or `h/j/k/l`) to move the cursor over the board.
3. Press **Enter** or **Space** to select the piece under the cursor.
4. Use the arrow keys (or `h/j/k/l`) to slide the selected piece.
5. Press **Escape** to deselect.
6. Slide the large block to the bottom-center to win.
