package main

import "github.com/charmbracelet/lipgloss"

// Theme holds all the colours for one visual theme.
type Theme struct {
	correct  lipgloss.Color
	wrong    lipgloss.Color
	pending  lipgloss.Color
	cursor   lipgloss.Color
	title    lipgloss.Color
	wpm      lipgloss.Color
	acc      lipgloss.Color
	timer    lipgloss.Color
	subtle   lipgloss.Color
	hint     lipgloss.Color
	pbFg     lipgloss.Color
	pbBg     lipgloss.Color
	menuSel  lipgloss.Color
	menuSelB lipgloss.Color
	menuDim  lipgloss.Color
	border   lipgloss.Color
	// syntax
	hlKw  lipgloss.Color
	hlBi  lipgloss.Color
	hlStr lipgloss.Color
	hlCmt lipgloss.Color
	hlNum lipgloss.Color
	hlPun lipgloss.Color
	// sparkline
	spark     lipgloss.Color
	sparkPeak lipgloss.Color
}

// Catppuccin Mocha (dark)
var mocha = Theme{
	correct: "#a6e3a1", wrong: "#f38ba8", pending: "#45475a",
	cursor: "#cdd6f4", title: "#cba6f7", wpm: "#f9e2af",
	acc: "#89dceb", timer: "#89b4fa", subtle: "#585b70",
	hint: "#313244", pbFg: "#1e1e2e", pbBg: "#f9e2af",
	menuSel: "#1e1e2e", menuSelB: "#cba6f7", menuDim: "#585b70",
	border: "#313244",
	hlKw: "#cba6f7", hlBi: "#89dceb", hlStr: "#a6e3a1",
	hlCmt: "#6c7086", hlNum: "#fab387", hlPun: "#89b4fa",
	spark: "#cba6f7", sparkPeak: "#f9e2af",
}

// Catppuccin Latte (light)
var latte = Theme{
	correct: "#40a02b", wrong: "#d20f39", pending: "#acb0be",
	cursor: "#4c4f69", title: "#8839ef", wpm: "#df8e1d",
	acc: "#04a5e5", timer: "#1e66f5", subtle: "#9ca0b0",
	hint: "#bcc0cc", pbFg: "#eff1f5", pbBg: "#df8e1d",
	menuSel: "#eff1f5", menuSelB: "#8839ef", menuDim: "#9ca0b0",
	border: "#bcc0cc",
	hlKw: "#8839ef", hlBi: "#04a5e5", hlStr: "#40a02b",
	hlCmt: "#9ca0b0", hlNum: "#fe640b", hlPun: "#1e66f5",
	spark: "#8839ef", sparkPeak: "#df8e1d",
}

// activeTheme is swapped at runtime by Ctrl+T.
var activeTheme = mocha

func applyTheme(t Theme) {
	activeTheme = t
	correctStyle    = lipgloss.NewStyle().Foreground(t.correct)
	incorrectStyle  = lipgloss.NewStyle().Foreground(t.wrong)
	pendingStyle    = lipgloss.NewStyle().Foreground(t.pending)
	cursorStyle     = lipgloss.NewStyle().Foreground(t.cursor).Underline(true)
	titleStyle      = lipgloss.NewStyle().Foreground(t.title).Bold(true).MarginBottom(1)
	wpmStyle        = lipgloss.NewStyle().Foreground(t.wpm).Bold(true)
	accStyle        = lipgloss.NewStyle().Foreground(t.acc).Bold(true)
	timeStyle       = lipgloss.NewStyle().Foreground(t.timer)
	subtleStyle     = lipgloss.NewStyle().Foreground(t.subtle)
	hintStyle       = lipgloss.NewStyle().Foreground(t.hint)
	pbStyle         = lipgloss.NewStyle().Foreground(t.pbFg).Background(t.pbBg).Bold(true)
	errorStyle      = lipgloss.NewStyle().Foreground(t.wrong)
	selectedStyle   = lipgloss.NewStyle().Foreground(t.menuSel).Background(t.menuSelB).Bold(true).Padding(0, 1).MarginRight(1)
	dimSelectedStyle = lipgloss.NewStyle().Foreground(t.menuSel).Background(t.menuDim).Bold(true).Padding(0, 1).MarginRight(1)
	optionStyle     = lipgloss.NewStyle().Foreground(t.subtle).Padding(0, 1).MarginRight(1)
	hlKeyword       = lipgloss.NewStyle().Foreground(t.hlKw)
	hlBuiltin       = lipgloss.NewStyle().Foreground(t.hlBi)
	hlString        = lipgloss.NewStyle().Foreground(t.hlStr)
	hlComment       = lipgloss.NewStyle().Foreground(t.hlCmt)
	hlNumber        = lipgloss.NewStyle().Foreground(t.hlNum)
	hlPunct         = lipgloss.NewStyle().Foreground(t.hlPun)
	sparkBarStyle   = lipgloss.NewStyle().Foreground(t.spark)
	sparkPeakStyle  = lipgloss.NewStyle().Foreground(t.sparkPeak)
	cardStyle       = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(t.border).Padding(1, 4)
}

func init() { applyTheme(mocha) }

func isDark() bool { return activeTheme == mocha }

// All style vars — populated by applyTheme, used everywhere else.
var (
	correctStyle, incorrectStyle, pendingStyle, cursorStyle lipgloss.Style
	titleStyle, wpmStyle, accStyle, timeStyle               lipgloss.Style
	subtleStyle, hintStyle, pbStyle, errorStyle             lipgloss.Style
	selectedStyle, dimSelectedStyle, optionStyle            lipgloss.Style
	hlKeyword, hlBuiltin, hlString, hlComment, hlNumber, hlPunct lipgloss.Style
	sparkBarStyle, sparkPeakStyle                           lipgloss.Style
	cardStyle                                               lipgloss.Style
)
