package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func renderTodaySidebar(width int, data TodayData, cfg AppConfig, settingGoal bool, goalInput string, height int) string {
	contentW := width - 4 // 2 for border + 2 for padding

	// ── Clock ──
	now := time.Now()
	clock := lipgloss.NewStyle().
		Foreground(activeTheme.subtext0).
		Render(now.Format("15:04"))

	title := lipgloss.NewStyle().
		Foreground(activeTheme.mauve).Bold(true).
		Render("Today")

	titleRow := lipgloss.JoinHorizontal(lipgloss.Top, title, "  ", clock)

	// ── Stats summary ──
	var countLine string
	if data.Count > 0 {
		countLine = fmt.Sprintf("◆  %d tests", data.Count)
	} else {
		countLine = "◆  no tests yet"
	}

	// ── Duration / Goal progress ──
	elapsedMin := data.TotalElapsed / 60.0
	goalMin := float64(cfg.DailyGoalMinutes)

	elapsedStr := formatDuration(data.TotalElapsed)
	goalStr := formatDuration(float64(cfg.DailyGoalMinutes) * 60)

	var goalLine string
	var bar string
	if data.Count > 0 {
		goalLine = fmt.Sprintf("⏱  %s / %s min", elapsedStr, goalStr)
		// Progress bar
		pct := elapsedMin / goalMin
		if pct > 1.0 {
			pct = 1.0
		}
		barW := contentW - 4
		if barW < 5 {
			barW = 5
		}
		filled := int(float64(barW) * pct)
		barF := lipgloss.NewStyle().Foreground(activeTheme.green).Render(strings.Repeat("█", filled))
		barE := lipgloss.NewStyle().Foreground(activeTheme.surface0).Render(strings.Repeat("░", barW-filled))
		pctLabel := lipgloss.NewStyle().Foreground(activeTheme.subtext0).Render(fmt.Sprintf("%3d%%", int(pct*100)))
		bar = barF + barE + " " + pctLabel
	} else {
		goalLine = fmt.Sprintf("⏱  0:00 / %s min", goalStr)
		barW := contentW - 4
		if barW < 5 {
			barW = 5
		}
		barE := lipgloss.NewStyle().Foreground(activeTheme.surface0).Render(strings.Repeat("░", barW))
		bar = barE + "  0%"
	}

	// ── Avg / Best stats ──
	numStyle := lipgloss.NewStyle().Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(activeTheme.subtext0)

	var avgLine, bestLine string
	if data.Count > 0 {
		avgLine = lipgloss.JoinHorizontal(lipgloss.Top,
			dimStyle.Render("avg "),
			numStyle.Foreground(activeTheme.wpm).Render(fmt.Sprintf("%.0f", data.AvgWPM)),
			dimStyle.Render(" wpm · "),
			numStyle.Foreground(activeTheme.acc).Render(fmt.Sprintf("%.1f%%", data.AvgAcc)),
		)
		bestLine = lipgloss.JoinHorizontal(lipgloss.Top,
			dimStyle.Render("best "),
			numStyle.Foreground(activeTheme.yellow).Render(fmt.Sprintf("%.0f", data.BestWPM)),
			dimStyle.Render(" wpm · "),
			numStyle.Foreground(activeTheme.acc).Render(fmt.Sprintf("%.1f%%", data.BestAcc)),
		)
	}

	// ── Divider ──
	div := lipgloss.NewStyle().
		Foreground(activeTheme.surface0).
		Render(strings.Repeat("─", contentW))

	// ── Sessions list ──
	var sessionRows []string
	if data.Count > 0 {
		// How many sessions can we fit?
		maxRows := height - 18 // account for header, stats, divider, footer
		if maxRows < 1 {
			maxRows = 1
		}
		if maxRows > len(data.Sessions) {
			maxRows = len(data.Sessions)
		}
		for i := 0; i < maxRows; i++ {
			e := data.Sessions[i]
			row := renderTodayRow(e, contentW)
			sessionRows = append(sessionRows, row)
		}
	} else {
		sessionRows = append(sessionRows, dimStyle.Render("complete a test to"))
		sessionRows = append(sessionRows, dimStyle.Render("see today's activity"))
	}

	// ── Goal input mode ──
	var goalInputSection string
	if settingGoal {
		display := goalInput
		if display == "" {
			display = "0"
		}
		inputBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(activeTheme.mauve).
			Padding(0, 1).
			Width(contentW - 4).
			Render(
				lipgloss.NewStyle().Foreground(activeTheme.wpm).Render(display) +
					lipgloss.NewStyle().Foreground(activeTheme.mauve).Render("█"),
			)
		prompt := dimStyle.Render("daily goal (min):")
		goalInputSection = lipgloss.JoinVertical(lipgloss.Left, prompt, "", inputBox)
	}

	// ── Footer hint ──
	hint := lipgloss.NewStyle().Foreground(activeTheme.overlay0).Render("g goal  ·  ctrl+d close")

	// ── Assemble ──
	var parts []string
	parts = append(parts, titleRow, "")
	parts = append(parts, countLine)
	parts = append(parts, goalLine)
	parts = append(parts, bar)
	if data.Count > 0 {
		parts = append(parts, "", avgLine, bestLine)
	}
	parts = append(parts, "", div, "")
	if settingGoal {
		parts = append(parts, goalInputSection)
	} else {
		parts = append(parts, sessionRows...)
	}
	parts = append(parts, "", hint)

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)

	sidebarStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(activeTheme.surface1).
		Padding(0, 1).
		Width(contentW)

	return sidebarStyle.Render(content)
}

func renderTodayRow(e ScoreEntry, maxW int) string {
	t := e.At.Format("15:04")
	modeLabel := e.Mode
	if e.Duration > 0 {
		modeLabel = fmt.Sprintf("%s/%ds", e.Mode, e.Duration)
	}
	if e.Lang != "" {
		modeLabel += "/" + e.Lang
	}

	timeS := lipgloss.NewStyle().Foreground(activeTheme.surface2).Render(t)
	wpmS := lipgloss.NewStyle().Foreground(activeTheme.wpm).Bold(true).Render(fmt.Sprintf("%.0fw", e.WPM))
	accS := lipgloss.NewStyle().Foreground(activeTheme.acc).Render(fmt.Sprintf("%.0f%%", e.Accuracy))
	modeS := lipgloss.NewStyle().Foreground(activeTheme.subtext0).Render(modeLabel)

	row := lipgloss.JoinHorizontal(lipgloss.Top,
		timeS, "  ", wpmS, " ", accS, " ", modeS,
	)

	// Truncate if too long
	return lipgloss.NewStyle().MaxWidth(maxW).Render(row)
}

func formatDuration(secs float64) string {
	m := int(secs) / 60
	s := int(secs) % 60
	if m > 0 {
		return fmt.Sprintf("%d:%02d", m, s)
	}
	return fmt.Sprintf("0:%02d", s)
}
