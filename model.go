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
	stateMenu      appState = iota
	stateTyping
	stateResults
	stateHistory
	stateTimeInput // custom time entry
)

type testMode int

const (
	modeWords testMode = iota
	modeTime
	modeQuote
	modeCode
)

var modeNames = []string{"words", "time", "quote", "code"}
var modeCount = len(modeNames)

// ── Constants ─────────────────────────────────────────────────────────────────

const (
	numWords     = 30
	lineWidth    = 65
	visLines     = 3
	histPageSize = 12
	sparkWidth   = 32 // WPM sparkline bar count
)

var timeLimits = []int{15, 30, 60, 120}

// Sparkline unicode bars
var sparkBars = []rune("▁▂▃▄▅▆▇█")

// ── Messages ──────────────────────────────────────────────────────────────────

type tickMsg   time.Time
type exportMsg struct {
	path   string
	err    error
	format string
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func exportJSONCmd() tea.Cmd {
	return func() tea.Msg {
		path, err := exportJSON()
		return exportMsg{path: path, err: err, format: "json"}
	}
}

func exportCSVCmd() tea.Cmd {
	return func() tea.Msg {
		path, err := exportCSV()
		return exportMsg{path: path, err: err, format: "csv"}
	}
}

// ── mistakeEntry for sorting heatmap ─────────────────────────────────────────

type mistakeEntry struct {
	ch    rune
	count int
}

// ── Model ─────────────────────────────────────────────────────────────────────

type Model struct {
	state appState
	mode  testMode

	timeLimitIdx  int
	timeLeft      int
	customTimeSecs int    // 0 = use timeLimits preset
	customTimeStr  string // digits being typed in stateTimeInput
	langIdx      int
	activeSnippet Snippet

	target      []rune
	input       []rune
	activeQuote Quote
	startTime   time.Time
	elapsed     time.Duration
	started     bool

	blindMode  bool
	focusMode  bool // hide stats while typing
	darkTheme  bool // true = mocha, false = latte

	totalKeys int
	errors    int
	// mistake heatmap: count errors per expected rune
	mistakeMap map[rune]int

	// WPM sampling: one sample per second
	wpmSamples []float64
	lastSample time.Time

	// frozen results
	finalWPM    float64
	finalAcc    float64
	isPB        bool
	exportMsg   string

	// pre-computed token kinds for syntax highlighting
	// per-rune Chroma styles (code mode only)
	hlMap StyleMap

	menuRow int
	menuCol int

	histOffset int
	histData   []ScoreEntry

	width  int
	height int

	lines   []string
	offsets []int
}

func NewModel() Model {
	return Model{
		state:        stateMenu,
		mode:         modeWords,
		timeLimitIdx: 1,
		langIdx:      0,
		mistakeMap:   make(map[rune]int),
		darkTheme:    true,
	}
}

func (m *Model) loadText() {
	var text string
	switch m.mode {
	case modeWords:
		text = generateWords(numWords)
	case modeTime:
		text = generateWords(200)
	case modeQuote:
		m.activeQuote = randomQuote()
		text = m.activeQuote.Text
	case modeCode:
		m.activeSnippet = randomSnippet(langKeys[m.langIdx])
		text = m.activeSnippet.Code
	}
	m.target = []rune(text)
	// Code mode: preserve actual newlines and indentation.
	// All other modes: soft-wrap prose at lineWidth.
	if m.mode == modeCode {
		m.lines, m.offsets = wrapCodeLines(text)
	} else {
		m.lines, m.offsets = wrapIntoLines(text, lineWidth)
	}
	m.input = nil
	m.totalKeys = 0
	m.errors = 0
	m.mistakeMap = make(map[rune]int)
	m.wpmSamples = nil
	m.started = false
	m.elapsed = 0
	m.exportMsg = ""
	if m.mode == modeTime {
		m.timeLeft = m.activeDuration()
	}
	// Build Chroma style map for code mode.
	if m.mode == modeCode {
		m.hlMap = BuildStyleMap(text, langKeys[m.langIdx])
	} else {
		m.hlMap = nil
	}
}

func (m Model) modeKey() string { return modeNames[int(m.mode)] }
func (m Model) langKey() string {
	if m.mode == modeCode {
		return langKeys[m.langIdx]
	}
	return ""
}

// activeDuration returns the effective timer duration in seconds.
// Returns 0 for non-time modes.
func (m Model) activeDuration() int {
	if m.mode != modeTime {
		return 0
	}
	if m.customTimeSecs > 0 {
		return m.customTimeSecs
	}
	if m.timeLimitIdx < len(timeLimits) {
		return timeLimits[m.timeLimitIdx]
	}
	return 60 // fallback
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
		if m.state != stateTyping {
			break
		}
		// In time mode: always count down regardless of whether typing has started.
		// This prevents the timer freezing if the first tick fires before any keypress.
		if m.mode == modeTime {
			if m.started {
				m.wpmSamples = append(m.wpmSamples, m.calcWPM())
				m.lastSample = time.Time(msg)
			}
			m.timeLeft--
			if m.timeLeft <= 0 {
				if !m.started {
					// Timer expired before any typing — just go to results with zero stats
					m.started = true
					m.startTime = time.Now().Add(-time.Duration(m.activeDuration()) * time.Second)
				}
				return m.finishTest(), nil
			}
			return m, tickCmd()
		}
		// Non-time modes: only sample WPM once typing has started
		if m.started {
			m.wpmSamples = append(m.wpmSamples, m.calcWPM())
			m.lastSample = time.Time(msg)
			return m, tickCmd()
		}

	case exportMsg:
		if msg.err != nil {
			m.exportMsg = errorStyle.Render("export failed: " + msg.err.Error())
		} else {
			m.exportMsg = accStyle.Render("saved → " + msg.path)
		}

	case tea.KeyMsg:
		switch m.state {
		case stateMenu:
			return m.updateMenu(msg)
		case stateTyping:
			return m.updateTyping(msg)
		case stateResults:
			return m.updateResults(msg)
		case stateHistory:
			return m.updateHistory(msg)
		case stateTimeInput:
			return m.updateTimeInput(msg)
		}
	}
	return m, nil
}

