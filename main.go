package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// gameMode tracks which UI mode is active.
type gameMode int

const (
	modeFreePlay    gameMode = iota // normal puzzle play
	modeEditor                      // board editor
	modeNameInput                   // nickname entry before league
	modeLeague                      // league puzzle browser
	modeLeaguePlay                  // playing a league puzzle
	modeLeaderboard                 // leaderboard view
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
	colorCustom = lipgloss.Color("45")  // cyan for custom badge
	colorGhost  = lipgloss.Color("240") // dim gray for ghost preview
	colorEditor = lipgloss.Color("177") // light purple for editor badge
	colorError  = lipgloss.Color("196") // red for error messages
	colorLeague = lipgloss.Color("220") // gold for league badge
	colorLocked = lipgloss.Color("240") // dim for locked puzzles
	colorScored = lipgloss.Color("82")  // green for scored puzzles
	colorRank   = lipgloss.Color("220") // gold for rank numbers
)

func diffColor(d Difficulty) lipgloss.Color {
	switch d {
	case Easy:
		return colorEasy
	case Medium:
		return colorMedDif
	case Hard:
		return colorHard
	case Custom:
		return colorCustom
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

// editorSolveMsg is sent when the editor's solvability check completes.
type editorSolveMsg struct {
	optimal int // -1 if unsolvable, otherwise the optimal move count
}

type model struct {
	mode gameMode

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

	editPiece   PieceKind // currently selected piece type for placement
	editError   string    // error message to display in editor
	editSolving bool      // true while checking solvability
	custom      bool      // true when playing a custom board

	// League fields.
	nickname      string    // current player name
	nameInput     string    // text buffer for nickname entry
	saveData      *SaveData // persistent save data (loaded on league entry)
	leagueIdx     int       // selected puzzle index in league browser
	leagueScroll  int       // scroll offset for league browser
	leagueScore   int       // score for last completed league puzzle
	leagueNewBest bool      // true when a new best score was achieved
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

func editorSolveCmd(b *Board) tea.Cmd {
	return func() tea.Msg {
		return editorSolveMsg{optimal: Solve(b)}
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
		m.custom = false
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

	case editorSolveMsg:
		m.editSolving = false
		if msg.optimal == -1 {
			m.editError = "Unsolvable! Adjust pieces and try again."
			return m, nil
		}
		// Transition to play mode with the custom board.
		m.mode = modeFreePlay
		m.optimal = msg.optimal
		m.difficulty = Custom
		m.custom = true
		m.won = false
		m.selected = -1
		m.history = nil
		m.cursorX = 0
		m.cursorY = 0
		m.board.Moves = 0
		m.editError = ""
		m.hint = nil
		m.hintLoading = false
		if m.cheatMode {
			m.hintSeq++
			m.hintLoading = true
			return m, computeHintCmd(m.board.Clone(), m.hintSeq)
		}
		return m, nil

	case tea.KeyMsg:
		key := msg.String()

		// Quit — always available.
		if key == "q" || key == "ctrl+c" {
			return m, tea.Quit
		}

		// Route by mode.
		switch m.mode {
		case modeEditor:
			return m.updateEditor(msg)
		case modeNameInput:
			return m.updateNameInput(msg)
		case modeLeague:
			return m.updateLeague(msg)
		case modeLeaguePlay:
			return m.updateLeaguePlay(msg)
		case modeLeaderboard:
			return m.updateLeaderboard(msg)
		default:
			return m.updateFreePlay(msg)
		}
	}
	return m, nil
}

// enterLeague transitions to the league. If no last_player, goes to name input first.
func (m model) enterLeague() (model, tea.Cmd) {
	m.saveData = loadSave()
	if m.saveData.LastPlayer != "" {
		m.nickname = m.saveData.LastPlayer
		return m.enterLeagueBrowser(), nil
	}
	m.mode = modeNameInput
	m.nameInput = ""
	return m, nil
}

// enterLeagueBrowser sets up the league puzzle browser.
func (m model) enterLeagueBrowser() model {
	m.mode = modeLeague
	m.leagueScore = 0
	m.leagueNewBest = false
	pd := m.saveData.player(m.nickname)
	// Position cursor at the highest unlocked puzzle.
	m.leagueIdx = pd.highestUnlocked()
	m.leagueScroll = 0
	m.adjustLeagueScroll()
	return m
}

const leagueVisible = 15 // number of puzzles visible at once in the browser

// adjustLeagueScroll ensures the selected item is visible.
func (m *model) adjustLeagueScroll() {
	if m.leagueIdx < m.leagueScroll {
		m.leagueScroll = m.leagueIdx
	}
	if m.leagueIdx >= m.leagueScroll+leagueVisible {
		m.leagueScroll = m.leagueIdx - leagueVisible + 1
	}
}

// updateFreePlay handles keys in the normal free-play mode.
func (m model) updateFreePlay(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Ignore keys while generating.
	if m.loading {
		return m, nil
	}

	// Enter league mode.
	if key == "g" {
		m2, cmd := m.enterLeague()
		return m2, cmd
	}

	// Enter editor mode.
	if key == "e" && !m.won {
		m.mode = modeEditor
		m.board = &Board{Pieces: []Piece{}}
		m.cursorX = 0
		m.cursorY = 0
		m.selected = -1
		m.editPiece = Large
		m.editError = ""
		m.editSolving = false
		m.hint = nil
		m.hintLoading = false
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
		m.custom = false
		return m, generateBoardCmd(d)
	}

	// New game (same difficulty) — if custom, exit custom mode.
	if key == "n" {
		if m.custom {
			m.difficulty = Easy
			m.custom = false
		}
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

	return m.updatePlay(msg)
}

// updatePlay handles shared play controls (cursor, select, move, undo) used
// by both free play and league play.
func (m model) updatePlay(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Undo.
	if key == "u" && len(m.history) > 0 {
		m.board = m.history[len(m.history)-1]
		m.history = m.history[:len(m.history)-1]
		m.selected = -1
		m.won = false
		if m.cheatMode {
			m.hintSeq++
			m.hint = nil
			m.hintLoading = true
			return m, computeHintCmd(m.board.Clone(), m.hintSeq)
		}
		return m, nil
	}

	// Reset to starting state.
	if key == "U" && len(m.history) > 0 {
		m.board = m.history[0]
		m.history = nil
		m.selected = -1
		m.won = false
		m.hint = nil
		m.hintLoading = false
		if m.cheatMode {
			m.hintSeq++
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
		if m.selected != -1 {
			m.selected = -1
			return m, nil
		}
		// In league play, Esc with no selection returns to league browser.
		if m.mode == modeLeaguePlay {
			m = m.enterLeagueBrowser()
			return m, nil
		}
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
					// In league play, auto-save the score.
					if m.mode == modeLeaguePlay {
						m.leagueScore = calcScore(m.optimal, m.board.Moves)
						m.leagueNewBest = false
						pd := m.saveData.player(m.nickname)
						prev, hasPrev := pd.Scores[m.leagueIdx]
						if !hasPrev || m.leagueScore > prev {
							pd.Scores[m.leagueIdx] = m.leagueScore
							m.leagueNewBest = true
							_ = m.saveData.save()
						}
					}
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
	return m, nil
}

// updateNameInput handles nickname text entry.
func (m model) updateNameInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc":
		// Cancel — return to free play.
		m.mode = modeFreePlay
		return m, nil
	case "enter":
		name := strings.TrimSpace(m.nameInput)
		if name == "" {
			return m, nil
		}
		m.nickname = name
		m.saveData.LastPlayer = name
		_ = m.saveData.save()
		m = m.enterLeagueBrowser()
		return m, nil
	case "backspace":
		if len(m.nameInput) > 0 {
			m.nameInput = m.nameInput[:len(m.nameInput)-1]
		}
	default:
		// Only accept printable characters, limit length.
		if len(key) == 1 && len(m.nameInput) < 20 {
			m.nameInput += key
		}
	}
	return m, nil
}

// updateLeague handles keys in the league puzzle browser.
func (m model) updateLeague(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	pd := m.saveData.player(m.nickname)
	maxIdx := pd.highestUnlocked()

	switch key {
	case "esc":
		// Return to free play.
		m.mode = modeFreePlay
		return m, nil
	case "tab":
		// Show leaderboard.
		m.mode = modeLeaderboard
		return m, nil
	case "@":
		// Switch player — go to name input.
		m.mode = modeNameInput
		m.nameInput = ""
		return m, nil
	case "up", "k":
		if m.leagueIdx > 0 {
			m.leagueIdx--
			m.adjustLeagueScroll()
		}
	case "down", "j":
		if m.leagueIdx < len(presets)-1 {
			m.leagueIdx++
			m.adjustLeagueScroll()
		}
	case "home", "g":
		m.leagueIdx = 0
		m.adjustLeagueScroll()
	case "end", "G":
		m.leagueIdx = len(presets) - 1
		m.adjustLeagueScroll()
	case "enter", " ":
		// Start puzzle if unlocked.
		if m.leagueIdx <= maxIdx {
			preset := presets[m.leagueIdx]
			m.mode = modeLeaguePlay
			m.board = &Board{
				Pieces: append([]Piece{}, preset.Pieces...),
			}
			m.optimal = preset.Optimal
			m.won = false
			m.selected = -1
			m.history = nil
			m.cursorX = 0
			m.cursorY = 0
			m.leagueScore = 0
			m.leagueNewBest = false
			m.hint = nil
			m.hintLoading = false
			if m.cheatMode {
				m.hintSeq++
				m.hintLoading = true
				return m, computeHintCmd(m.board.Clone(), m.hintSeq)
			}
		}
	}
	return m, nil
}

// updateLeaguePlay handles keys during league puzzle play.
func (m model) updateLeaguePlay(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

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

	// Toggle coordinate labels.
	if key == "c" {
		m.showCoords = !m.showCoords
		return m, nil
	}

	return m.updatePlay(msg)
}

// updateLeaderboard handles keys in the leaderboard view.
func (m model) updateLeaderboard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc", "tab":
		// Return to league browser.
		m.mode = modeLeague
	}
	return m, nil
}

func (m model) View() string {
	switch m.mode {
	case modeEditor:
		return m.viewEditor()
	case modeNameInput:
		return m.viewNameInput()
	case modeLeague:
		return m.viewLeague()
	case modeLeaguePlay:
		return m.viewLeaguePlay()
	case modeLeaderboard:
		return m.viewLeaderboard()
	default:
		return m.viewFreePlay()
	}
}

// viewFreePlay renders the normal free-play screen.
func (m model) viewFreePlay() string {
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

	m.renderBoard(&sb)

	sb.WriteString("\n")

	// Status: moves.
	movesStr := fmt.Sprintf("  Moves: %d", m.board.Moves)
	if len(m.history) > 0 {
		movesStr += "  (u: undo  U: restart)"
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
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("  u: undo  U: restart  n: new game  1/2/3: change difficulty  q: quit"))
		sb.WriteString("\n")
	} else {
		sb.WriteString("\n")
		if m.selected != -1 {
			selStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
			sb.WriteString(selStyle.Render("  Piece selected — arrow keys to move, esc to deselect"))
		} else {
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(
				"  Arrows/hjkl: move  Enter/Space: select  n: new  e: editor  g: league  ?: cheat  1/2/3: difficulty  q: quit"))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// viewNameInput renders the nickname entry screen.
func (m model) viewNameInput() string {
	var sb strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255")).
		Render("KLOTSKI PUZZLE")
	sb.WriteString(title)
	sb.WriteString("  ")
	badge := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(colorLeague).
		Padding(0, 1).
		Render("LEAGUE")
	sb.WriteString(badge)
	sb.WriteString("\n\n")

	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true).Render("  Enter your nickname:"))
	sb.WriteString("\n\n")

	// Text input with cursor.
	inputStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	cursorChar := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("_")
	sb.WriteString("  > ")
	sb.WriteString(inputStyle.Render(m.nameInput))
	sb.WriteString(cursorChar)
	sb.WriteString("\n\n")

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	sb.WriteString(helpStyle.Render("  Enter: confirm  Esc: cancel  q: quit"))
	sb.WriteString("\n")

	return sb.String()
}

// viewLeague renders the league puzzle browser.
func (m model) viewLeague() string {
	var sb strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255")).
		Render("KLOTSKI PUZZLE")
	sb.WriteString(title)
	sb.WriteString("  ")
	badge := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(colorLeague).
		Padding(0, 1).
		Render("LEAGUE")
	sb.WriteString(badge)

	// Player name.
	sb.WriteString("  ")
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true).Render(m.nickname))

	sb.WriteString("\n\n")

	pd := m.saveData.player(m.nickname)
	maxIdx := pd.highestUnlocked()

	// Summary line.
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	sb.WriteString(dimStyle.Render(fmt.Sprintf("  Score: %d/%d  Completed: %d/%d",
		pd.totalScore(), maxLeagueScore(), pd.completed(), len(presets))))
	sb.WriteString("\n\n")

	// Puzzle list.
	end := m.leagueScroll + leagueVisible
	if end > len(presets) {
		end = len(presets)
	}

	for i := m.leagueScroll; i < end; i++ {
		p := presets[i]
		selected := i == m.leagueIdx
		locked := i > maxIdx
		score, scored := pd.Scores[i]

		var line string
		numStr := fmt.Sprintf("%3d.", i+1)
		optStr := fmt.Sprintf("(%d moves)", p.Optimal)

		if locked {
			lockStyle := lipgloss.NewStyle().Foreground(colorLocked)
			line = lockStyle.Render(fmt.Sprintf("  %s  locked  %s", numStr, optStr))
		} else if scored {
			scoreStyle := lipgloss.NewStyle().Foreground(colorScored)
			dimOptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
			line = fmt.Sprintf("  %s  %s  %s",
				lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Render(numStr),
				scoreStyle.Render(fmt.Sprintf("%d/10", score)),
				dimOptStyle.Render(optStr))
		} else {
			// Unlocked but not yet scored.
			line = fmt.Sprintf("  %s  %s  %s",
				lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true).Render(numStr),
				lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(" --  "),
				lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(optStr))
		}

		if selected {
			// Highlight with a cursor indicator.
			sb.WriteString(lipgloss.NewStyle().Foreground(colorCursor).Bold(true).Render("> "))
			sb.WriteString(line)
		} else {
			sb.WriteString("  ")
			sb.WriteString(line)
		}
		sb.WriteString("\n")
	}

	// Scroll indicators.
	if m.leagueScroll > 0 {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("  ... %d more above", m.leagueScroll)))
		sb.WriteString("\n")
	}
	if end < len(presets) {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("  ... %d more below", len(presets)-end)))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	sb.WriteString(helpStyle.Render("  Arrows/jk: browse  Enter: play  Tab: leaderboard  @: switch player  Esc: back  q: quit"))
	sb.WriteString("\n")

	return sb.String()
}

