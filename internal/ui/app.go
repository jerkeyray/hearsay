package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jerkeyray/hearsay/cases/streetlight"
)

type screen int

const (
	screenSplash screen = iota
	screenBriefing
	screenPlaceholder
)

type model struct {
	screen      screen
	splash      splashModel
	briefing    briefingModel
	placeholder string
	quitting    bool
}

func New() tea.Model {
	return model{screen: screenSplash, splash: newSplash()}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenSplash:
		next, cmd, choice, picked := m.splash.Update(msg)
		m.splash = next
		if !picked {
			return m, cmd
		}
		switch choice {
		case choiceQuit:
			m.quitting = true
			return m, tea.Quit
		case choiceNew:
			m.briefing = newBriefing(streetlight.Case)
			m.screen = screenBriefing
		case choiceContinue:
			m.screen = screenPlaceholder
			m.placeholder = "continue — save picker lands here later"
		}
		return m, nil

	case screenBriefing:
		next, cmd, advance, back := m.briefing.Update(msg)
		m.briefing = next
		if back {
			m.screen = screenSplash
			return m, nil
		}
		if advance {
			m.screen = screenPlaceholder
			m.placeholder = "interrogation — three-pane layout lands here next step"
			return m, nil
		}
		return m, cmd

	case screenPlaceholder:
		if k, ok := msg.(tea.KeyMsg); ok {
			switch k.String() {
			case "q", "ctrl+c", "esc":
				m.screen = screenSplash
			}
		}
		return m, nil
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	switch m.screen {
	case screenSplash:
		return m.splash.View()
	case screenBriefing:
		return m.briefing.View()
	case screenPlaceholder:
		return styleBorder.Render(m.placeholder + "\n\n" + styleDim.Render("esc back · q quit"))
	}
	return ""
}
