package main

// profile.go
// Profile Dashboard — Ctrl+P from any screen.
// Shows: WPM trend sparkline, personal bests table, daily activity heatmap.

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Profile data

type ProfileData struct {
	entries       []ScoreEntry // all sessions, newest first
	totalSessions int
	totalTime     float64 // minutes estimated
	avgWPM        float64
	avgAcc        float64
	bestEver      float64
	streak        int        // current daily streak
	trend         []float64  // last 30 session WPMs for sparkline
	bests         []pbRow    // personal bests table
	activity      [7][24]int // [weekday][hour] session count for heatmap
	weekActivity  [7]int     // sessions per day of week (Sun=0)
	dailyMap      []dayCell  // last 70 days for github-style activity map
}

type pbRow struct {
	mode  string
	lang  string
	dur   int
	wpm   float64
	acc   float64
	count int
	at    time.Time
}

type dayCell struct {
	date  time.Time
	count int
}

func loadProfileData() ProfileData {
	entries := recentSessions(500)
	if len(entries) == 0 {
		return ProfileData{}
	}

	pd := ProfileData{entries: entries, totalSessions: len(entries)}

	// Avg WPM / acc
	wpmSum, accSum := 0.0, 0.0
	for _, e := range entries {
		wpmSum += e.WPM
		accSum += e.Accuracy
		if e.WPM > pd.bestEver {
			pd.bestEver = e.WPM
		}
	}
	pd.avgWPM = wpmSum / float64(len(entries))
	pd.avgAcc = accSum / float64(len(entries))

	// Estimated total time (sum of durations, fall back to 1 min per session)
	for _, e := range entries {
		if e.Duration > 0 {
			pd.totalTime += float64(e.Duration) / 60.0
		} else {
			pd.totalTime += 1.0
		}
	}

	// Trend: last 30 sessions WPMs (entries are newest-first, reverse for chart)
	n := 30
	if len(entries) < n {
		n = len(entries)
	}
	pd.trend = make([]float64, n)
	for i := 0; i < n; i++ {
		pd.trend[n-1-i] = entries[i].WPM
	}

	// Personal bests table — group by (mode, lang, duration)
	type key struct {
		mode, lang string
		dur        int
	}
	type agg struct {
		best  float64
		acc   float64
		count int
		at    time.Time
	}
	aggMap := make(map[key]*agg)
	for _, e := range entries {
		k := key{e.Mode, e.Lang, e.Duration}
		a := aggMap[k]
		if a == nil {
			a = &agg{}
			aggMap[k] = a
		}
		a.count++
		if e.WPM > a.best {
			a.best = e.WPM
			a.acc = e.Accuracy
			a.at = e.At
		}
	}
	for k, a := range aggMap {
		pd.bests = append(pd.bests, pbRow{
			mode: k.mode, lang: k.lang, dur: k.dur,
			wpm: a.best, acc: a.acc, count: a.count, at: a.at,
		})
	}
	sort.Slice(pd.bests, func(i, j int) bool {
		return pd.bests[i].wpm > pd.bests[j].wpm
	})
	if len(pd.bests) > 8 {
		pd.bests = pd.bests[:8]
	}

	// Activity heatmap — last 70 days
	now := time.Now()
	dayMap := make(map[string]int)
	for _, e := range entries {
		dayKey := e.At.Format("2006-01-02")
		dayMap[dayKey]++
	}
	pd.dailyMap = make([]dayCell, 70)
	for i := 0; i < 70; i++ {
		d := now.AddDate(0, 0, -(69 - i))
		pd.dailyMap[i] = dayCell{
			date:  d,
			count: dayMap[d.Format("2006-01-02")],
		}
	}

	// Weekly pattern (sessions per day of week)
	for _, e := range entries {
		wd := int(e.At.Weekday())
		pd.weekActivity[wd]++
	}

	// Daily streak
	pd.streak = calcStreak(entries)

	return pd
}

func calcStreak(entries []ScoreEntry) int {
	if len(entries) == 0 {
		return 0
	}
	seen := make(map[string]bool)
	for _, e := range entries {
		seen[e.At.Format("2006-01-02")] = true
	}
	streak := 0
	now := time.Now()
	// Start from today; if today has no sessions, check yesterday
	d := now
	if !seen[d.Format("2006-01-02")] {
		d = d.AddDate(0, 0, -1)
	}
	for seen[d.Format("2006-01-02")] {
		streak++
		d = d.AddDate(0, 0, -1)
	}
	return streak
}

