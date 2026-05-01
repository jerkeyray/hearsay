package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jerkeyray/hearsay/internal/kase"
)

// casePickerModel lists the available cases. It only renders when
// len(cases) > 1; with a single case the router skips it and goes
// straight to briefing.
type casePickerModel struct {
	cases  []kase.Case
	cursor int
}

func newCasePicker(cs []kase.Case) casePickerModel {
	return casePickerModel{cases: cs}
}

func (m casePickerModel) Init() tea.Cmd { return nil }

// Update returns (next, cmd, picked, back). picked=true uses cases[cursor];
// back=true returns to splash.
func (m casePickerModel) Update(msg tea.Msg) (casePickerModel, tea.Cmd, bool, bool) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "q", "ctrl+c":
			return m, tea.Quit, false, false
		case "esc":
			return m, nil, false, true
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.cases)-1 {
				m.cursor++
			}
		case "enter":
			return m, nil, true, false
		}
	}
	return m, nil, false, false
}

// Selected returns the case at the cursor.
func (m casePickerModel) Selected() kase.Case { return m.cases[m.cursor] }

func (m casePickerModel) View() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("choose a case"))
	b.WriteString("\n\n")
	for i, c := range m.cases {
		row := c.Title
		if row == "" {
			row = c.ID
		}
		if i == m.cursor {
			b.WriteString(styleSelected.Render("▸ " + row))
		} else {
			b.WriteString(styleMuted.Render("  " + row))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(styleDim.Render("↑↓ select · enter open · esc back · q quit"))
	return styleBorder.Render(b.String())
}
