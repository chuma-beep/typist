package main

import (
	"fmt"
	"math"
	"sort"
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
	stateTimeInput   // custom time entry
	stateConfirmQuit // esc confirmation dialog
	stateGame        // game mode active
	stateGameOver    // game over screen
)

type testMode int

const (
	modeWords testMode = iota
	modeTime
	modeQuote
	modeCode
	modeGame
)

var modeNames = []string{"words", "time", "quote", "code", "game"}
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

	timeLimitIdx   int
	timeLeft       int
	customTimeSecs int      // 0 = use timeLimits preset
	customTimeStr  string   // digits being typed in stateTimeInput
	prevState      appState // state to return to if quit is cancelled
	langIdx        int
	activeSnippet  Snippet

	target      []rune
	input       []rune
	activeQuote Quote
	startTime   time.Time
	elapsed     time.Duration
	started     bool

	blindMode bool
	focusMode bool // hide stats while typing
	themeIdx  int  // index into themes slice

	totalKeys     int
	errors        int
	rawCharsTyped int // every keypress including backspace — for raw WPM
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

	gameState *GameState
}

func NewModel() Model {
	return Model{
		state:        stateMenu,
		mode:         modeWords,
		timeLimitIdx: 1,
		langIdx:      0,
		mistakeMap:   make(map[rune]int),
		themeIdx:     0,
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
	m.rawCharsTyped = 0
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

	case gameTickMsg:
		if m.state == stateGame && m.gameState != nil {
			m.gameState.tick()
			if m.gameState.over {
				m.state = stateGameOver
				return m, nil
			}
			return m, gameTickCmd()
		}
		return m, nil
	case tickMsg:
		if m.state != stateTyping {
			break
		}
		if m.mode == modeTime {
			// Keep ticking so the timer is ready the moment typing starts.
			// Only decrement after the first keypress.
			if m.started {
				m.wpmSamples = append(m.wpmSamples, m.calcWPM())
				m.lastSample = time.Time(msg)
				m.timeLeft--
				if m.timeLeft <= 0 {
					return m.finishTest(), nil
				}
			}
			return m, tickCmd()
		}
		// Non-time modes: sample WPM each second after typing starts
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
		case stateConfirmQuit:
			return m.updateConfirmQuit(msg)
		case stateGame:
			return m.updateGameMode(msg)
		case stateGameOver:
			return m.updateGameOver(msg)
		}
	}
	return m, nil
}

// toggleTheme cycles through all available themes and rebuilds any active hlMap.
func (m *Model) toggleTheme() {
	m.themeIdx = (m.themeIdx + 1) % len(themes)
	applyTheme(themes[m.themeIdx])
	if m.mode == modeCode && len(m.target) > 0 {
		m.hlMap = BuildStyleMap(string(m.target), langKeys[m.langIdx])
	}
}

// goToMenu resets to the main menu preserving user preferences.
func (m Model) goToMenu() Model {
	next := NewModel()
	next.width, next.height = m.width, m.height
	next.mode, next.timeLimitIdx, next.langIdx = m.mode, m.timeLimitIdx, m.langIdx
	next.customTimeSecs = m.customTimeSecs
	next.focusMode, next.themeIdx = m.focusMode, m.themeIdx
	return next
}

// ── Menu ──────────────────────────────────────────────────────────────────────

func (m Model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	subCount := m.subRowCount()
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		m.prevState = stateMenu
		m.state = stateConfirmQuit
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
	case tea.KeyCtrlT:
		m.toggleTheme()
		return m, nil
	case tea.KeyEnter:
		if m.mode == modeGame {
			m.gameState = newGameState()
			m.state = stateGame
			return m, gameTickCmd()
		}
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
	case modeGame:
		return 0
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
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyCtrlG:
		next := m.goToMenu()
		return next, nil
	case tea.KeyEsc:
		m.prevState = stateTyping
		m.state = stateConfirmQuit
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
		m.toggleTheme()
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
			m.rawCharsTyped++ // backspace costs a keystroke
		}
	case tea.KeySpace:
		m.rawCharsTyped++
		m = m.appendRune(' ')
	case tea.KeyTab:
		m.rawCharsTyped++
		m = m.appendRune('\t')
	case tea.KeyEnter:
		if m.mode == modeCode {
			m.rawCharsTyped++
			m = m.appendRune('\n')
		}
	case tea.KeyRunes:
		for _, r := range msg.Runes {
			m.rawCharsTyped++
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
	// Set state first so calcWPM uses m.elapsed (not a second time.Since call)
	m.state = stateResults
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
	return m
}

