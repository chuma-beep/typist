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
	stateMenu appState = iota
	stateTyping
	stateResults
	stateHistory
	stateConfirm
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

type tickMsg time.Time
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
	langIdx       int
	activeSnippet Snippet

	target      []rune
	input       []rune
	activeQuote Quote
	startTime   time.Time
	elapsed     time.Duration
	started     bool

	blindMode bool
	focusMode bool // hide stats while typing
	darkTheme bool // true = mocha, false = latte

	totalKeys int
	errors    int
	// mistake heatmap: count errors per expected rune
	mistakeMap map[rune]int

	// WPM sampling: one sample per second
	wpmSamples []float64
	lastSample time.Time

	// frozen results
	finalWPM  float64
	finalAcc  float64
	isPB      bool
	exportMsg string

	confirmQuit bool

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
		m.timeLeft = timeLimits[m.timeLimitIdx]
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

// ── Init ──────────────────────────────────────────────────────────────────────

func (m Model) Init() tea.Cmd { return nil }

// ── Update ────────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		if m.state == stateTyping && m.started {
			// Record WPM sample every second
			m.wpmSamples = append(m.wpmSamples, m.calcWPM())
			m.lastSample = time.Time(msg)

			if m.mode == modeTime {
				m.timeLeft--
				if m.timeLeft <= 0 {
					return m.finishTest(), nil
				}
			}
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
		case stateConfirm:
			return m.updateConfirm(msg)
		}
	}
	return m, nil
}

// ── Menu ──────────────────────────────────────────────────────────────────────

func (m Model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	subCount := m.subRowCount()
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.state = stateConfirm
		return m, nil
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
		return len(timeLimits)
	case modeCode:
		return len(langKeys)
	}
	return 0
}

func (m *Model) applySubRow() {
	switch m.mode {
	case modeTime:
		m.timeLimitIdx = m.menuCol
	case modeCode:
		m.langIdx = m.menuCol
	}
}

// ── Typing ────────────────────────────────────────────────────────────────────