// ── Menu ──────────────────────────────────────────────────────────────────────

func (m Model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	subCount := m.subRowCount()
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		return m, tea.Quit
	case tea.KeyLeft:
		if m.menuRow == 0 {
			m.menuCol = (m.menuCol + modeCount - 1) % modeCount
			m.mode = testMode(m.menuCol)
		} else {
			m.menuCol = (m.menuCol + subCount - 1) % subCount
			m.applySubRow()
		}
	case tea.KeyRight:
		if m.menuRow == 0 {
			m.menuCol = (m.menuCol + 1) % modeCount
			m.mode = testMode(m.menuCol)
		} else {
			m.menuCol = (m.menuCol + 1) % subCount
			m.applySubRow()
		}
	case tea.KeyUp:
		if m.menuRow == 1 {
			m.menuRow = 0
			m.menuCol = int(m.mode)
		}
	case tea.KeyDown:
		if subCount > 0 && m.menuRow == 0 {
			m.menuRow = 1
			if m.mode == modeTime {
				m.menuCol = m.timeLimitIdx
			} else {
				m.menuCol = m.langIdx
			}
		}
	case tea.KeyEnter:
		// If time mode + custom slot selected, go to time input screen
		if m.mode == modeTime && m.menuRow == 1 && m.timeLimitIdx == len(timeLimits) {
			m.state = stateTimeInput
			m.customTimeStr = ""
			return m, nil
		}
		m.loadText()
		m.state = stateTyping
		return m, tickCmd()
	default:
		if len(msg.Runes) > 0 {
			m.loadText()
			m.state = stateTyping
			m.started = true
			nm, cmd2 := m.handleTypingKey(msg)
			return nm, tea.Batch(tickCmd(), cmd2)
		}
	}
	return m, nil
}

func (m Model) subRowCount() int {
	switch m.mode {
	case modeTime:
		return len(timeLimits) + 1 // +1 for custom
	case modeCode:
		return len(langKeys)
	}
	return 0
}

