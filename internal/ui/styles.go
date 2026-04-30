package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorAccent = lipgloss.AdaptiveColor{Light: "#b08040", Dark: "#d4a061"}
	colorDim    = lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}
	colorMuted  = lipgloss.AdaptiveColor{Light: "#999999", Dark: "#555555"}
	colorWarn   = lipgloss.AdaptiveColor{Light: "#a04040", Dark: "#c46060"}

	styleTitle    = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	styleDim      = lipgloss.NewStyle().Foreground(colorDim)
	styleMuted    = lipgloss.NewStyle().Foreground(colorMuted)
	styleSelected = lipgloss.NewStyle().Foreground(colorAccent)
	styleBorder   = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorMuted).
			Padding(1, 2)
)