// Rendering
func viewProfile(pd ProfileData, width, height int) string {
	if pd.totalSessions == 0 {
		empty := lipgloss.JoinVertical(lipgloss.Center,
			lipgloss.NewStyle().Foreground(activeTheme.mauve).Bold(true).Render("profile"),
			"",
			lipgloss.NewStyle().Foreground(activeTheme.subtext0).Render("no sessions recorded yet"),
			lipgloss.NewStyle().Foreground(activeTheme.subtext0).Render("complete a test to see your stats here"),
			"",
			hintStyle.Render("ctrl+o  close"),
		)
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, empty)
	}

	dim := lipgloss.NewStyle().Foreground(activeTheme.subtext0)
	head := lipgloss.NewStyle().Foreground(activeTheme.surface2)
	num := lipgloss.NewStyle().Bold(true)

	// header stat row
	mkStat := func(val, label string, col lipgloss.Color) string {
		return lipgloss.NewStyle().Width(14).Render(
			lipgloss.JoinVertical(lipgloss.Left,
				num.Foreground(col).Render(val),
				dim.Render(label),
			),
		)
	}

	var streakVal string
	if pd.streak > 0 {
		streakVal = fmt.Sprintf("%d day", pd.streak)
		if pd.streak != 1 {
			streakVal += "s"
		}
	} else {
		streakVal = "—"
	}
	var timeVal string
	if pd.totalTime >= 60 {
		timeVal = fmt.Sprintf("%.0fh", pd.totalTime/60)
	} else {
		timeVal = fmt.Sprintf("%.0fm", pd.totalTime)
	}

	headerRow := lipgloss.JoinHorizontal(lipgloss.Bottom,
		mkStat(fmt.Sprintf("%.0f", pd.avgWPM), "avg wpm", activeTheme.wpm),
		mkStat(fmt.Sprintf("%.1f%%", pd.avgAcc), "avg acc", activeTheme.acc),
		mkStat(fmt.Sprintf("%.0f", pd.bestEver), "best wpm", activeTheme.yellow),
		mkStat(fmt.Sprintf("%d", pd.totalSessions), "sessions", activeTheme.mauve),
		mkStat(timeVal, "time spent", activeTheme.timer),
		mkStat(streakVal, "streak", activeTheme.green),
	)

	divW := 14 * 6
	div := lipgloss.NewStyle().Foreground(activeTheme.surface0).Render(strings.Repeat("─", divW))

	// WPM trend chart
	trendStr := renderTrendChart(pd.trend)
	trendLabel := dim.Render(fmt.Sprintf("last %d sessions", len(pd.trend)))

	// Personal bests table
	pbTitle := lipgloss.NewStyle().Foreground(activeTheme.subtext0).Render("personal bests")

	colMode := func(s string, w int) string {
		return lipgloss.NewStyle().Width(w).Render(s)
	}

	pbHeader := lipgloss.JoinHorizontal(lipgloss.Top,
		colMode(head.Render("mode"), 10),
		colMode(head.Render("lang"), 8),
		colMode(head.Render("dur"), 6),
		colMode(head.Render("wpm"), 8),
		colMode(head.Render("acc"), 8),
		colMode(head.Render("tests"), 7),
		head.Render("date"),
	)
	pbDiv := lipgloss.NewStyle().Foreground(activeTheme.surface0).Render(strings.Repeat("─", 55))

	var pbLines []string
	pbLines = append(pbLines, pbHeader, pbDiv)
	for i, r := range pd.bests {
		modeS := lipgloss.NewStyle().Foreground(activeTheme.mauve)
		if i == 0 {
			modeS = modeS.Bold(true)
		}
		label := r.mode
		if r.lang != "" {
			label += "/" + r.lang
		}
		durS := "—"
		if r.dur > 0 {
			durS = fmt.Sprintf("%ds", r.dur)
		}
		wpmColor := activeTheme.wpm
		if i == 0 {
			wpmColor = activeTheme.yellow
		}

		row := lipgloss.JoinHorizontal(lipgloss.Top,
			colMode(modeS.Render(label), 10),
			colMode(dim.Render(r.lang), 8),
			colMode(dim.Render(durS), 6),
			colMode(num.Foreground(wpmColor).Render(fmt.Sprintf("%.0f", r.wpm)), 8),
			colMode(num.Foreground(activeTheme.acc).Render(fmt.Sprintf("%.1f%%", r.acc)), 8),
			colMode(dim.Render(fmt.Sprintf("%d", r.count)), 7),
			dim.Render(r.at.Format("Jan 02")),
		)
		pbLines = append(pbLines, row)
	}

	// Activity heatmap
	actTitle := dim.Render("daily activity  (last 70 days)")

	// Find max for scaling
	maxAct := 0
	for _, dc := range pd.dailyMap {
		if dc.count > maxAct {
			maxAct = dc.count
		}
	}

	// Build 10 columns of 7 rows (10 weeks × 7 days)
	// Start on Sunday so columns are aligned by week
	actRows := renderActivityMap(pd.dailyMap, maxAct)
	dayLabels := dim.Render("S M T W T F S")

	// Weekly pattern bar
	weekTitle := dim.Render("activity by day of week")
	weekBar := renderWeekBar(pd.weekActivity)

	// Assemble
	title := lipgloss.NewStyle().Foreground(activeTheme.mauve).Bold(true).Render("profile")
	hint := hintStyle.Render("ctrl+o/ctrl+p  close   ctrl+t  theme")

	left := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		headerRow,
		"",
		div,
		"",
		trendLabel,
		trendStr,
		"",
		pbTitle,
		lipgloss.JoinVertical(lipgloss.Left, pbLines...),
	)

	right := lipgloss.JoinVertical(lipgloss.Left,
		actTitle,
		"",
		actRows,
		dayLabels,
		"",
		weekTitle,
		"",
		weekBar,
	)

	// Side by side if wide enough, stacked otherwise
	var body string
	if width >= 90 {
		rightPadded := lipgloss.NewStyle().MarginLeft(4).Render(right)
		body = lipgloss.JoinHorizontal(lipgloss.Top, left, rightPadded)
	} else {
		body = lipgloss.JoinVertical(lipgloss.Left, left, "", right)
	}

	full := lipgloss.JoinVertical(lipgloss.Left, body, "", hint)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, full)
}