// ── Results ───────────────────────────────────────────────────────────────────

func (m Model) updateResults(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyCtrlG:
		next := m.goToMenu()
		return next, nil
	case tea.KeyCtrlT:
		m.toggleTheme()
		return m, nil
	case tea.KeyEsc:
		m.prevState = stateResults
		m.state = stateConfirmQuit
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

// ── Confirm quit ─────────────────────────────────────────────────────────────

func (m Model) updateConfirmQuit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyCtrlG:
		next := m.goToMenu()
		return next, nil
	case tea.KeyCtrlT:
		m.toggleTheme()
		return m, nil
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "y", "Y":
			return m, tea.Quit
		case "n", "N", "q", "Q":
			m.state = m.prevState
			return m, nil
		}
	case tea.KeyEsc, tea.KeyEnter:
		// Esc or Enter on the confirm screen = cancel (safe default)
		m.state = m.prevState
		return m, nil
	}
	return m, nil
}

// ── Time input ───────────────────────────────────────────────────────────────

// updateTimeInput handles the custom time entry screen.
// User types digits; enter confirms; esc cancels back to menu.
func (m Model) updateTimeInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyCtrlG:
		next := m.goToMenu()
		return next, nil
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
	case tea.KeyCtrlG:
		next := m.goToMenu()
		return next, nil
	case tea.KeyCtrlT:
		m.toggleTheme()
		return m, nil
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

// calcRawWPM calculates WPM counting every keystroke including backspaces.
// This is the "honest" speed — correct chars WPM is always shown alongside.
func (m Model) calcRawWPM() float64 {
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
	return float64(m.rawCharsTyped) / 5.0 / mins
}

// correctChars returns the number of correctly typed characters.
func (m Model) correctChars() int {
	correct := 0
	for i, r := range m.input {
		if i < len(m.target) && r == m.target[i] {
			correct++
		}
	}
	return correct
}

// wpmStdDev returns the standard deviation of wpmSamples — a measure of
// consistency. Low = steady pace; high = lots of speed swings.
func (m Model) wpmStdDev() float64 {
	if len(m.wpmSamples) < 2 {
		return 0
	}
	mean := 0.0
	for _, v := range m.wpmSamples {
		mean += v
	}
	mean /= float64(len(m.wpmSamples))

	variance := 0.0
	for _, v := range m.wpmSamples {
		d := v - mean
		variance += d * d
	}
	variance /= float64(len(m.wpmSamples))
	return math.Sqrt(variance)
}

// deltaFromLast returns the WPM difference vs the most recent previous
// session with the same mode/lang/duration. Returns 0, false if no prior.
func (m Model) deltaFromLast(dur int) (float64, bool) {
	sessions := recentSessions(50)
	for _, s := range sessions {
		if s.Mode == m.modeKey() && s.Lang == m.langKey() && s.Duration == dur {
			// recentSessions sorts newest-first; the current session was just
			// saved, so skip it and return the one after.
			// We compare by time — skip any session within 5s of now.
			if s.At.After(m.startTime) || m.startTime.Sub(s.At).Seconds() < 5 {
				continue
			}
			return m.finalWPM - s.WPM, true
		}
	}
	return 0, false
}

