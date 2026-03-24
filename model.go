package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Enums ─────────────────────────────────────────────────────────────────────

type appState int

const (
	stateMenu    appState = iota
	stateTyping
	stateResults
)

type testMode int

const (
	modeWords  testMode = iota // fixed word count
	modeTime                   // countdown timer
	modeQuote                  // random literary quote
)

// ── Constants ─────────────────────────────────────────────────────────────────

const (
	numWords    = 30
	lineWidth   = 60
	visLines    = 3
)

var timeLimits = []int{15, 30, 60, 120} // seconds

// ── Tick message ──────────────────────────────────────────────────────────────

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// ── Model ─────────────────────────────────────────────────────────────────────

type Model struct {
	state appState
	mode  testMode

	// time-mode settings
	timeLimitIdx int // index into timeLimits
	timeLeft     int // seconds remaining

	// content
	target     []rune
	input      []rune
	activeQuote Quote  // only in quote mode
	startTime  time.Time
	elapsed    time.Duration
	started    bool

	// stats
	totalKeys int
	errors    int

	// frozen results
	finalWPM float64
	finalAcc float64
	isPB     bool

	// menu cursor
	menuRow int // 0=mode, 1=time limit (only if modeTime)
	menuCol int // which option within the row

	// layout
	width  int
	height int

	// line wrapping
	lines   []string
	offsets []int
}

func NewModel() Model {
	m := Model{
		state:        stateMenu,
		mode:         modeWords,
		timeLimitIdx: 1, // default 30s
		menuRow:      0,
		menuCol:      0,
	}
	return m
}

func (m *Model) loadText() {
	switch m.mode {
	case modeWords:
		text := generateWords(numWords)
		m.target = []rune(text)
		m.lines, m.offsets = wrapIntoLines(text, lineWidth)
	case modeTime:
		// generate a long stream of words — more than any timer needs
		text := generateWords(200)
		m.target = []rune(text)
		m.lines, m.offsets = wrapIntoLines(text, lineWidth)
	case modeQuote:
		m.activeQuote = randomQuote()
		m.target = []rune(m.activeQuote.Text)
		m.lines, m.offsets = wrapIntoLines(m.activeQuote.Text, lineWidth)
	}
	m.input = nil
	m.totalKeys = 0
	m.errors = 0
	m.started = false
	m.elapsed = 0
	if m.mode == modeTime {
		m.timeLeft = timeLimits[m.timeLimitIdx]
	}
}

// ── Init ──────────────────────────────────────────────────────────────────────

func (m Model) Init() tea.Cmd { return nil }

// ── Update ────────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		if m.state == stateTyping && m.mode == modeTime && m.started {
			m.timeLeft--
			if m.timeLeft <= 0 {
				return m.finishTest(), nil
			}
			return m, tickCmd()
		}

	case tea.KeyMsg:
		switch m.state {
		case stateMenu:
			return m.updateMenu(msg)
		case stateTyping:
			return m.updateTyping(msg)
		case stateResults:
			return m.updateResults(msg)
		}
	}
	return m, nil
}

// ── Menu update ───────────────────────────────────────────────────────────────

func (m Model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		return m, tea.Quit

	case tea.KeyLeft:
		if m.menuRow == 0 {
			m.menuCol = (m.menuCol + 2) % 3 // 3 modes
		} else {
			m.menuCol = (m.menuCol + len(timeLimits) - 1) % len(timeLimits)
		}
		if m.menuRow == 0 {
			m.mode = testMode(m.menuCol)
		} else {
			m.timeLimitIdx = m.menuCol
		}

	case tea.KeyRight:
		if m.menuRow == 0 {
			m.menuCol = (m.menuCol + 1) % 3
		} else {
			m.menuCol = (m.menuCol + 1) % len(timeLimits)
		}
		if m.menuRow == 0 {
			m.mode = testMode(m.menuCol)
		} else {
			m.timeLimitIdx = m.menuCol
		}

	case tea.KeyUp, tea.KeyDown:
		if m.mode == modeTime {
			if m.menuRow == 0 {
				m.menuRow = 1
				m.menuCol = m.timeLimitIdx
			} else {
				m.menuRow = 0
				m.menuCol = int(m.mode)
			}
		}

	case tea.KeyEnter:
		m.loadText()
		m.state = stateTyping
		m.startTime = time.Now()
		if m.mode == modeTime {
			return m, tickCmd()
		}
		return m, nil

	default:
		// any printable key immediately starts the test
		if len(msg.Runes) > 0 {
			m.loadText()
			m.state = stateTyping
			m.startTime = time.Now()
			m.started = true
			var cmd tea.Cmd
			m, cmd2 := m.handleTypingKey(msg)
			cmds := []tea.Cmd{cmd2}
			if m.(Model).mode == modeTime {
				cmds = append(cmds, tickCmd())
			}
			_ = cmd
			return m, tea.Batch(cmds...)
		}
	}
	return m, nil
}

