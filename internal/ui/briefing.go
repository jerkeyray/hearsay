package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jerkeyray/hearsay/internal/kase"
)

type briefingModel struct {
	kase kase.Case
}

func newBriefing(c kase.Case) briefingModel { return briefingModel{kase: c} }

func (m briefingModel) Init() tea.Cmd { return nil }

// Update returns (next, cmd, advance). When advance is true the parent
// should transition to the next screen (interrogation). esc returns to
// splash; q/ctrl+c quits.
func (m briefingModel) Update(msg tea.Msg) (briefingModel, tea.Cmd, bool, bool) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "enter":
			return m, nil, true, false
		case "esc":
			return m, nil, false, true
		case "q", "ctrl+c":
			return m, tea.Quit, false, false
		}
	}
	return m, nil, false, false
}

func (m briefingModel) View() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render(m.kase.Title))
	b.WriteString("\n\n")
	b.WriteString(m.kase.Briefing)
	b.WriteString("\n\n")
	b.WriteString(styleDim.Render("enter begin · esc back · q quit"))
	return b.String()
}