func (m *Model) applySubRow() {
	switch m.mode {
	case modeTime:
		m.timeLimitIdx = m.menuCol
		// If "custom" slot selected, customTimeSecs stays until changed
	case modeCode:
		m.langIdx = m.menuCol
	}
}

// ── Typing ────────────────────────────────────────────────────────────────────

func (m Model) updateTyping(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		return m, tea.Quit
	case tea.KeyCtrlR:
		m.loadText()
		m.state = stateTyping
		return m, tickCmd()
	case tea.KeyCtrlB:
		m.blindMode = !m.blindMode
		return m, nil
	case tea.KeyCtrlF:
		m.focusMode = !m.focusMode
		return m, nil
	case tea.KeyCtrlT:
		m.darkTheme = !m.darkTheme
		if m.darkTheme {
			applyTheme(mocha)
		} else {
			applyTheme(latte)
		}
		// Rebuild hlMap with new styles
		if m.mode == modeCode && len(m.target) > 0 {
			m.hlMap = BuildStyleMap(string(m.target), langKeys[m.langIdx])
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
	case tea.KeyTab:
		m = m.appendRune('\t')
	case tea.KeyEnter:
		if m.mode == modeCode {
			m = m.appendRune('\n')
		}
	case tea.KeyRunes:
		for _, r := range msg.Runes {
			m = m.appendRune(r)
		}
	}
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
	expected := m.target[pos]
	if r != expected {
		m.errors++
		m.mistakeMap[expected]++
	}
	m.input = append(m.input, r)
	return m
}

func (m Model) finishTest() Model {
	m.elapsed = time.Since(m.startTime)
	m.finalWPM = m.calcWPM()
	m.finalAcc = m.calcAccuracy()
	// Append final WPM sample
	m.wpmSamples = append(m.wpmSamples, m.finalWPM)

	dur := m.activeDuration()
	pb := personalBest(m.modeKey(), m.langKey(), dur)
	m.isPB = m.finalWPM > pb

	saveScore(ScoreEntry{
		WPM: m.finalWPM, Accuracy: m.finalAcc,
		Mode: m.modeKey(), Lang: m.langKey(),
		Duration: dur, At: time.Now(),
	})
	m.state = stateResults
	return m
}

// ── Results ───────────────────────────────────────────────────────────────────

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
			next.width, next.height = m.width, m.height
			next.mode, next.timeLimitIdx, next.langIdx = m.mode, m.timeLimitIdx, m.langIdx
			return next, nil
		case "h", "H":
			m.histData = recentSessions(200)
			m.histOffset = 0
			m.state = stateHistory
			return m, nil
		case "j", "J":
			return m, exportJSONCmd()
		case "c", "C":
			return m, exportCSVCmd()
		}
	}
	return m, nil
}

func (m Model) restart() (tea.Model, tea.Cmd) {
	m.loadText()
	m.state = stateTyping
	return m, tickCmd()
}

// ── Time input ───────────────────────────────────────────────────────────────

// updateTimeInput handles the custom time entry screen.
// User types digits; enter confirms; esc cancels back to menu.
func (m Model) updateTimeInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		m.state = stateMenu
		m.customTimeStr = ""
		return m, nil
	case tea.KeyBackspace:
		if len(m.customTimeStr) > 0 {
			m.customTimeStr = m.customTimeStr[:len(m.customTimeStr)-1]
		}
		return m, nil
	case tea.KeyEnter:
		if m.customTimeStr == "" {
			m.state = stateMenu
			return m, nil
		}
		secs := 0
		for _, ch := range m.customTimeStr {
			secs = secs*10 + int(ch-'0')
		}
		if secs < 1 {
			secs = 1
		}
		if secs > 3600 {
			secs = 3600
		}
		m.customTimeSecs = secs
		m.customTimeStr = ""
		// jump straight into the test
		m.loadText()
		m.state = stateTyping
		return m, tickCmd()
	case tea.KeyRunes:
		for _, r := range msg.Runes {
			if r >= '0' && r <= '9' && len(m.customTimeStr) < 4 {
				m.customTimeStr += string(r)
			}
		}
	}
	return m, nil
}

// ── History ───────────────────────────────────────────────────────────────────

