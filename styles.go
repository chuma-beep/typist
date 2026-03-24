package main

import "github.com/charmbracelet/lipgloss"

// Catppuccin Mocha palette
var (
	correctStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1"))
	incorrectStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8"))
	pendingStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#45475a"))
	cursorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4")).Underline(true)

	titleStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7")).Bold(true).MarginBottom(1)
	wpmStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#f9e2af")).Bold(true)
	accStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#89dceb")).Bold(true)
	timeStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#89b4fa"))
	subtleStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#585b70"))
	hintStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#313244"))
	pbStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#1e1e2e")).Background(lipgloss.Color("#f9e2af")).Bold(true)

	// Menu styles
	selectedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#1e1e2e")).Background(lipgloss.Color("#cba6f7")).Bold(true).Padding(0, 1).MarginRight(1)
	dimSelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#1e1e2e")).Background(lipgloss.Color("#585b70")).Bold(true).Padding(0, 1).MarginRight(1)
	optionStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#585b70")).Padding(0, 1).MarginRight(1)

	cardStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#313244")).
		Padding(1, 4)
)
