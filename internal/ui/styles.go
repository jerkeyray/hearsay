package ui

import "github.com/charmbracelet/lipgloss"

// PRD §7.1: monospace, mostly dim. Single warm-amber accent for
// active state; dim red reserved for warnings; spacing breathes.
// All colors are AdaptiveColor so light and dark terminals both
// read correctly.
var (
	colorAccent = lipgloss.AdaptiveColor{Light: "#a06030", Dark: "#d4a061"}
	colorDim    = lipgloss.AdaptiveColor{Light: "#5a5a5a", Dark: "#8a8a8a"}
	colorMuted  = lipgloss.AdaptiveColor{Light: "#909090", Dark: "#5a5a5a"}
	colorWarn   = lipgloss.AdaptiveColor{Light: "#a04040", Dark: "#c46060"}
	_           = colorWarn // reserved for future use (M5 polish, real warnings)

	styleTitle    = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	styleDim      = lipgloss.NewStyle().Foreground(colorDim)
	styleMuted    = lipgloss.NewStyle().Foreground(colorMuted)
	styleSelected = lipgloss.NewStyle().Foreground(colorAccent)
	styleBorder   = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorMuted).
			Padding(1, 2)
)