func (m Model) updateHistory(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	max := len(m.histData) - histPageSize
	if max < 0 {
		max = 0
	}
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		m.state = stateMenu
		return m, nil
	case tea.KeyDown:
		if m.histOffset < max {
			m.histOffset++
		}
	case tea.KeyUp:
		if m.histOffset > 0 {
			m.histOffset--
		}
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "q", "Q":
			m.state = stateMenu
		case "j", "J":
			if m.histOffset < max {
				m.histOffset++
			}
		case "k", "K":
			if m.histOffset > 0 {
				m.histOffset--
			}
		}
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
	case stateHistory:
		return m.viewHistory()
	case stateTimeInput:
		return m.viewTimeInput()
	}
	return ""
}

// ── Menu view ─────────────────────────────────────────────────────────────────

func (m Model) viewMenu() string {
	// Logo — double border for visual weight
	logo := lipgloss.NewStyle().
		Foreground(activeTheme.mauve).Bold(true).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(activeTheme.surface1).
		Padding(0, 2).
		Render("typist")

	tagline := lipgloss.NewStyle().Foreground(activeTheme.overlay0).
		Render("offline · open source · no paywall")

	// Section label
	sectionLabel := lipgloss.NewStyle().Foreground(activeTheme.surface2).
		Render("select mode")

	// Mode pills
	var modeBtns []string
	for i, label := range modeNames {
		if i == int(m.mode) {
			modeBtns = append(modeBtns, selectedStyle.Render(label))
		} else {
			modeBtns = append(modeBtns, optionStyle.Render(label))
		}
	}
	modeRow := lipgloss.JoinHorizontal(lipgloss.Center, modeBtns...)

	// Sub-option row
	var subRow string
	buildSubBtns := func(labels []string, activeIdx, focusRow int) string {
		var btns []string
		for i, label := range labels {
			s := optionStyle
			if i == activeIdx {
				if focusRow == 1 && m.menuRow == 1 {
					s = selectedStyle
				} else {
					s = dimSelectedStyle
				}
			}
			btns = append(btns, s.Render(label))
		}
		return lipgloss.JoinHorizontal(lipgloss.Center, btns...)
	}
	switch m.mode {
	case modeTime:
		labels := make([]string, len(timeLimits)+1)
		for i, t := range timeLimits { labels[i] = fmt.Sprintf("%ds", t) }
		// Custom slot
		if m.customTimeSecs > 0 && m.timeLimitIdx == len(timeLimits) {
			labels[len(timeLimits)] = fmt.Sprintf("%ds✎", m.customTimeSecs)
		} else {
			labels[len(timeLimits)] = "custom"
		}
		subRow = "\n" + buildSubBtns(labels, m.timeLimitIdx, 1)
	case modeCode:
		subRow = "\n" + buildSubBtns(langKeys, m.langIdx, 1)
	}

	var hint string
	if m.mode == modeTime || m.mode == modeCode {
		hint = hintStyle.Render("← →  mode   ↑ ↓  option   enter  start   esc  quit")
	} else {
		hint = hintStyle.Render("← →  mode   enter  start   esc  quit")
	}

	// Theme indicator
	themeLabel := lipgloss.NewStyle().Foreground(activeTheme.surface2).
		Render(fmt.Sprintf("ctrl+t  theme: %s", func() string {
			if isDark() { return "mocha" }
			return "latte"
		}()))

	body := lipgloss.JoinVertical(lipgloss.Center,
		logo, "", tagline, "", "", sectionLabel, "",
		modeRow+subRow, "", "", hint, "", themeLabel,
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)
}

// ── Typing view ───────────────────────────────────────────────────────────────