// topMistakes returns the top n most-missed characters, sorted by count desc.
func (m Model) topMistakes(n int) []mistakeEntry {
	entries := make([]mistakeEntry, 0, len(m.mistakeMap))
	for ch, count := range m.mistakeMap {
		entries = append(entries, mistakeEntry{ch, count})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})
	if len(entries) > n {
		return entries[:n]
	}
	return entries
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
	case stateConfirmQuit:
		return m.viewConfirmQuit()
	case stateGame:
		if m.gameState != nil {
			return m.gameState.render(m.width, m.height)
		}
	case stateGameOver:
		if m.gameState != nil {
			return m.gameState.renderGameOver(m.width, m.height)
		}
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
		for i, t := range timeLimits {
			labels[i] = fmt.Sprintf("%ds", t)
		}
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
		Render("ctrl+t  theme: " + activeTheme.name)

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
		parts = append(parts, hintStyle.Render("ctrl+r restart  ctrl+g menu  ctrl+b blind  ctrl+f focus  ctrl+t theme  esc quit"))
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
	numStyle := lipgloss.NewStyle().Bold(true)

	// ── Primary stat blocks ─────────────────────────────────────────────
	// Keep badges off the column width — ANSI codes confuse lipgloss Width().
	// Render the four numbers in fixed-width columns, then badges on a separate
	// line so nothing gets pushed out of alignment.
	colW := 10
	mkCol := func(val, label string) string {
		return lipgloss.NewStyle().Width(colW).Render(
			lipgloss.JoinVertical(lipgloss.Left, val, subtleStyle.Render(label)),
		)
	}
	wpmCol := mkCol(numStyle.Foreground(activeTheme.wpm).Render(fmt.Sprintf("%.0f", m.finalWPM)), "wpm")
	accCol := mkCol(numStyle.Foreground(activeTheme.acc).Render(fmt.Sprintf("%.1f%%", m.finalAcc)), "acc")
	timeCol := mkCol(numStyle.Foreground(activeTheme.timer).Render(fmt.Sprintf("%.1fs", m.elapsed.Seconds())), "time")
	pbCol := mkCol(numStyle.Foreground(activeTheme.subtext0).Render(fmt.Sprintf("%.0f", pb)), "best")
	statsRow := lipgloss.JoinHorizontal(lipgloss.Bottom, wpmCol, accCol, timeCol, pbCol)

	// Badges row: "new best!" and delta, on their own line, no width constraints
	var badgeParts []string
	if m.isPB {
		badgeParts = append(badgeParts, pbStyle.Render(" new best! "))
	}
	if delta, ok := m.deltaFromLast(dur); ok {
		sign := "+"
		col := activeTheme.green
		if delta < 0 {
			sign = ""
			col = activeTheme.red
		}
		badgeParts = append(badgeParts,
			lipgloss.NewStyle().Foreground(col).Bold(true).
				Render(fmt.Sprintf("%s%.0f from last", sign, delta)))
	}
	badgeRow := ""
	if len(badgeParts) > 0 {
		badgeRow = strings.Join(badgeParts, "  ")
	}

	// ── Divider ───────────────────────────────────────────────────────────
	// Use terminal width minus margins for the chart; min 20, max 80
	divW := m.width - 20
	if divW < 20 {
		divW = 20
	}
	if divW > 80 {
		divW = 80
	}
	div := lipgloss.NewStyle().Foreground(activeTheme.surface0).Render(strings.Repeat("─", divW))

	// ── Secondary metrics row ─────────────────────────────────────────────
	correct := m.correctChars()
	incorrect := m.errors
	total := m.totalKeys
	stddev := m.wpmStdDev()

	labelS := lipgloss.NewStyle().Foreground(activeTheme.surface2)
	valS := lipgloss.NewStyle().Foreground(activeTheme.subtext1).Bold(true)

	rawWPM := m.calcRawWPM()

	// char counts
	charBlock := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top,
			valS.Foreground(activeTheme.correct).Render(fmt.Sprintf("%d", correct)),
			labelS.Render("/"),
			valS.Foreground(activeTheme.wrong).Render(fmt.Sprintf("%d", incorrect)),
			labelS.Render("/"),
			valS.Render(fmt.Sprintf("%d", total)),
		),
		labelS.Render("correct / errors / total"),
	)

	// raw vs net WPM block
	rawBlock := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top,
			valS.Foreground(activeTheme.wpm).Render(fmt.Sprintf("%.0f", m.finalWPM)),
			labelS.Render("  net  "),
			valS.Foreground(activeTheme.subtext0).Render(fmt.Sprintf("%.0f", rawWPM)),
			labelS.Render("  raw"),
		),
		labelS.Render("net wpm / raw wpm"),
	)

	// consistency
	var consLabel string
	switch {
	case stddev < 5:
		consLabel = lipgloss.NewStyle().Foreground(activeTheme.green).Render("very consistent")
	case stddev < 12:
		consLabel = lipgloss.NewStyle().Foreground(activeTheme.yellow).Render("consistent")
	case stddev < 20:
		consLabel = lipgloss.NewStyle().Foreground(activeTheme.peach).Render("variable")
	default:
		consLabel = lipgloss.NewStyle().Foreground(activeTheme.red).Render("inconsistent")
	}
	consBlock := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top,
			valS.Render(fmt.Sprintf("±%.1f", stddev)),
			labelS.Render(" wpm  "),
			consLabel,
		),
		labelS.Render("consistency (std dev)"),
	)

	// error rate
	errRate := 0.0
	if total > 0 {
		errRate = float64(incorrect) / float64(total) * 100
	}
	errBlock := lipgloss.JoinVertical(lipgloss.Left,
		valS.Foreground(activeTheme.wrong).Render(fmt.Sprintf("%.1f%%", errRate)),
		labelS.Render("error rate"),
	)

	spacer := strings.Repeat(" ", 4)
	metricsRow := lipgloss.JoinHorizontal(lipgloss.Top,
		rawBlock, spacer, charBlock, spacer, consBlock, spacer, errBlock,
	)

	// ── Top mistakes detail ───────────────────────────────────────────────
	var mistakesLine string
	if top := m.topMistakes(8); len(top) > 0 {
		var sb strings.Builder
		sb.WriteString(labelS.Render("missed:  "))
		for _, e := range top {
			label := string(e.ch)
			switch e.ch {
			case ' ':
				label = "spc"
			case '\t':
				label = "tab"
			case '\n':
				label = "ret"
			}
			intensity := float64(e.count) / float64(top[0].count)
			var col lipgloss.Color
			switch {
			case intensity >= 0.75:
				col = activeTheme.red
			case intensity >= 0.4:
				col = activeTheme.peach
			default:
				col = activeTheme.surface2
			}
			sb.WriteString(lipgloss.NewStyle().
				Foreground(activeTheme.base).
				Background(col).
				Padding(0, 1).
				Render(label))
			sb.WriteString(lipgloss.NewStyle().Foreground(activeTheme.surface2).
				Render(fmt.Sprintf("×%d ", e.count)))
		}
		mistakesLine = sb.String()
	}

	// ── Bar chart ─────────────────────────────────────────────────────────
	chartRows := m.renderBarChart(divW)

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
		exportLine = "\n" + m.exportMsg
	}

	parts := []string{"", statsRow}
	if badgeRow != "" {
		parts = append(parts, badgeRow)
	}
	parts = append(parts, "", div, "", metricsRow)
	if mistakesLine != "" {
		parts = append(parts, "", mistakesLine)
	}
	parts = append(parts, "", div, "")
	parts = append(parts, chartRows...)
	kbRows := m.renderKeyboard()
	if len(kbRows) > 0 {
		parts = append(parts, "", labelS.Render("keyboard"))
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
	emptyBar := lipgloss.NewStyle().Foreground(activeTheme.surface1).Render(strings.Repeat("━", width-filled))
	pctLabel := lipgloss.NewStyle().Foreground(activeTheme.subtext0).Render(fmt.Sprintf(" %d%%", pct))

	return filledBar + emptyBar + pctLabel
}