// ── Typing update ─────────────────────────────────────────────────────────────

func (m Model) updateTyping(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		return m, tea.Quit
	case tea.KeyCtrlR:
		m.loadText()
		m.state = stateTyping
		m.startTime = time.Now()
		if m.mode == modeTime {
			return m, tickCmd()
		}
		return m, nil
	}
	return m.handleTypingKey(msg)
}

func (m Model) handleTypingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if !m.started {
		m.started = true
		m.startTime = time.Now()
	}

	switch msg.Type {
	case tea.KeyBackspace:
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}

	case tea.KeySpace:
		m = m.appendRune(' ')

	case tea.KeyRunes:
		for _, r := range msg.Runes {
			m = m.appendRune(r)
		}
	}

	// word/quote mode: finish when all chars typed
	if m.mode != modeTime && len(m.input) >= len(m.target) {
		return m.finishTest(), nil
	}

	return m, nil
}

func (m Model) appendRune(r rune) Model {
	pos := len(m.input)
	if pos >= len(m.target) {
		return m
	}
	m.totalKeys++
	if r != m.target[pos] {
		m.errors++
	}
	m.input = append(m.input, r)
	return m
}

func (m Model) finishTest() Model {
	m.elapsed = time.Since(m.startTime)
	m.finalWPM = m.calcWPM()
	m.finalAcc = m.calcAccuracy()

	modeKey := []string{"words", "time", "quote"}[int(m.mode)]
	dur := 0
	if m.mode == modeTime {
		dur = timeLimits[m.timeLimitIdx]
	}
	pb := personalBest(modeKey, dur)
	m.isPB = m.finalWPM > pb

	saveScore(ScoreEntry{
		WPM:      m.finalWPM,
		Accuracy: m.finalAcc,
		Mode:     modeKey,
		Duration: dur,
		At:       time.Now(),
	})

	m.state = stateResults
	return m
}

// ── Results update ────────────────────────────────────────────────────────────

func (m Model) updateResults(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		return m, tea.Quit
	case tea.KeyEnter:
		return m.restart()
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "r", "R":
			return m.restart()
		case "m", "M":
			next := NewModel()
			next.width = m.width
			next.height = m.height
			next.mode = m.mode
			next.timeLimitIdx = m.timeLimitIdx
			return next, nil
		}
	}
	return m, nil
}

func (m Model) restart() (tea.Model, tea.Cmd) {
	m.loadText()
	m.state = stateTyping
	m.startTime = time.Now()
	if m.mode == modeTime {
		return m, tickCmd()
	}
	return m, nil
}

// ── Metrics ───────────────────────────────────────────────────────────────────

func (m Model) calcWPM() float64 {
	elapsed := m.elapsed
	if m.state != stateResults {
		if !m.started {
			return 0
		}
		elapsed = time.Since(m.startTime)
	}
	mins := elapsed.Minutes()
	if mins == 0 {
		return 0
	}
	correct := 0
	for i, r := range m.input {
		if i < len(m.target) && r == m.target[i] {
			correct++
		}
	}
	return float64(correct) / 5.0 / mins
}

func (m Model) calcAccuracy() float64 {
	if m.totalKeys == 0 {
		return 100
	}
	return float64(m.totalKeys-m.errors) / float64(m.totalKeys) * 100
}

// ── Views ─────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	switch m.state {
	case stateMenu:
		return m.viewMenu()
	case stateTyping:
		return m.viewTyping()
	case stateResults:
		return m.viewResults()
	}
	return ""
}

func (m Model) viewMenu() string {
	title := titleStyle.Render("typist")
	sub := subtleStyle.Render("offline · open source · no paywall")

	// Mode selector
	modes := []string{"words", "time", "quote"}
	var modeButtons []string
	for i, label := range modes {
		if i == int(m.mode) {
			modeButtons = append(modeButtons, selectedStyle.Render(" "+label+" "))
		} else {
			modeButtons = append(modeButtons, optionStyle.Render(" "+label+" "))
		}
	}
	modeRow := lipgloss.JoinHorizontal(lipgloss.Center, modeButtons...)

	// Time limit selector (only shown in time mode)
	var timeLimitRow string
	if m.mode == modeTime {
		var tlButtons []string
		for i, t := range timeLimits {
			label := fmt.Sprintf("%ds", t)
			if i == m.timeLimitIdx {
				if m.menuRow == 1 {
					tlButtons = append(tlButtons, selectedStyle.Render(" "+label+" "))
				} else {
					tlButtons = append(tlButtons, dimSelectedStyle.Render(" "+label+" "))
				}
			} else {
				tlButtons = append(tlButtons, optionStyle.Render(" "+label+" "))
			}
		}
		timeLimitRow = "\n" + lipgloss.JoinHorizontal(lipgloss.Center, tlButtons...)
	}

	// Navigation hint
	var navHint string
	if m.mode == modeTime {
		navHint = subtleStyle.Render("← → switch · ↑ ↓ row · enter start · esc quit")
	} else {
		navHint = subtleStyle.Render("← → switch · enter start · esc quit")
	}

	body := lipgloss.JoinVertical(lipgloss.Center,
		title,
		sub,
		"",
		modeRow+timeLimitRow,
		"",
		navHint,
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)
}

