package main

import (
	"fmt"
	"sort"
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
	stateHistory
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
	lineWidth    = 60
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

	timeLimitIdx int
	timeLeft     int
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
	}
	return ""
}

// ── Menu view ─────────────────────────────────────────────────────────────────

func (m Model) viewMenu() string {
	title := titleStyle.Render("typist")
	sub := subtleStyle.Render("offline · open source · no paywall")

	var modeBtns []string
	for i, label := range modeNames {
		if i == int(m.mode) {
			modeBtns = append(modeBtns, selectedStyle.Render(" "+label+" "))
		} else {
			modeBtns = append(modeBtns, optionStyle.Render(" "+label+" "))
		}
	}
	modeRow := lipgloss.JoinHorizontal(lipgloss.Center, modeBtns...)

	var subRow string
	switch m.mode {
	case modeTime:
		var btns []string
		for i, t := range timeLimits {
			label := fmt.Sprintf("%ds", t)
			s := optionStyle
			if i == m.timeLimitIdx {
				s = dimSelectedStyle
				if m.menuRow == 1 {
					s = selectedStyle
				}
			}
			btns = append(btns, s.Render(" "+label+" "))
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
			btns = append(btns, s.Render(" "+lang+" "))
		}
		subRow = "\n" + lipgloss.JoinHorizontal(lipgloss.Center, btns...)
	}

	hint := subtleStyle.Render("← → switch")
	if m.mode == modeTime || m.mode == modeCode {
		hint = subtleStyle.Render("← → switch · ↑ ↓ row · enter start · esc quit")
	} else {
		hint = subtleStyle.Render("← → switch · enter start · esc quit")
	}

	body := lipgloss.JoinVertical(lipgloss.Center,
		title, sub, "", modeRow+subRow, "", hint,
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

	var parts []string
	if !m.focusMode {
		stats := lipgloss.JoinHorizontal(lipgloss.Top,
			wpmStyle.Render(fmt.Sprintf("%.0f", m.calcWPM())),
			subtleStyle.Render(" wpm   "),
			accStyle.Render(fmt.Sprintf("%.0f%%", m.calcAccuracy())),
			subtleStyle.Render(" acc"),
			timerPart, langTag, blindTag,
		)
		parts = append(parts, stats, "")

		var meta string
		switch m.mode {
		case modeQuote:
			meta = subtleStyle.Render("— " + m.activeQuote.Author)
		case modeCode:
			meta = subtleStyle.Render(langKeys[m.langIdx] + " · tab+enter live")
		}
		if meta != "" {
			parts = append(parts, meta)
		}
	}

	hint := hintStyle.Render("ctrl+r restart · ctrl+b blind · ctrl+f focus · ctrl+t theme · esc quit")
	parts = append(parts, textBlock, "", hint)

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
	title := titleStyle.Render("results")

	pbTag := ""
	if m.isPB {
		pbTag = "  " + pbStyle.Render(" new best! ")
	}

	wpmLine := lipgloss.JoinHorizontal(lipgloss.Top,
		wpmStyle.Render(fmt.Sprintf("%-8.0f", m.finalWPM)),
		subtleStyle.Render("wpm"), pbTag,
	)
	accLine := lipgloss.JoinHorizontal(lipgloss.Top,
		accStyle.Render(fmt.Sprintf("%-8.1f", m.finalAcc)),
		subtleStyle.Render("accuracy"),
	)
	timeLine := lipgloss.JoinHorizontal(lipgloss.Top,
		timeStyle.Render(fmt.Sprintf("%-8.1f", m.elapsed.Seconds())),
		subtleStyle.Render("seconds"),
	)

	dur := 0
	if m.mode == modeTime {
		dur = timeLimits[m.timeLimitIdx]
	}
	pb := personalBest(m.modeKey(), m.langKey(), dur)
	pbLine := lipgloss.JoinHorizontal(lipgloss.Top,
		subtleStyle.Render(fmt.Sprintf("%-8.0f", pb)),
		subtleStyle.Render("personal best"),
	)

	// WPM sparkline
	sparkLine := m.renderSparkline()

	// Mistake heatmap (top 6 most missed chars)
	heatLine := m.renderMistakeHeat()

	card := cardStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		title, "",
		wpmLine, accLine, timeLine, "", pbLine, "",
		subtleStyle.Render("wpm over time"),
		sparkLine,
		heatLine,
	))

	actions := lipgloss.JoinVertical(lipgloss.Left,
		pendingStyle.Render("enter / r  → again"),
		pendingStyle.Render("m          → menu"),
		pendingStyle.Render("h          → history"),
		pendingStyle.Render("j / c      → export json / csv"),
		hintStyle.Render("esc        → quit"),
	)

	var exportLine string
	if m.exportMsg != "" {
		exportLine = "\n" + m.exportMsg
	}

	body := lipgloss.JoinVertical(lipgloss.Center, card, "", actions, exportLine)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)
}

// renderSparkline builds an ASCII bar chart from wpmSamples.
func (m Model) renderSparkline() string {
	samples := m.wpmSamples
	if len(samples) == 0 {
		return subtleStyle.Render("no data")
	}

	// Downsample / upsample to sparkWidth
	bars := make([]float64, sparkWidth)
	for i := range bars {
		idx := int(float64(i) / float64(sparkWidth) * float64(len(samples)))
		if idx >= len(samples) {
			idx = len(samples) - 1
		}
		bars[i] = samples[idx]
	}

	maxWPM := 0.0
	for _, v := range bars {
		if v > maxWPM {
			maxWPM = v
		}
	}
	if maxWPM == 0 {
		maxWPM = 1
	}

	var sb strings.Builder
	for i, v := range bars {
		normalized := v / maxWPM
		barIdx := int(normalized * float64(len(sparkBars)-1))
		if barIdx < 0 {
			barIdx = 0
		}
		ch := string(sparkBars[barIdx])
		// Highlight peak
		if v == maxWPM {
			sb.WriteString(sparkPeakStyle.Render(ch))
		} else {
			sb.WriteString(sparkBarStyle.Render(ch))
		}
		_ = i
	}

	// Min/max labels
	minWPM := bars[0]
	for _, v := range bars {
		if v < minWPM {
			minWPM = v
		}
	}
	label := subtleStyle.Render(fmt.Sprintf(" %.0f–%.0f wpm", minWPM, maxWPM))
	return sb.String() + label
}

// renderMistakeHeat shows the top missed characters.
func (m Model) renderMistakeHeat() string {
	if len(m.mistakeMap) == 0 {
		return ""
	}

	entries := make([]mistakeEntry, 0, len(m.mistakeMap))
	for ch, count := range m.mistakeMap {
		entries = append(entries, mistakeEntry{ch, count})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})
	if len(entries) > 6 {
		entries = entries[:6]
	}

	var sb strings.Builder
	sb.WriteString(subtleStyle.Render("missed: "))
	for _, e := range entries {
		label := string(e.ch)
		switch e.ch {
		case ' ':
			label = "spc"
		case '\t':
			label = "tab"
		case '\n':
			label = "ret"
		}
		// Color intensity by count
		style := hlComment
		if e.count >= 5 {
			style = incorrectStyle
		} else if e.count >= 2 {
			style = hlNumber
		}
		sb.WriteString(style.Render(fmt.Sprintf("[%s×%d] ", label, e.count)))
	}
	return sb.String()
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