func (m Model) viewTyping() string {
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
			display := string(ch)
			if ch == '\t' {
				display = "    "
			}

			if absPos < len(m.input) {
				typed := m.input[absPos]
				correct := typed == ch
				if m.blindMode {
					// Blind mode: neutral dot for correct, X for wrong
					if correct {
						sb.WriteString(lipgloss.NewStyle().Foreground(activeTheme.surface2).Render("·"))
					} else {
						sb.WriteString(lipgloss.NewStyle().Foreground(activeTheme.wrong).Render("✗"))
					}
				} else {
					if correct {
						// Dim the correct chars slightly — monkeytype feel
						sb.WriteString(lipgloss.NewStyle().Foreground(activeTheme.subtext1).Render(display))
					} else {
						// Red bg on wrong chars, show what was typed
						d := string(typed)
						if ch == ' ' || ch == '\t' || ch == '\n' {
							d = "·"
						}
						sb.WriteString(lipgloss.NewStyle().
							Foreground(activeTheme.wrong).
							Background(activeTheme.heatBg4).
							Render(d))
					}
				}
			} else if absPos == cursorPos {
				// Solid block cursor — background colour, not underline
				sb.WriteString(cursorStyle.Render(display))
			} else {
				// Pending text
				if m.mode == modeCode && m.hlMap != nil {
					sb.WriteString(m.pendingWithHL(absPos, display))
				} else {
					sb.WriteString(pendingStyle.Render(display))
				}
			}
		}
		renderedLines = append(renderedLines, sb.String())
	}

	textBlock := strings.Join(renderedLines, "\n")

	// Stats bar
	var timerPart string
	if m.mode == modeTime {
		col := timeStyle
		if m.timeLeft <= 10 {
			col = incorrectStyle
		}
		timerPart = "   " + col.Render(fmt.Sprintf("%ds", m.timeLeft))
	}
	var blindTag string
	if m.blindMode {
		blindTag = "   " + pbStyle.Render(" blind ")
	}
	var langTag string
	if m.mode == modeCode {
		langTag = "   " + subtleStyle.Render(langKeys[m.langIdx])
	}

	// Progress bar — filled with thin unicode blocks
	progress := m.renderProgress()

	var parts []string
	if !m.focusMode {
		// Stats row: big WPM on left, acc centre, timer/lang/tags right
		wpmNum   := wpmStyle.Render(fmt.Sprintf("%.0f", m.calcWPM()))
		wpmLabel := subtleStyle.Render(" wpm")
		accNum   := accStyle.Render(fmt.Sprintf("%.0f", m.calcAccuracy()))
		accLabel := subtleStyle.Render("% acc")
		statsLeft  := lipgloss.JoinHorizontal(lipgloss.Top, wpmNum, wpmLabel)
		statsRight := lipgloss.JoinHorizontal(lipgloss.Top, accNum, accLabel, timerPart, langTag, blindTag)
		spacer := strings.Repeat(" ", 4)
		stats := lipgloss.JoinHorizontal(lipgloss.Top, statsLeft, spacer, statsRight)
		parts = append(parts, stats, "")

		switch m.mode {
		case modeQuote:
			parts = append(parts, subtleStyle.Render("— "+m.activeQuote.Author))
		case modeCode:
			parts = append(parts, subtleStyle.Render(langKeys[m.langIdx]+" snippet"))
		}
	}

	parts = append(parts, progress, textBlock, "")

	if !m.focusMode {
		parts = append(parts, hintStyle.Render("tab  enter  ctrl+r  ctrl+b  ctrl+f  ctrl+t  esc"))
	}

	body := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)
}

// pendingWithHL returns a styled string for a pending char using the Chroma StyleMap.
func (m Model) pendingWithHL(pos int, display string) string {
	if pos < len(m.hlMap) {
		return m.hlMap[pos].Render(display)
	}
	return pendingStyle.Render(display)
}

// ── Results view ──────────────────────────────────────────────────────────────

