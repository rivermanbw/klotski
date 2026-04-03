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

		// Undo.
		if key == "u" && len(m.history) > 0 && !m.won {
			m.board = m.history[len(m.history)-1]
			m.history = m.history[:len(m.history)-1]
			m.selected = -1
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
			sb.WriteString("  │")
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
				"  Arrows/hjkl: move  Enter/Space: select  n: new  1/2/3: difficulty  q: quit"))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (m model) renderCell(x, y, idx, line int) string {
	isCursor := (x == m.cursorX && y == m.cursorY)
	isSelected := (idx != -1 && idx == m.selected)

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
			if line == 1 && x == p.X && y == p.Y {
				label = "  L  "
			} else if line == 1 && x == p.X+1 && y == p.Y {
				label = "  L  "
			} else {
				label = "     "
			}
		}
	}

	style := lipgloss.NewStyle().Foreground(fg)

	if isSelected {
		style = style.Background(lipgloss.Color("22"))
	}

	if isCursor && !m.won {
		if line == 0 {
			cursorStyle := lipgloss.NewStyle().Foreground(colorCursor).Bold(true)
			if isSelected {
				cursorStyle = cursorStyle.Background(lipgloss.Color("22"))
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