// renderBarChart renders a WPM chart using a 5-row grid.
// Each column is one sample. Scaling uses the actual min→max range so
// variation is always visible even for consistent typists.
func (m Model) renderBarChart(maxCols int) []string {
	const chartH = 5      // rows tall
	const minRange = 15.0 // minimum WPM range shown — prevents flat-line look
	if maxCols < 20 {
		maxCols = 20
	}

	allSamples := m.wpmSamples
	if len(allSamples) == 0 {
		return []string{hintStyle.Render("no data yet")}
	}

	// Drop first 2 noisy samples only when there are enough to spare
	samples := allSamples
	if len(samples) > 4 {
		samples = samples[2:]
	}

	// Compute stats
	minV, maxV := samples[0], samples[0]
	peakIdx := 0
	sum := 0.0
	for i, v := range samples {
		if v > maxV {
			maxV = v
			peakIdx = i
		}
		if v < minV {
			minV = v
		}
		sum += v
	}
	avg := sum / float64(len(samples))

	// Enforce a minimum visible range centred on the average so the
	// chart always shows meaningful height variation.
	if maxV-minV < minRange {
		half := minRange / 2
		minV = avg - half
		maxV = avg + half
		if minV < 0 {
			minV = 0
		}
	}
	rangeV := maxV - minV
	if rangeV == 0 {
		rangeV = 1
	}

	// Limit columns so we don't overflow the terminal
	cols := samples
	if len(cols) > maxCols {
		// downsample: pick evenly spaced samples
		down := make([]float64, maxCols)
		for i := range down {
			idx := int(float64(i) / float64(maxCols) * float64(len(samples)))
			if idx >= len(samples) {
				idx = len(samples) - 1
			}
			down[i] = samples[idx]
		}
		cols = down
		peakIdx = 0
		for i, v := range cols {
			if v > cols[peakIdx] {
				peakIdx = i
			}
		}
	}

	// Map each column value to a height 1..chartH
	heights := make([]int, len(cols))
	for i, v := range cols {
		h := int((v-minV)/rangeV*float64(chartH-1)) + 1
		if h < 1 {
			h = 1
		}
		if h > chartH {
			h = chartH
		}
		heights[i] = h
	}

	// Y-axis label width
	topLabel := fmt.Sprintf("%3.0f", maxV)
	midLabel := fmt.Sprintf("%3.0f", (minV+maxV)/2)
	botLabel := fmt.Sprintf("%3.0f", minV)
	yW := len(topLabel) // always 3

	// Build grid rows top→bottom
	emptyStyle := lipgloss.NewStyle().Foreground(activeTheme.surface1)
	fillStyle := sparkBarStyle
	peakStyle := sparkPeakStyle
	topStyle := lipgloss.NewStyle().Foreground(activeTheme.wpm).Bold(true)
	yStyle := hintStyle

	rows := make([]string, chartH)
	for row := 0; row < chartH; row++ {
		level := chartH - row // 1=bottom, chartH=top
		var sb strings.Builder

		// Y-axis label (only on rows 0, mid, bottom)
		switch row {
		case 0:
			sb.WriteString(yStyle.Render(topLabel + " "))
		case chartH / 2:
			sb.WriteString(yStyle.Render(midLabel + " "))
		case chartH - 1:
			sb.WriteString(yStyle.Render(botLabel + " "))
		default:
			sb.WriteString(strings.Repeat(" ", yW+1))
		}

		for ci, h := range heights {
			switch {
			case level > h:
				sb.WriteString(emptyStyle.Render("·"))
			case level == h:
				if ci == peakIdx {
					sb.WriteString(peakStyle.Render("█"))
				} else {
					sb.WriteString(topStyle.Render("▆"))
				}
			default:
				if ci == peakIdx {
					sb.WriteString(peakStyle.Render("█"))
				} else {
					sb.WriteString(fillStyle.Render("█"))
				}
			}
		}
		rows[row] = sb.String()
	}

	// X-axis
	xAxis := strings.Repeat(" ", yW+1) +
		lipgloss.NewStyle().Foreground(activeTheme.surface1).
			Render(strings.Repeat("─", len(cols)))

	// Duration and stats footer
	dur := len(allSamples)
	footer := lipgloss.JoinHorizontal(lipgloss.Top,
		hintStyle.Render(fmt.Sprintf("%ds  ", dur)),
		subtleStyle.Render(fmt.Sprintf("%.0f avg  ", avg)),
		sparkPeakStyle.Render(fmt.Sprintf("%.0f peak wpm", maxV)),
	)

	label := lipgloss.NewStyle().Foreground(activeTheme.subtext0).Render("wpm over time")
	result := []string{label}
	result = append(result, rows...)
	result = append(result, xAxis, footer)
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
		if n > maxCount {
			maxCount = n
		}
	}

	countFor := func(r rune) int { return m.mistakeMap[r] }

	renderKey := func(label string, count int) string {
		return keyHeatStyle(count, maxCount).Render(label)
	}

	kbRows := [][]struct {
		label string
		r     rune
	}{
		{{"q", 'q'}, {" w", 'w'}, {" e", 'e'}, {" r", 'r'}, {" t", 't'}, {" y", 'y'}, {" u", 'u'}, {" i", 'i'}, {" o", 'o'}, {" p", 'p'}},
		{{" a", 'a'}, {" s", 's'}, {" d", 'd'}, {" f", 'f'}, {" g", 'g'}, {" h", 'h'}, {" j", 'j'}, {" k", 'k'}, {" l", 'l'}},
		{{"  z", 'z'}, {" x", 'x'}, {" c", 'c'}, {" v", 'v'}, {" b", 'b'}, {" n", 'n'}, {" m", 'm'}},
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

// ── Confirm quit view ─────────────────────────────────────────────────────────

func (m Model) viewConfirmQuit() string {
	// Dim overlay: show whatever the underlying screen was
	var underlying string
	switch m.prevState {
	case stateMenu:
		underlying = m.viewMenu()
	case stateTyping:
		underlying = m.viewTyping()
	case stateResults:
		underlying = m.viewResults()
	default:
		underlying = ""
	}
	_ = underlying // used for context; dialog renders on top centered

	prompt := lipgloss.NewStyle().
		Foreground(activeTheme.text).Bold(true).
		Render("quit typist?")

	sub := lipgloss.NewStyle().
		Foreground(activeTheme.subtext0).
		Render("your session will be lost")

	yBtn := lipgloss.NewStyle().
		Foreground(activeTheme.base).
		Background(activeTheme.red).
		Bold(true).Padding(0, 3).
		Render("y  quit")

	nBtn := lipgloss.NewStyle().
		Foreground(activeTheme.base).
		Background(activeTheme.green).
		Bold(true).Padding(0, 3).
		Render("n  stay")

	btnRow := lipgloss.JoinHorizontal(lipgloss.Top, yBtn, "   ", nBtn)

	hint := hintStyle.Render("esc / enter  →  stay")

	dialog := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(activeTheme.surface1).
		Background(activeTheme.mantle).
		Padding(2, 4).
		Render(lipgloss.JoinVertical(lipgloss.Center,
			prompt, "", sub, "", btnRow, "", hint,
		))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
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
		if secs > 3600 {
			secs = 3600
		}
		mins := secs / 60
		rem := secs % 60
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
		col(dimLabel, 8, "wpm"),
		col(dimLabel, 8, "acc%"),
		col(dimLabel, 12, "mode"),
		col(dimLabel, 8, "lang"),
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
		wpmS := lipgloss.NewStyle().Foreground(activeTheme.wpm).Bold(true)
		accS := lipgloss.NewStyle().Foreground(activeTheme.acc)
		modeS := lipgloss.NewStyle().Foreground(activeTheme.text)
		langS := lipgloss.NewStyle().Foreground(activeTheme.timer)
		dateS := lipgloss.NewStyle().Foreground(activeTheme.overlay0)
		row := lipgloss.JoinHorizontal(lipgloss.Top,
			col(wpmS, 8, fmt.Sprintf("%.0f", e.WPM)),
			col(accS, 8, fmt.Sprintf("%.1f%%", e.Accuracy)),
			col(modeS, 12, modeLabel),
			col(langS, 8, lang),
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
