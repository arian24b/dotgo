package tui

import "github.com/charmbracelet/lipgloss"

// bast.sh look: no boxes, purple accent, badge + [n] tabs, grouped list | detail pane.
var (
	purple = lipgloss.Color("#8b5cf6")
	dim    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7089"))
	accent = lipgloss.NewStyle().Foreground(purple).Bold(true)
	badge  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff")).Background(purple).Padding(0, 1)
	selRow = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("#232a3b"))
	green  = lipgloss.NewStyle().Foreground(lipgloss.Color("#4ade80"))
	yellow = lipgloss.NewStyle().Foreground(lipgloss.Color("#facc15"))
	red    = lipgloss.NewStyle().Foreground(lipgloss.Color("#f87171"))
	cyan   = lipgloss.NewStyle().Foreground(lipgloss.Color("#38bdf8"))
)