func (m Model) viewResults() string {
	dur := m.activeDuration()
	pb := personalBest(m.modeKey(), m.langKey(), dur)

	// ── Stat blocks: big number + small label below ───────────────────────
	// Use a large bold style for the numbers to give visual weight.
	numStyle := lipgloss.NewStyle().Bold(true)

	wpmNum  := numStyle.Foreground(activeTheme.wpm).Render(fmt.Sprintf("%.0f", m.finalWPM))
	accNum  := numStyle.Foreground(activeTheme.acc).Render(fmt.Sprintf("%.1f%%", m.finalAcc))
	timeNum := numStyle.Foreground(activeTheme.timer).Render(fmt.Sprintf("%.1fs", m.elapsed.Seconds()))
	pbNum   := numStyle.Foreground(activeTheme.subtle).Render(fmt.Sprintf("%.0f", pb))

	pbBadge := ""
	if m.isPB {
		pbBadge = "  " + pbStyle.Render(" new best! ")
	}

	colW := 12
	wpmCol  := lipgloss.NewStyle().Width(colW).Render(
		lipgloss.JoinVertical(lipgloss.Left, wpmNum+pbBadge, subtleStyle.Render("wpm")))
	accCol  := lipgloss.NewStyle().Width(colW).Render(
		lipgloss.JoinVertical(lipgloss.Left, accNum,  subtleStyle.Render("acc")))
	timeCol := lipgloss.NewStyle().Width(colW).Render(
		lipgloss.JoinVertical(lipgloss.Left, timeNum, subtleStyle.Render("time")))
	pbCol   := lipgloss.NewStyle().Width(colW).Render(
		lipgloss.JoinVertical(lipgloss.Left, pbNum,   subtleStyle.Render("best")))

	statsRow := lipgloss.JoinHorizontal(lipgloss.Bottom, wpmCol, accCol, timeCol, pbCol)

	// ── Divider ───────────────────────────────────────────────────────────
	divW := colW * 4
	div  := subtleStyle.Render(strings.Repeat("─", divW))

	// ── Vertical bar chart ────────────────────────────────────────────────
	chartRows := m.renderBarChart(divW)

	// ── Keyboard heatmap ──────────────────────────────────────────────────
	kbRows := m.renderKeyboard()

	// ── Actions ───────────────────────────────────────────────────────────
	sep := hintStyle.Render("  ·  ")
	actions := lipgloss.JoinHorizontal(lipgloss.Top,
		pendingStyle.Render("enter"), hintStyle.Render(" again"),
		sep, pendingStyle.Render("m"), hintStyle.Render(" menu"),
		sep, pendingStyle.Render("h"), hintStyle.Render(" history"),
		sep, pendingStyle.Render("j"), hintStyle.Render(" json"),
		sep, pendingStyle.Render("c"), hintStyle.Render(" csv"),
		sep, pendingStyle.Render("esc"), hintStyle.Render(" quit"),
	)

	var exportLine string
	if m.exportMsg != "" {
		exportLine = "" + m.exportMsg
	}

	parts := []string{"", statsRow, "", div, ""}
	parts = append(parts, chartRows...)
	if len(kbRows) > 0 {
		parts = append(parts, "", hintStyle.Render("mistakes"))
		parts = append(parts, kbRows...)
	}
	parts = append(parts, "", actions)
	if exportLine != "" {
		parts = append(parts, exportLine)
	}

	body := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)
}

// renderProgress draws a polished progress bar using ━ characters.
func (m Model) renderProgress() string {
	total := len(m.target)
	if total == 0 {
		return ""
	}
	done := len(m.input)
	if done > total {
		done = total
	}
	width := 50
	filled := int(float64(done) / float64(total) * float64(width))
	pct := int(float64(done) / float64(total) * 100)

	filledBar := lipgloss.NewStyle().Foreground(activeTheme.mauve).Render(strings.Repeat("━", filled))
	emptyBar  := lipgloss.NewStyle().Foreground(activeTheme.surface1).Render(strings.Repeat("━", width-filled))
	pctLabel  := lipgloss.NewStyle().Foreground(activeTheme.subtext0).Render(fmt.Sprintf(" %d%%", pct))

	return filledBar + emptyBar + pctLabel
}