// viewLeaguePlay renders the league play screen (similar to free play but with league info).
func (m model) viewLeaguePlay() string {
	var sb strings.Builder

	// Title.
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255")).
		Render("KLOTSKI PUZZLE")
	sb.WriteString(title)

	// League badge.
	sb.WriteString("  ")
	badge := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(colorLeague).
		Padding(0, 1).
		Render(fmt.Sprintf("LEAGUE #%d", m.leagueIdx+1))
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

	m.renderBoard(&sb)

	sb.WriteString("\n")

	// Status: moves.
	movesStr := fmt.Sprintf("  Moves: %d", m.board.Moves)
	if len(m.history) > 0 {
		movesStr += "  (u: undo  U: restart)"
	}
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(movesStr))
	sb.WriteString("\n")

	// Show current best score for this puzzle.
	pd := m.saveData.player(m.nickname)
	if prevScore, ok := pd.Scores[m.leagueIdx]; ok {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(
			fmt.Sprintf("  Current best: %d/10", prevScore)))
		sb.WriteString("\n")
	}

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
		// Score line.
		scoreStr := fmt.Sprintf("  Score: %d/10", m.leagueScore)
		if m.leagueNewBest {
			scoreStr += "  NEW BEST!"
		}
		sb.WriteString(winStyle.Render(scoreStr))
		sb.WriteString("\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("  u: undo  U: restart (retry for better score)  Esc: back to league  q: quit"))
		sb.WriteString("\n")
	} else {
		sb.WriteString("\n")
		if m.selected != -1 {
			selStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
			sb.WriteString(selStyle.Render("  Piece selected — arrow keys to move, esc to deselect"))
		} else {
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(
				"  Arrows/hjkl: move  Enter/Space: select  c: coords  ?: cheat  Esc: back to league  q: quit"))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// viewLeaderboard renders the leaderboard.
func (m model) viewLeaderboard() string {
	var sb strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255")).
		Render("KLOTSKI PUZZLE")
	sb.WriteString(title)
	sb.WriteString("  ")
	badge := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(colorLeague).
		Padding(0, 1).
		Render("LEADERBOARD")
	sb.WriteString(badge)
	sb.WriteString("\n\n")

	entries := m.saveData.leaderboard()

	if len(entries) == 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("  No players yet."))
		sb.WriteString("\n")
	} else {
		// Header.
		headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Bold(true)
		sb.WriteString(headerStyle.Render(fmt.Sprintf("  %-4s  %-20s  %6s  %9s", "Rank", "Player", "Score", "Completed")))
		sb.WriteString("\n")
		sb.WriteString(headerStyle.Render("  " + strings.Repeat("-", 45)))
		sb.WriteString("\n")

		for i, e := range entries {
			rankStyle := lipgloss.NewStyle().Foreground(colorRank).Bold(true)
			nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
			scoreStyle := lipgloss.NewStyle().Foreground(colorScored)
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

			// Highlight current player.
			if e.Name == m.nickname {
				nameStyle = nameStyle.Bold(true).Foreground(lipgloss.Color("220"))
			}

			sb.WriteString(fmt.Sprintf("  %s  %s  %s  %s",
				rankStyle.Render(fmt.Sprintf("%-4s", fmt.Sprintf("#%d", i+1))),
				nameStyle.Render(fmt.Sprintf("%-20s", e.Name)),
				scoreStyle.Render(fmt.Sprintf("%6d", e.Total)),
				dimStyle.Render(fmt.Sprintf("%5d/%d", e.Completed, len(presets))),
			))
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	sb.WriteString(helpStyle.Render("  Esc/Tab: back to league  q: quit"))
	sb.WriteString("\n")

	return sb.String()
}

// renderBoard draws the grid (with box-drawing borders) into the string builder.
// It is shared between the play mode View and the editor viewEditor.
func (m model) renderBoard(sb *strings.Builder) {
	grid := m.board.occupancy()

	// Ghost piece for editor preview.
	var ghost *Piece
	var ghostGrid [BoardW][BoardH]bool
	if m.mode == modeEditor && m.board.PieceAt(m.cursorX, m.cursorY) == -1 {
		candidate := Piece{Kind: m.editPiece, X: m.cursorX, Y: m.cursorY}
		if m.board.CanPlace(candidate) {
			ghost = &candidate
			for _, c := range candidate.Cells() {
				ghostGrid[c[0]][c[1]] = true
			}
		}
	}

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
				cellStr := m.renderCell(x, y, idx, line, ghost, ghostGrid)
				sb.WriteString(cellStr)
				if x < BoardW-1 {
					// Check if we should merge border with ghost cells.
					sameReal := idx != -1 && x+1 < BoardW && grid[x+1][y] == idx
					sameGhost := ghost != nil && ghostGrid[x][y] && ghostGrid[x+1][y]
					if sameReal || sameGhost {
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
				sameReal := top != -1 && top == bot
				sameGhost := ghost != nil && ghostGrid[x][y] && ghostGrid[x][y+1]
				if sameReal || sameGhost {
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
}

func (m model) renderCell(x, y, idx, line int, ghost *Piece, ghostGrid [BoardW][BoardH]bool) string {
	isCursor := (x == m.cursorX && y == m.cursorY)
	isSelected := (idx != -1 && idx == m.selected)
	isHinted := m.cheatMode && m.hint != nil && idx != -1 && idx == m.hint.PieceIndex && !m.won
	isGhost := ghost != nil && ghostGrid[x][y]

	var label string
	var fg lipgloss.Color

	if isGhost && idx == -1 {
		// Ghost preview cell.
		fg = colorGhost
		switch ghost.Kind {
		case Small:
			if line == 0 {
				label = "  s  "
			} else {
				label = "     "
			}
		case Vertical:
			if line == 0 && y == ghost.Y {
				label = "  m  "
			} else if line == 1 && y == ghost.Y+1 {
				label = "  m  "
			} else {
				label = "     "
			}
		case Horizontal:
			if line == 0 {
				label = "  m  "
			} else {
				label = "     "
			}
		case Large:
			if line == 0 {
				label = "  L  "
			} else {
				label = "     "
			}
		}
		style := lipgloss.NewStyle().Foreground(fg)
		// Show cursor on ghost origin cell.
		if isCursor && line == 0 {
			cursorStyle := lipgloss.NewStyle().Foreground(colorCursor).Bold(true)
			return cursorStyle.Render(fmt.Sprintf(" [%s] ", editPieceShort(ghost.Kind)))
		}
		return style.Render(label)
	}

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
			cursorLabel := "[*]"
			if m.mode == modeEditor && idx == -1 {
				cursorLabel = fmt.Sprintf("[%s]", editPieceShort(m.editPiece))
			}
			return cursorStyle.Render(fmt.Sprintf(" %s ", cursorLabel))
		}
		style = style.Bold(true)
	}

	return style.Render(label)
}

// editPieceShort returns a short label for a piece kind (used in cursor).
func editPieceShort(k PieceKind) string {
	switch k {
	case Large:
		return "L"
	case Vertical:
		return "V"
	case Horizontal:
		return "H"
	case Small:
		return "s"
	}
	return "?"
}

// editPieceLabel returns a descriptive label for the piece selector.
func editPieceLabel(k PieceKind) string {
	switch k {
	case Large:
		return "Large 2x2"
	case Vertical:
		return "Vertical 1x2"
	case Horizontal:
		return "Horizontal 2x1"
	case Small:
		return "Small 1x1"
	}
	return "?"
}

// editPieceColor returns the display color for a piece kind.
func editPieceColor(k PieceKind) lipgloss.Color {
	switch k {
	case Large:
		return colorLarge
	case Horizontal, Vertical:
		return colorMedium
	case Small:
		return colorSmall
	}
	return colorEmpty
}

// countPieces counts pieces by kind on the board.
func countPieces(b *Board) (large, vert, horiz, small int) {
	for _, p := range b.Pieces {
		switch p.Kind {
		case Large:
			large++
		case Vertical:
			vert++
		case Horizontal:
			horiz++
		case Small:
			small++
		}
	}
	return
}

// validateEditor checks if the editor board is valid for play.
// Returns an error message, or empty string if valid.
func validateEditor(b *Board) string {
	l, _, _, _ := countPieces(b)
	if l == 0 {
		return "Need exactly 1 Large (2x2) piece."
	}
	if l > 1 {
		return "Too many Large pieces (max 1)."
	}
	if b.IsWon() {
		return "Large piece is already at the goal!"
	}
	// Count occupied cells — need at least 2 empty for movement.
	occupied := 0
	for _, p := range b.Pieces {
		occupied += len(p.Cells())
	}
	if BoardW*BoardH-occupied < 2 {
		return "Need at least 2 empty cells."
	}
	return ""
}

// updateEditor handles key events in editor mode.
func (m model) updateEditor(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Don't allow most actions while solving.
	if m.editSolving {
		return m, nil
	}

	switch key {
	// Move cursor.
	case "up", "k":
		if m.cursorY > 0 {
			m.cursorY--
		}
	case "down", "j":
		if m.cursorY < BoardH-1 {
			m.cursorY++
		}
	case "left", "h":
		if m.cursorX > 0 {
			m.cursorX--
		}
	case "right", "l":
		if m.cursorX < BoardW-1 {
			m.cursorX++
		}

	// Cycle piece type.
	case "tab":
		m.editError = ""
		switch m.editPiece {
		case Large:
			m.editPiece = Vertical
		case Vertical:
			m.editPiece = Horizontal
		case Horizontal:
			m.editPiece = Small
		case Small:
			m.editPiece = Large
		}

	// Place piece.
	case "enter", " ":
		m.editError = ""
		p := Piece{Kind: m.editPiece, X: m.cursorX, Y: m.cursorY}
		if m.board.CanPlace(p) {
			m.board.Pieces = append(m.board.Pieces, p)
		} else {
			m.editError = "Can't place here — overlaps or out of bounds."
		}

	// Remove piece at cursor.
	case "x", "backspace", "delete":
		m.editError = ""
		if !m.board.RemovePieceAt(m.cursorX, m.cursorY) {
			m.editError = "No piece here to remove."
		}

	// Clear board.
	case "r":
		m.board = &Board{Pieces: []Piece{}}
		m.editError = ""

	// Toggle coordinates.
	case "c":
		m.showCoords = !m.showCoords

	// Play — validate and solve.
	case "p":
		m.editError = ""
		if errMsg := validateEditor(m.board); errMsg != "" {
			m.editError = errMsg
			return m, nil
		}
		m.editSolving = true
		m.editError = ""
		return m, editorSolveCmd(m.board.Clone())

	// Cancel — exit editor, generate a random board.
	case "esc":
		m.mode = modeFreePlay
		m.editError = ""
		m.loading = true
		return m, generateBoardCmd(m.difficulty)
	}

	return m, nil
}

// viewEditor renders the editor UI.
func (m model) viewEditor() string {
	var sb strings.Builder

	// Title.
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255")).
		Render("KLOTSKI PUZZLE")
	sb.WriteString(title)

	// Editor badge.
	sb.WriteString("  ")
	badge := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(colorEditor).
		Padding(0, 1).
		Render("EDITOR")
	sb.WriteString(badge)
	sb.WriteString("\n\n")

	// Piece selector.
	kinds := []PieceKind{Large, Vertical, Horizontal, Small}
	sb.WriteString("  Piece: ")
	for i, k := range kinds {
		style := lipgloss.NewStyle().Foreground(editPieceColor(k))
		if k == m.editPiece {
			style = style.Bold(true).Underline(true)
		}
		sb.WriteString(style.Render(editPieceLabel(k)))
		if i < len(kinds)-1 {
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("  "))
		}
	}
	sb.WriteString("\n")

	// Piece counts.
	l, v, h, s := countPieces(m.board)
	occupied := 0
	for _, p := range m.board.Pieces {
		occupied += len(p.Cells())
	}
	empty := BoardW*BoardH - occupied
	countStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	sb.WriteString(countStyle.Render(fmt.Sprintf("  L:%d  V:%d  H:%d  S:%d  Empty:%d", l, v, h, s, empty)))
	sb.WriteString("\n\n")

	m.renderBoard(&sb)

	sb.WriteString("\n")

	// Error message.
	if m.editError != "" {
		errStyle := lipgloss.NewStyle().Foreground(colorError).Bold(true)
		sb.WriteString(errStyle.Render("  " + m.editError))
		sb.WriteString("\n")
	}

	// Solving indicator.
	if m.editSolving {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("  Checking solvability..."))
		sb.WriteString("\n")
	}

	// Help.
	sb.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	sb.WriteString(helpStyle.Render("  Arrows/hjkl: move cursor  Tab: cycle piece  Enter/Space: place"))
	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render("  x/Backspace: remove  r: clear  c: coords  p: play  Esc: cancel  q: quit"))
	sb.WriteString("\n")

	return sb.String()
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
