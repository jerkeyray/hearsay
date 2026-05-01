package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type splashChoice int

const (
	choiceNew splashChoice = iota
	choiceContinue
	choiceQuit
)

func (c splashChoice) label() string {
	switch c {
	case choiceNew:
		return "new case"
	case choiceContinue:
		return "continue"
	case choiceQuit:
		return "quit"
	}
	return ""
}

type splashModel struct {
	cursor splashChoice
}

func newSplash() splashModel { return splashModel{cursor: choiceNew} }

func (m splashModel) Init() tea.Cmd { return nil }

func (m splashModel) Update(msg tea.Msg) (splashModel, tea.Cmd, splashChoice, bool) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > choiceNew {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < choiceQuit {
				m.cursor++
			}
		case "enter":
			return m, nil, m.cursor, true
		case "q", "ctrl+c":
			return m, tea.Quit, choiceQuit, false
		}
	}
	return m, nil, 0, false
}

func (m splashModel) View() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("hearsay"))
	b.WriteString("\n\n")
	b.WriteString(styleDim.Render("a memory interrogation game"))
	b.WriteString("\n\n")
	for c := choiceNew; c <= choiceQuit; c++ {
		line := "[ " + c.label() + " ]"
		if c == m.cursor {
			b.WriteString(styleSelected.Render("▸ " + line))
		} else {
			b.WriteString(styleMuted.Render("  " + line))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(styleDim.Render("? help · q quit"))
	return styleBorder.Render(b.String())
}