func (m Model) viewTyping() string {
	// figure out which line the cursor is on
	cursorPos := len(m.input)
	cursorLine := 0
	for i, off := range m.offsets {
		if off <= cursorPos {
			cursorLine = i
		}
	}

	startLine := cursorLine
	if startLine+visLines > len(m.lines) {
		startLine = len(m.lines) - visLines
	}
	if startLine < 0 {
		startLine = 0
	}

	var renderedLines []string
	for li := startLine; li < startLine+visLines && li < len(m.lines); li++ {
		lineStart := m.offsets[li]
		lineRunes := []rune(m.lines[li])
		var sb strings.Builder

		for ci, ch := range lineRunes {
			absPos := lineStart + ci
			if absPos < len(m.input) {
				typed := m.input[absPos]
				if typed == ch {
					sb.WriteString(correctStyle.Render(string(ch)))
				} else {
					display := string(typed)
					if ch == ' ' {
						display = "·"
					}
					sb.WriteString(incorrectStyle.Render(display))
				}
			} else if absPos == cursorPos {
				sb.WriteString(cursorStyle.Render(string(ch)))
			} else {
				sb.WriteString(pendingStyle.Render(string(ch)))
			}
		}
		renderedLines = append(renderedLines, sb.String())
	}

	textBlock := strings.Join(renderedLines, "\n")

	// Stats bar
	wpmVal := fmt.Sprintf("%.0f", m.calcWPM())
	accVal := fmt.Sprintf("%.0f%%", m.calcAccuracy())

	var timerPart string
	if m.mode == modeTime {
		color := timeStyle
		if m.timeLeft <= 10 {
			color = incorrectStyle
		}
		timerPart = "   " + color.Render(fmt.Sprintf("%ds", m.timeLeft))
	}

	stats := lipgloss.JoinHorizontal(lipgloss.Top,
		wpmStyle.Render(wpmVal),
		subtleStyle.Render(" wpm   "),
		accStyle.Render(accVal),
		subtleStyle.Render(" acc"),
		timerPart,
	)

	// quote attribution
	var quoteAttr string
	if m.mode == modeQuote {
		quoteAttr = subtleStyle.Render("— " + m.activeQuote.Author)
	}

	hint := hintStyle.Render("ctrl+r restart · esc quit")

	var parts []string
	parts = append(parts, stats, "")
	if quoteAttr != "" {
		parts = append(parts, quoteAttr)
	}
	parts = append(parts, textBlock, "", hint)

	body := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)
}

func (m Model) viewResults() string {
	title := titleStyle.Render("results")

	pbTag := ""
	if m.isPB {
		pbTag = "  " + pbStyle.Render(" new best! ")
	}

	wpmLine := lipgloss.JoinHorizontal(lipgloss.Top,
		wpmStyle.Render(fmt.Sprintf("%-8.0f", m.finalWPM)),
		subtleStyle.Render("wpm"),
		pbTag,
	)
	accLine := lipgloss.JoinHorizontal(lipgloss.Top,
		accStyle.Render(fmt.Sprintf("%-8.1f", m.finalAcc)),
		subtleStyle.Render("accuracy"),
	)
	timeLine := lipgloss.JoinHorizontal(lipgloss.Top,
		timeStyle.Render(fmt.Sprintf("%-8.1f", m.elapsed.Seconds())),
		subtleStyle.Render("seconds"),
	)

	// personal best line
	modeKey := []string{"words", "time", "quote"}[int(m.mode)]
	dur := 0
	if m.mode == modeTime {
		dur = timeLimits[m.timeLimitIdx]
	}
	pb := personalBest(modeKey, dur)
	pbLine := lipgloss.JoinHorizontal(lipgloss.Top,
		subtleStyle.Render(fmt.Sprintf("%-8.0f", pb)),
		subtleStyle.Render("personal best"),
	)

	card := cardStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		title, "",
		wpmLine,
		accLine,
		timeLine,
		"",
		pbLine,
	))

	actions := lipgloss.JoinVertical(lipgloss.Center,
		pendingStyle.Render("enter / r  → again"),
		pendingStyle.Render("m          → menu"),
		hintStyle.Render("esc        → quit"),
	)

	body := lipgloss.JoinVertical(lipgloss.Center, card, "", actions)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)
}