// renderBarChart renders a vertical bar chart of wpmSamples.
// Each cell is individually styled — no string-replacement hacks.
func (m Model) renderBarChart(width int) []string {
	samples := m.wpmSamples
	chartH := 7
	chartW := width
	if chartW < 8 { chartW = 8 }

	if len(samples) == 0 {
		return []string{hintStyle.Render("no data yet")}
	}

	// Resample to chartW columns
	cols := make([]float64, chartW)
	for i := range cols {
		idx := int(float64(i) / float64(chartW) * float64(len(samples)))
		if idx >= len(samples) { idx = len(samples) - 1 }
		cols[i] = samples[idx]
	}

	maxV, minV := cols[0], cols[0]
	peakIdx := 0
	for i, v := range cols {
		if v > maxV { maxV = v; peakIdx = i }
		if v < minV { minV = v }
	}
	if maxV == 0 { maxV = 1 }

	// height of each column (0..chartH)
	heights := make([]int, chartW)
	for i, v := range cols { heights[i] = int(v / maxV * float64(chartH)) }

	// Build each row by styling each cell individually
	rows := make([]string, chartH)
	for row := 0; row < chartH; row++ {
		threshold := chartH - row // filled if height >= threshold
		var sb strings.Builder
		for ci, h := range heights {
			if h >= threshold {
				if ci == peakIdx {
					sb.WriteString(sparkPeakStyle.Render("█"))
				} else if threshold <= chartH/2 {
					sb.WriteString(sparkBarStyle.Render("█"))
				} else {
					// Upper portion — dimmer fill
					sb.WriteString(lipgloss.NewStyle().Foreground(activeTheme.surface2).Render("▓"))
				}
			} else {
				sb.WriteString(lipgloss.NewStyle().Foreground(activeTheme.surface0).Render("░"))
			}
		}
		rows[row] = sb.String()
	}

	// Y-axis labels
	rows[0]        += "  " + lipgloss.NewStyle().Foreground(activeTheme.wpm).Bold(true).Render(fmt.Sprintf("%.0f", maxV))
	rows[chartH/2] += "  " + hintStyle.Render(fmt.Sprintf("%.0f", (maxV+minV)/2))
	rows[chartH-1] += "  " + hintStyle.Render(fmt.Sprintf("%.0f", minV))

	// X axis + time labels
	axis := lipgloss.NewStyle().Foreground(activeTheme.surface1).Render(strings.Repeat("─", chartW))
	pad  := chartW - 6
	if pad < 0 { pad = 0 }
	tLabel := hintStyle.Render(fmt.Sprintf("0s%s%.0fs", strings.Repeat(" ", pad), float64(len(samples))))

	label := lipgloss.NewStyle().Foreground(activeTheme.subtext0).Render("wpm over time")
	result := []string{label}
	result = append(result, rows...)
	result = append(result, axis, tLabel)
	return result
}

// renderKeyboard renders a QWERTY heatmap using keyHeatStyle from styles.go.
// Returns one rendered string per keyboard row.
func (m Model) renderKeyboard() []string {
	if len(m.mistakeMap) == 0 {
		return nil
	}
	maxCount := 0
	for _, n := range m.mistakeMap {
		if n > maxCount { maxCount = n }
	}

	countFor := func(r rune) int { return m.mistakeMap[r] }

	renderKey := func(label string, count int) string {
		return keyHeatStyle(count, maxCount).Render(label)
	}

	kbRows := [][]struct{ label string; r rune }{
		{{"q",'q'},{" w",'w'},{" e",'e'},{" r",'r'},{" t",'t'},{" y",'y'},{" u",'u'},{" i",'i'},{" o",'o'},{" p",'p'}},
		{{" a",'a'},{" s",'s'},{" d",'d'},{" f",'f'},{" g",'g'},{" h",'h'},{" j",'j'},{" k",'k'},{" l",'l'}},
		{{"  z",'z'},{" x",'x'},{" c",'c'},{" v",'v'},{" b",'b'},{" n",'n'},{" m",'m'}},
	}

	var rows []string
	for _, row := range kbRows {
		var sb strings.Builder
		for _, key := range row {
			sb.WriteString(renderKey(key.label, countFor(key.r)))
		}
		rows = append(rows, sb.String())
	}

	// Special keys
	spRow := lipgloss.JoinHorizontal(lipgloss.Top,
		renderKey("  space  ", countFor(' ')),
		renderKey("  tab ", countFor('\t')),
		renderKey("  ret ", countFor('\n')),
	)
	rows = append(rows, spRow)
	return rows
}

// ── Time input view ──────────────────────────────────────────────────────────

