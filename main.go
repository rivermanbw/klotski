package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Colors for piece types.
var (
	colorSmall  = lipgloss.Color("214") // orange
	colorMedium = lipgloss.Color("39")  // blue
	colorLarge  = lipgloss.Color("196") // red
	colorEmpty  = lipgloss.Color("236") // dark gray
	colorCursor = lipgloss.Color("226") // yellow
	colorWin    = lipgloss.Color("82")  // bright green

	colorEasy   = lipgloss.Color("82")  // green
	colorMedDif = lipgloss.Color("214") // orange
	colorHard   = lipgloss.Color("196") // red
	colorHint   = lipgloss.Color("53")  // dark purple for hint background
	colorHintFg = lipgloss.Color("213") // bright pink for hint arrow/text
)

func diffColor(d Difficulty) lipgloss.Color {
	switch d {
	case Easy:
		return colorEasy
	case Medium:
		return colorMedDif
	case Hard:
		return colorHard
	}
	return colorEasy
}

// boardReadyMsg is sent when board generation (done in background) completes.
type boardReadyMsg struct {
	board   *Board
	optimal int
	diff    Difficulty
}

// hintReadyMsg is sent when the background hint computation completes.
type hintReadyMsg struct {
	hint *Hint
	seq  int // sequence number to discard stale results
}

type model struct {
	board      *Board
	cursorX    int
	cursorY    int
	selected   int // index of selected piece, or -1
	history    []*Board
	won        bool
	difficulty Difficulty
	optimal    int  // minimum moves to solve (from BFS at generation time)
	loading    bool // true while generating a new board
	showCoords bool // toggle coordinate labels on the board

	cheatMode   bool  // whether cheat mode is active
	hint        *Hint // current hint (nil if unavailable or not computed)
	hintLoading bool  // true while computing a hint
	hintSeq     int   // sequence counter to discard stale hint results
}

func initialModel() model {
	return model{
		cursorX:    0,
		cursorY:    0,
		selected:   -1,
		difficulty: Easy,
		loading:    true,
	}
}

func generateBoardCmd(diff Difficulty) tea.Cmd {
	return func() tea.Msg {
		b, opt := NewRandomBoard(diff)
		return boardReadyMsg{board: b, optimal: opt, diff: diff}
	}
}

func computeHintCmd(b *Board, seq int) tea.Cmd {
	return func() tea.Msg {
		return hintReadyMsg{hint: SolveNextMove(b), seq: seq}
	}
}

func dirArrow(dir Direction) string {
	switch dir {
	case Up:
		return "↑"
	case Down:
		return "↓"
	case Left:
		return "←"
	case Right:
		return "→"
	}
	return "?"
}

