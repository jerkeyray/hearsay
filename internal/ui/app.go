package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	quitting bool
}

func New() tea.Model {
	return model{}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	return "hearsay\n\npress q to quit\n"
}