func (m Model) viewTimeInput() string {
	title := titleStyle.Render("custom time")
	sub := lipgloss.NewStyle().Foreground(activeTheme.subtext0).Render("enter a duration in seconds  (max 3600 = 1 hour)")

	// Display the digits typed so far, with cursor
	display := m.customTimeStr
	if display == "" {
		display = " "
	}
	var dispSecs string
	if m.customTimeStr != "" {
		secs := 0
		for _, ch := range m.customTimeStr {
			secs = secs*10 + int(ch-'0')
		}
		if secs > 3600 { secs = 3600 }
		mins := secs / 60
		rem  := secs % 60
		if mins > 0 {
			dispSecs = fmt.Sprintf("%dm %02ds", mins, rem)
		} else {
			dispSecs = fmt.Sprintf("%ds", secs)
		}
	}

	inputBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(activeTheme.mauve).
		Padding(0, 2).
		Width(16).
		Align(lipgloss.Center).
		Render(
			wpmStyle.Render(display) +
			lipgloss.NewStyle().Foreground(activeTheme.mauve).Render("█"),
		)

	var convLine string
	if dispSecs != "" {
		convLine = subtleStyle.Render("= " + dispSecs)
	}

	hint := hintStyle.Render("digits  ·  backspace  ·  enter confirm  ·  esc cancel")

	body := lipgloss.JoinVertical(lipgloss.Center,
		title, "", sub, "", inputBox, convLine, "", hint,
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)
}

// ── History view ──────────────────────────────────────────────────────────────

func (m Model) viewHistory() string {
	title := titleStyle.Render("session history")

	if len(m.histData) == 0 {
		body := lipgloss.JoinVertical(lipgloss.Center,
			title, "",
			subtleStyle.Render("no sessions yet"),
			"", hintStyle.Render("esc → back"),
		)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)
	}

	col := func(s lipgloss.Style, w int, text string) string {
		return lipgloss.NewStyle().Width(w).Inline(true).Render(s.Render(text))
	}
	dimLabel := lipgloss.NewStyle().Foreground(activeTheme.surface2)
	header := lipgloss.JoinHorizontal(lipgloss.Top,
		col(dimLabel, 8,  "wpm"),
		col(dimLabel, 8,  "acc%"),
		col(dimLabel, 12, "mode"),
		col(dimLabel, 8,  "lang"),
		col(dimLabel, 14, "date"),
	)
	divider := lipgloss.NewStyle().Foreground(activeTheme.surface0).Render(strings.Repeat("─", 52))

	end := m.histOffset + histPageSize
	if end > len(m.histData) {
		end = len(m.histData)
	}

	var rows []string
	for _, e := range m.histData[m.histOffset:end] {
		modeLabel := e.Mode
		if e.Duration > 0 {
			modeLabel += fmt.Sprintf("/%ds", e.Duration)
		}
		lang := e.Lang
		if lang == "" {
			lang = "—"
		}
		wpmS  := lipgloss.NewStyle().Foreground(activeTheme.wpm).Bold(true)
		accS  := lipgloss.NewStyle().Foreground(activeTheme.acc)
		modeS := lipgloss.NewStyle().Foreground(activeTheme.text)
		langS := lipgloss.NewStyle().Foreground(activeTheme.timer)
		dateS := lipgloss.NewStyle().Foreground(activeTheme.overlay0)
		row := lipgloss.JoinHorizontal(lipgloss.Top,
			col(wpmS,  8,  fmt.Sprintf("%.0f", e.WPM)),
			col(accS,  8,  fmt.Sprintf("%.1f%%", e.Accuracy)),
			col(modeS, 12, modeLabel),
			col(langS, 8,  lang),
			col(dateS, 14, e.At.Format("Jan 02 15:04")),
		)
		rows = append(rows, row)
	}

	scroll := fmt.Sprintf("%d–%d of %d", m.histOffset+1, end, len(m.histData))
	nav := subtleStyle.Render(scroll + "   j/k scroll · esc back")

	parts := []string{title, "", header, divider}
	parts = append(parts, rows...)
	parts = append(parts, "", nav)

	body := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)
}