func (m Model) updateTyping(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.state = stateConfirm
		return m, nil
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

	dur := 0
	if m.mode == modeTime {
		dur = timeLimits[m.timeLimitIdx]
	}
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
		m.state = stateConfirm
		return m, nil
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

// ── Confirm ────────────────────────────────────────────────────────────────────

func (m Model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		return m, nil
	case tea.KeyEnter, tea.KeyRunes:
		switch string(msg.Runes) {
		case "y", "Y", "q", "Q":
			return m, tea.Quit
		case "n", "N":
			m.state = stateMenu
			m.confirmQuit = false
		default:
			return m, nil
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
	case stateConfirm:
		return m.viewConfirm()
	}
	return ""
}

// ── Menu view ─────────────────────────────────────────────────────────────────

func (m Model) viewMenu() string {
	// Logo
	logo := titleStyle.Render("typist")
	tagline := subtleStyle.Render("offline · open source · no paywall")

	// Mode pills — monkeytype style: all on one row, selected is bright
	var modeBtns []string
	for i, label := range modeNames {
		if i == int(m.mode) {
			modeBtns = append(modeBtns, selectedStyle.Render(label))
		} else {
			modeBtns = append(modeBtns, optionStyle.Render(label))
		}
	}
	modeRow := lipgloss.JoinHorizontal(lipgloss.Center, modeBtns...)

	// Sub-options (time durations or language picker) — dimmer row below
	var subRow string
	switch m.mode {
	case modeTime:
		var btns []string
		for i, t := range timeLimits {
			label := fmt.Sprintf("%d", t)
			s := optionStyle
			if i == m.timeLimitIdx {
				s = dimSelectedStyle
				if m.menuRow == 1 {
					s = selectedStyle
				}
			}
			btns = append(btns, s.Render(label))
		}
		subRow = "\n" + lipgloss.JoinHorizontal(lipgloss.Center, btns...)
	case modeCode:
		var btns []string
		for i, lang := range langKeys {
			s := optionStyle
			if i == m.langIdx {
				s = dimSelectedStyle
				if m.menuRow == 1 {
					s = selectedStyle
				}
			}
			btns = append(btns, s.Render(lang))
		}
		subRow = "\n" + lipgloss.JoinHorizontal(lipgloss.Center, btns...)
	}

	var hint string
	if m.mode == modeTime || m.mode == modeCode {
		hint = hintStyle.Render("← →  mode   ↑ ↓  option   enter  start   esc  quit")
	} else {
		hint = hintStyle.Render("← →  mode   enter  start   esc  quit")
	}

	body := lipgloss.JoinVertical(lipgloss.Center,
		logo,
		tagline,
		"",
		"",
		modeRow+subRow,
		"",
		"",
		hint,
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
					if correct {
						sb.WriteString(correctStyle.Render("·"))
					} else {
						sb.WriteString(incorrectStyle.Render("·"))
					}
				} else {
					if correct {
						sb.WriteString(correctStyle.Render(display))
					} else {
						d := string(typed)
						if ch == ' ' || ch == '\t' || ch == '\n' {
							d = "·"
						}
						sb.WriteString(incorrectStyle.Render(d))
					}
				}
			} else if absPos == cursorPos {
				sb.WriteString(cursorStyle.Render(display))
			} else {
				// Pending — apply syntax highlighting in code mode
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
		wpmNum := wpmStyle.Render(fmt.Sprintf("%.0f", m.calcWPM()))
		wpmLabel := subtleStyle.Render(" wpm")
		accNum := accStyle.Render(fmt.Sprintf("%.0f", m.calcAccuracy()))
		accLabel := subtleStyle.Render("% acc")
		statsLeft := lipgloss.JoinHorizontal(lipgloss.Top, wpmNum, wpmLabel)
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
	header := titleStyle.Render("results")

	pbBadge := ""
	if m.isPB {
		pbBadge = pbStyle.Render("  new best! ")
	}

	bigWPM := lipgloss.NewStyle().
		Foreground(activeTheme.wpm).
		Bold(true).
		Width(20).
		Render(fmt.Sprintf("%.0f", m.finalWPM))

	bigAcc := lipgloss.NewStyle().
		Foreground(activeTheme.acc).
		Bold(true).
		Width(12).
		Render(fmt.Sprintf("%.1f%%", m.finalAcc))

	bigTime := lipgloss.NewStyle().
		Foreground(activeTheme.timer).
		Bold(true).
		Width(12).
		Render(fmt.Sprintf("%.1fs", m.elapsed.Seconds()))

	statsLine := lipgloss.JoinHorizontal(lipgloss.Bottom,
		bigWPM+pbBadge,
		bigAcc+subtleStyle.Render("  acc"),
		bigTime+subtleStyle.Render("  time"),
	)

	divW := 50
	div := subtleStyle.Render(strings.Repeat("─", divW))

	chartRows := m.renderBarChart(divW)

	kbRows := m.renderKeyboard()

	sep := hintStyle.Render(" · ")
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
		exportLine = "\n" + m.exportMsg
	}

	parts := []string{header, "", statsLine, "", div}
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

// renderProgress draws a thin progress bar showing completion through the text.
func (m Model) renderProgress() string {
	total := len(m.target)
	if total == 0 {
		return ""
	}
	done := len(m.input)
	if done > total {
		done = total
	}
	width := 40
	filled := int(float64(done) / float64(total) * float64(width))
	bar := strings.Repeat("─", filled) + strings.Repeat(" ", width-filled)
	pct := int(float64(done) / float64(total) * 100)
	return hintStyle.Render("│") +
		subtleStyle.Render(bar) +
		hintStyle.Render("│") +
		hintStyle.Render(fmt.Sprintf(" %d%%", pct))
}

// renderBarChart renders a vertical bar chart of wpmSamples.
// Returns a slice of strings — one per row — so the caller can join them.
func (m Model) renderBarChart(width int) []string {
	samples := m.wpmSamples
	chartH := 6 // rows tall
	chartW := width

	if len(samples) == 0 {
		return []string{subtleStyle.Render("no data yet")}
	}

	// Sample down/up to chartW columns
	cols := make([]float64, chartW)
	for i := range cols {
		idx := int(float64(i) / float64(chartW) * float64(len(samples)))
		if idx >= len(samples) {
			idx = len(samples) - 1
		}
		cols[i] = samples[idx]
	}

	maxV := 0.0
	minV := cols[0]
	peakIdx := 0
	for i, v := range cols {
		if v > maxV {
			maxV = v
			peakIdx = i
		}
		if v < minV {
			minV = v
		}
	}
	if maxV == 0 {
		maxV = 1
	}

	// Heights: 0..chartH for each column
	heights := make([]int, chartW)
	for i, v := range cols {
		heights[i] = int(v / maxV * float64(chartH))
	}

	// Build rows top→bottom
	rows := make([]string, chartH)
	for row := 0; row < chartH; row++ {
		// row 0 = top, row chartH-1 = bottom
		threshold := chartH - row // bar must be >= threshold to fill this cell
		var sb strings.Builder
		for ci, h := range heights {
			if h >= threshold {
				if ci == peakIdx {
					sb.WriteString(sparkPeakStyle.Render("█"))
				} else {
					// Shade bars: brighter nearer the top
					if threshold > chartH/2 {
						sb.WriteString(subtleStyle.Render("▓"))
					} else {
						sb.WriteString(sparkBarStyle.Render("█"))
					}
				}
			} else {
				sb.WriteString(hintStyle.Render("░"))
			}
		}
		rows[row] = sb.String()
	}

	// Y-axis labels (right side)
	rows[0] += "  " + subtleStyle.Render(fmt.Sprintf("%.0f wpm", maxV))
	rows[chartH/2] += "  " + hintStyle.Render(fmt.Sprintf("%.0f", (maxV+minV)/2))
	rows[chartH-1] += "  " + hintStyle.Render(fmt.Sprintf("%.0f", minV))

	// X-axis (time labels)
	xAxis := strings.Repeat("─", chartW)
	timeLabel := fmt.Sprintf("0s%s%.0fs", strings.Repeat(" ", chartW-6), float64(len(samples)))
	result := []string{hintStyle.Render("wpm")}
	result = append(result, rows...)
	result = append(result, hintStyle.Render(xAxis))
	result = append(result, subtleStyle.Render(timeLabel))
	return result
}

// renderKeyboard renders a simplified QWERTY heatmap — rows of keys
// colored by mistake frequency. Returns one string per keyboard row.
func (m Model) renderKeyboard() []string {
	if len(m.mistakeMap) == 0 {
		return nil
	}

	// Find max for normalisation
	maxCount := 0
	for _, n := range m.mistakeMap {
		if n > maxCount {
			maxCount = n
		}
	}
	if maxCount == 0 {
		maxCount = 1
	}

	kbLayout := [][]string{
		{"q", "w", "e", "r", "t", "y", "u", "i", "o", "p"},
		{"a", "s", "d", "f", "g", "h", "j", "k", "l"},
		{"z", "x", "c", "v", "b", "n", "m"},
	}
	special := map[rune]string{
		' ':  "spc",
		'\t': "tab",
		'\n': "ret",
	}

	// Build lookup: display string → count
	countFor := func(key string) int {
		// Try as rune
		r := []rune(key)
		if len(r) == 1 {
			return m.mistakeMap[r[0]]
		}
		return 0
	}
	// special keys stored as rune
	specCount := map[string]int{
		"spc": m.mistakeMap[' '],
		"tab": m.mistakeMap['\t'],
		"ret": m.mistakeMap['\n'],
	}
	_ = special

	renderKey := func(label string, count int) string {
		h := float64(count) / float64(maxCount)
		var style lipgloss.Style
		switch {
		case count == 0:
			style = lipgloss.NewStyle().
				Foreground(activeTheme.hint).
				Background(activeTheme.pending).
				Padding(0, 1)
		case h < 0.4:
			style = lipgloss.NewStyle().
				Foreground(activeTheme.hlNum).
				Background(lipgloss.Color("#313244")).
				Padding(0, 1)
		default:
			style = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#1e1e2e")).
				Background(activeTheme.wrong).
				Padding(0, 1).
				Bold(true)
		}
		return style.Render(label)
	}

	var rows []string
	for _, row := range kbLayout {
		var sb strings.Builder
		for i, k := range row {
			if i > 0 {
				sb.WriteString(" ")
			}
			sb.WriteString(renderKey(k, countFor(k)))
		}
		rows = append(rows, sb.String())
	}

	// Special keys row
	var spRow strings.Builder
	for i, k := range []string{"spc", "tab", "ret"} {
		if i > 0 {
			spRow.WriteString("  ")
		}
		spRow.WriteString(renderKey(k, specCount[k]))
	}
	rows = append(rows, spRow.String())
	return rows
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

	header := lipgloss.JoinHorizontal(lipgloss.Top,
		subtleStyle.Render(fmt.Sprintf("%-8s", "wpm")),
		subtleStyle.Render(fmt.Sprintf("%-8s", "acc%")),
		subtleStyle.Render(fmt.Sprintf("%-12s", "mode")),
		subtleStyle.Render(fmt.Sprintf("%-8s", "lang")),
		subtleStyle.Render("date"),
	)
	divider := subtleStyle.Render(strings.Repeat("─", 50))

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
		row := lipgloss.JoinHorizontal(lipgloss.Top,
			wpmStyle.Render(fmt.Sprintf("%-8.0f", e.WPM)),
			accStyle.Render(fmt.Sprintf("%-8.1f", e.Accuracy)),
			pendingStyle.Render(fmt.Sprintf("%-12s", modeLabel)),
			timeStyle.Render(fmt.Sprintf("%-8s", lang)),
			hintStyle.Render(e.At.Format("Jan 02 15:04")),
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

func (m Model) viewConfirm() string {
	body := lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render("quit?"),
		"",
		subtleStyle.Render("any progress will be lost"),
		"",
		hintStyle.Render("enter / y / q")+subtleStyle.Render("  quit"),
		hintStyle.Render("esc / n")+subtleStyle.Render("       cancel"),
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)
}