// Trend chart

// renderTrendChart is a 5-row grid chart like the results screen.
// It shows the WPM trend over recent sessions with Y-axis labels.
func renderTrendChart(samples []float64) string {
	const chartH = 5
	const minRange = 15.0
	maxCols := 30
	if maxCols < 20 {
		maxCols = 20
	}

	if len(samples) == 0 {
		return hintStyle.Render("no data")
	}

	// Drop first 2 noisy samples only when there are enough to spare
	filtered := samples
	if len(filtered) > 4 {
		filtered = filtered[2:]
	}

	// Compute stats
	minV, maxV := filtered[0], filtered[0]
	peakIdx := 0
	sum := 0.0
	for i, v := range filtered {
		if v > maxV {
			maxV = v
			peakIdx = i
		}
		if v < minV {
			minV = v
		}
		sum += v
	}
	avg := sum / float64(len(filtered))

	// Enforce min visible range
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

	// Limit columns (downsample if needed)
	cols := filtered
	if len(cols) > maxCols {
		down := make([]float64, maxCols)
		for i := range down {
			idx := int(float64(i) / float64(maxCols) * float64(len(filtered)))
			if idx >= len(filtered) {
				idx = len(filtered) - 1
			}
			down[i] = filtered[idx]
		}
		cols = down
		peakIdx = 0
		for i, v := range cols {
			if v > cols[peakIdx] {
				peakIdx = i
			}
		}
	}

	// Map to heights 1..chartH
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

	// Y-axis labels
	topLabel := fmt.Sprintf("%3.0f", maxV)
	midLabel := fmt.Sprintf("%3.0f", (minV+maxV)/2)
	botLabel := fmt.Sprintf("%3.0f", minV)
	yW := len(topLabel)

	emptyStyle := lipgloss.NewStyle().Foreground(activeTheme.surface1)
	fillStyle := sparkBarStyle
	peakStyle := sparkPeakStyle
	yStyle := hintStyle

	// Build rows top→bottom
	rows := make([]string, chartH)
	for row := 0; row < chartH; row++ {
		level := chartH - row
		var sb strings.Builder

		// Y-axis label
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
					sb.WriteString(sparkBarStyle.Render("▆"))
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

	// Footer stats
	footer := lipgloss.JoinHorizontal(lipgloss.Top,
		hintStyle.Render(fmt.Sprintf("%d sessions  ", len(filtered))),
		sparkBarStyle.Render(fmt.Sprintf("%.0f avg  ", avg)),
		sparkPeakStyle.Render(fmt.Sprintf("%.0f peak", maxV)),
	)

	result := []string{rows[0], rows[1], rows[2], rows[3], rows[4], xAxis, footer}
	return strings.Join(result, "\n")
}

// Activity map

func renderActivityMap(days []dayCell, maxCount int) string {
	if maxCount == 0 {
		maxCount = 1
	}

	// We have 70 cells — lay them out as 10 columns of 7 rows (week columns)
	// Pad to align to week boundary
	startWd := int(days[0].date.Weekday()) // 0=Sun
	padded := make([]dayCell, startWd)     // empty padding at start
	padded = append(padded, days...)
	// Pad end to full weeks
	for len(padded)%7 != 0 {
		padded = append(padded, dayCell{})
	}
	weeks := len(padded) / 7

	// Build week columns (7 rows each)
	cols := make([][]dayCell, weeks)
	for w := 0; w < weeks; w++ {
		cols[w] = padded[w*7 : (w+1)*7]
	}

	// Render row by row
	var rows []string
	for row := 0; row < 7; row++ {
		var sb strings.Builder
		for _, col := range cols {
			cell := col[row]
			sb.WriteString(activityCell(cell.count, maxCount))
			sb.WriteString(" ")
		}
		rows = append(rows, sb.String())
	}

	return strings.Join(rows, "\n")
}

func activityCell(count, maxCount int) string {
	if count == 0 {
		return lipgloss.NewStyle().Foreground(activeTheme.surface0).Render("▪")
	}
	h := float64(count) / float64(maxCount)
	var col lipgloss.Color
	switch {
	case h < 0.25:
		col = activeTheme.surface2
	case h < 0.5:
		col = activeTheme.teal
	case h < 0.75:
		col = activeTheme.mauve
	default:
		col = activeTheme.yellow
	}
	return lipgloss.NewStyle().Foreground(col).Bold(h >= 0.75).Render("▪")
}

// Weekly bar chart

func renderWeekBar(counts [7]int) string {
	days := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	maxC := 0
	for _, c := range counts {
		if c > maxC {
			maxC = c
		}
	}
	if maxC == 0 {
		maxC = 1
	}

	const barH = 4
	const barRunes = "▁▂▃▄▅▆▇█"
	runes := []rune(barRunes)

	var cols []string
	for i, c := range counts {
		idx := int(float64(c) / float64(maxC) * float64(len(runes)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(runes) {
			idx = len(runes) - 1
		}

		bar := string(runes[idx])
		col := activeTheme.mauve
		if c == maxC {
			col = activeTheme.yellow
		}

		barStr := lipgloss.NewStyle().Foreground(col).Bold(c == maxC).Render(bar)
		dayStr := lipgloss.NewStyle().Foreground(activeTheme.surface2).Render(days[i])

		cols = append(cols, lipgloss.JoinVertical(lipgloss.Center,
			barStr, dayStr,
		))
	}
	return lipgloss.JoinHorizontal(lipgloss.Bottom, cols...)
}

// Unused import guard
var _ = math.Sqrt // ensure math is used (used in game.go too, but keep explicit)

//Model integration

func (m Model) updateProfile(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyCtrlP:
		// Toggle: close profile, return to previous state
		m.state = m.prevState
		return m, nil
	case tea.KeyCtrlT:
		m.toggleTheme()
		return m, nil
	case tea.KeyCtrlG:
		next := m.goToMenu()
		return next, nil
	case tea.KeyEsc:
		m.state = m.prevState
		return m, nil
	}
	return m, nil
}