func (m model) Init() tea.Cmd {
	return generateBoardCmd(m.difficulty)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case boardReadyMsg:
		m.board = msg.board
		m.optimal = msg.optimal
		m.difficulty = msg.diff
		m.loading = false
		m.won = false
		m.selected = -1
		m.history = nil
		m.cursorX = 0
		m.cursorY = 0
		m.hint = nil
		m.hintLoading = false
		if m.cheatMode {
			m.hintSeq++
			m.hintLoading = true
			return m, computeHintCmd(m.board.Clone(), m.hintSeq)
		}
		return m, nil

	case hintReadyMsg:
		if msg.seq == m.hintSeq && m.cheatMode {
			m.hint = msg.hint
			m.hintLoading = false
		}
		return m, nil

	case tea.KeyMsg:
		key := msg.String()

		// Quit — always available.
		if key == "q" || key == "ctrl+c" {
			return m, tea.Quit
		}

		// Ignore other keys while generating.
		if m.loading {
			return m, nil
		}

		// Cycle difficulty: 1/2/3.
		if key == "1" || key == "2" || key == "3" {
			var d Difficulty
			switch key {
			case "1":
				d = Easy
			case "2":
				d = Medium
			case "3":
				d = Hard
			}
			m.difficulty = d
			m.loading = true
			return m, generateBoardCmd(d)
		}

		// New game (same difficulty).
		if key == "n" {
			m.loading = true
			return m, generateBoardCmd(m.difficulty)
		}

		// Toggle coordinate labels.
		if key == "c" {
			m.showCoords = !m.showCoords
			return m, nil
		}

		// Toggle cheat mode.
		if key == "?" {
			m.cheatMode = !m.cheatMode
			if m.cheatMode && !m.won {
				m.hintSeq++
				m.hint = nil
				m.hintLoading = true
				return m, computeHintCmd(m.board.Clone(), m.hintSeq)
			}
			m.hint = nil
			m.hintLoading = false
			return m, nil
		}

		// Undo.
		if key == "u" && len(m.history) > 0 && !m.won {
			m.board = m.history[len(m.history)-1]
			m.history = m.history[:len(m.history)-1]
			m.selected = -1
			if m.cheatMode {
				m.hintSeq++
				m.hint = nil
				m.hintLoading = true
				return m, computeHintCmd(m.board.Clone(), m.hintSeq)
			}
			return m, nil
		}

		if m.won {
			return m, nil
		}

		// Deselect.
		if key == "esc" {
			m.selected = -1
			return m, nil
		}

		// Select / deselect piece.
		if key == "enter" || key == " " {
			if m.selected != -1 {
				m.selected = -1
			} else {
				idx := m.board.PieceAt(m.cursorX, m.cursorY)
				if idx != -1 {
					m.selected = idx
				}
			}
			return m, nil
		}

		// Directional input.
		var dir Direction
		var isDir bool
		switch key {
		case "up", "k":
			dir, isDir = Up, true
		case "down", "j":
			dir, isDir = Down, true
		case "left", "h":
			dir, isDir = Left, true
		case "right", "l":
			dir, isDir = Right, true
		}

		if isDir {
			if m.selected != -1 {
				snapshot := m.board.Clone()
				if m.board.Move(m.selected, dir) {
					m.history = append(m.history, snapshot)
					p := m.board.Pieces[m.selected]
					m.cursorX = p.X
					m.cursorY = p.Y
					if m.board.IsWon() {
						m.won = true
						m.selected = -1
						m.hint = nil
						m.hintLoading = false
					} else if m.cheatMode {
						m.hintSeq++
						m.hint = nil
						m.hintLoading = true
						return m, computeHintCmd(m.board.Clone(), m.hintSeq)
					}
				}
			} else {
				dx, dy := dirDelta(dir)
				nx, ny := m.cursorX+dx, m.cursorY+dy
				if nx >= 0 && nx < BoardW && ny >= 0 && ny < BoardH {
					m.cursorX = nx
					m.cursorY = ny
				}
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	var sb strings.Builder

	// Title.
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255")).
		Render("KLOTSKI PUZZLE")
	sb.WriteString(title)

	// Difficulty badge.
	sb.WriteString("  ")
	badge := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(diffColor(m.difficulty)).
		Padding(0, 1).
		Render(m.difficulty.String())
	sb.WriteString(badge)

	if m.optimal > 0 {
		optStr := fmt.Sprintf("  Best: %d moves", m.optimal)
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(optStr))
	}

	if m.cheatMode {
		sb.WriteString("  ")
		cheatBadge := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("0")).
			Background(colorHintFg).
			Padding(0, 1).
			Render("CHEAT")
		sb.WriteString(cheatBadge)
	}
	sb.WriteString("\n\n")

	// Loading screen.
	if m.loading {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Render("  Generating puzzle..."))
		sb.WriteString("\n")
		return sb.String()
	}

	grid := m.board.occupancy()

	// Column headers (when coordinate system is enabled).
	if m.showCoords {
		coordStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		sb.WriteString("   ")
		for x := range BoardW {
			sb.WriteString(coordStyle.Render(fmt.Sprintf("  %d  ", x)))
			if x < BoardW-1 {
				sb.WriteString(" ")
			}
		}
		sb.WriteString("\n")
	}

	// Top border.
	sb.WriteString("  ┌")
	for x := range BoardW {
		sb.WriteString("─────")
		if x < BoardW-1 {
			sb.WriteString("┬")
		}
	}
	sb.WriteString("┐\n")

	for y := range BoardH {
		// Two lines per cell for visual height.
		for line := 0; line < 2; line++ {
			// Row label on first line of each row.
			if m.showCoords && line == 0 {
				coordStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
				sb.WriteString(coordStyle.Render(fmt.Sprintf("%d", y)))
				sb.WriteString(" │")
			} else {
				sb.WriteString("  │")
			}
			for x := range BoardW {
				idx := grid[x][y]
				cellStr := m.renderCell(x, y, idx, line)
				sb.WriteString(cellStr)
				if x < BoardW-1 {
					if idx != -1 && x+1 < BoardW && grid[x+1][y] == idx {
						sb.WriteString(" ")
					} else {
						sb.WriteString("│")
					}
				}
			}
			sb.WriteString("│\n")
		}

		// Horizontal border between rows.
		if y < BoardH-1 {
			sb.WriteString("  ├")
			for x := range BoardW {
				top := grid[x][y]
				bot := grid[x][y+1]
				if top != -1 && top == bot {
					sb.WriteString("     ")
				} else {
					sb.WriteString("─────")
				}
				if x < BoardW-1 {
					sb.WriteString("┼")
				}
			}
			sb.WriteString("┤\n")
		}
	}

	// Bottom border.
	sb.WriteString("  └")
	for x := range BoardW {
		sb.WriteString("─────")
		if x < BoardW-1 {
			sb.WriteString("┴")
		}
	}
	sb.WriteString("┘\n")

	sb.WriteString("\n")

	// Status: moves.
	movesStr := fmt.Sprintf("  Moves: %d", m.board.Moves)
	if len(m.history) > 0 {
		movesStr += "  (u to undo)"
	}
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(movesStr))
	sb.WriteString("\n")

	// Hint display (cheat mode).
	if m.cheatMode && !m.won {
		hintStyle := lipgloss.NewStyle().Foreground(colorHintFg).Bold(true)
		if m.hintLoading {
			sb.WriteString(hintStyle.Render("  Computing hint..."))
			sb.WriteString("\n")
		} else if m.hint != nil {
			sb.WriteString(hintStyle.Render(fmt.Sprintf("  Hint: %s", dirArrow(m.hint.Dir))))
			sb.WriteString("\n")
		}
	}

	if m.won {
		winStyle := lipgloss.NewStyle().Bold(true).Foreground(colorWin)
		sb.WriteString("\n")
		sb.WriteString(winStyle.Render(fmt.Sprintf("  YOU WIN in %d moves!", m.board.Moves)))
		if m.board.Moves == m.optimal {
			sb.WriteString(winStyle.Render("  PERFECT!"))
		}
		sb.WriteString("\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("  n: new game  1/2/3: change difficulty  q: quit"))
		sb.WriteString("\n")
	} else {
		sb.WriteString("\n")
		if m.selected != -1 {
			selStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
			sb.WriteString(selStyle.Render("  Piece selected — arrow keys to move, esc to deselect"))
		} else {
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(
				"  Arrows/hjkl: move  Enter/Space: select  n: new  c: coords  ?: cheat  1/2/3: difficulty  q: quit"))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (m model) renderCell(x, y, idx, line int) string {
	isCursor := (x == m.cursorX && y == m.cursorY)
	isSelected := (idx != -1 && idx == m.selected)
	isHinted := m.cheatMode && m.hint != nil && idx != -1 && idx == m.hint.PieceIndex && !m.won

	var label string
	var fg lipgloss.Color

	if idx == -1 {
		label = "     "
		fg = colorEmpty
	} else {
		p := m.board.Pieces[idx]
		switch p.Kind {
		case Small:
			fg = colorSmall
			if line == 0 {
				label = "  s  "
			} else {
				label = "     "
			}
		case Vertical:
			fg = colorMedium
			if line == 0 && y == p.Y {
				label = "  m  "
			} else if line == 1 && y == p.Y+1 {
				label = "  m  "
			} else {
				label = "     "
			}
		case Horizontal:
			fg = colorMedium
			if line == 0 {
				label = "  m  "
			} else {
				label = "     "
			}
		case Large:
			fg = colorLarge
			if m.won {
				fg = colorWin
			}
			if line == 0 {
				label = "  L  "
			} else {
				label = "     "
			}
		}
	}

	style := lipgloss.NewStyle().Foreground(fg)

	if isHinted && !isSelected {
		style = style.Background(colorHint)
	}
	if isSelected {
		style = style.Background(lipgloss.Color("22"))
	}

	// Show direction arrow on line 1 of the hinted piece's origin cell.
	if isHinted && line == 1 && x == m.board.Pieces[idx].X && y == m.board.Pieces[idx].Y {
		arrowStyle := lipgloss.NewStyle().Foreground(colorHintFg).Bold(true)
		if isSelected {
			arrowStyle = arrowStyle.Background(lipgloss.Color("22"))
		} else {
			arrowStyle = arrowStyle.Background(colorHint)
		}
		return arrowStyle.Render(fmt.Sprintf("  %s  ", dirArrow(m.hint.Dir)))
	}

	if isCursor && !m.won {
		if line == 0 {
			cursorStyle := lipgloss.NewStyle().Foreground(colorCursor).Bold(true)
			if isSelected {
				cursorStyle = cursorStyle.Background(lipgloss.Color("22"))
			} else if isHinted {
				cursorStyle = cursorStyle.Background(colorHint)
			}
			return cursorStyle.Render(" [*] ")
		}
		style = style.Bold(true)
	}

	return style.Render(label)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
