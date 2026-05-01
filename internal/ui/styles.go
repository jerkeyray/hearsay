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

// frame wraps content in the standard hearsay border, sized to fill
// width × height (the terminal size, in cells). Falls back to an
// auto-sized border when dimensions are unknown (before the first
// tea.WindowSizeMsg arrives).
func frame(content string, width, height int) string {
	s := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(colorMuted).
		Padding(1, 2)
	// Subtract 4 for the border (left+right) and the padding (left+right=4 chars).
	// The lipgloss padding of (1,2) means 2-char horizontal pad on each side
	// plus a 1-char border, total 6 — but lipgloss applies Width to the *content*
	// area inside padding, so we just subtract the border + padding once.
	if width > 6 {
		s = s.Width(width - 6)
	}
	if height > 4 {
		s = s.Height(height - 4)
	}
	return s.Render(content)
}
