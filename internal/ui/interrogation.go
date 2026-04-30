package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jerkeyray/hearsay/internal/kase"
)

type pane int

const (
	paneTopics pane = iota
	paneTechniques
)

type exchange struct {
	topic     string
	technique string
	witness   string
}

type interrogationModel struct {
	kase       kase.Case
	focus      pane
	topicIdx   int
	techIdx    int
	exchanges  []exchange
}

func newInterrogation(c kase.Case) interrogationModel {
	return interrogationModel{
		kase:  c,
		focus: paneTopics,
	}
}

func (m interrogationModel) Init() tea.Cmd { return nil }

// Update returns (next, cmd, back). back=true asks the parent to leave
// the interrogation (return to splash for now).
func (m interrogationModel) Update(msg tea.Msg) (interrogationModel, tea.Cmd, bool) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil, false
	}
	switch k.String() {
	case "q", "ctrl+c":
		return m, tea.Quit, false
	case "esc":
		return m, nil, true
	case "tab":
		if m.focus == paneTopics {
			m.focus = paneTechniques
		} else {
			m.focus = paneTopics
		}
	case "up", "k":
		switch m.focus {
		case paneTopics:
			if m.topicIdx > 0 {
				m.topicIdx--
			}
		case paneTechniques:
			if m.techIdx > 0 {
				m.techIdx--
			}
		}
	case "down", "j":
		switch m.focus {
		case paneTopics:
			if m.topicIdx < len(m.kase.Topics)-1 {
				m.topicIdx++
			}
		case paneTechniques:
			if m.techIdx < len(kase.AllTechniques)-1 {
				m.techIdx++
			}
		}
	case "enter":
		if len(m.kase.Topics) == 0 {
			return m, nil, false
		}
		topic := m.kase.Topics[m.topicIdx].Name
		tech := kase.AllTechniques[m.techIdx].Label()
		m.exchanges = append(m.exchanges, exchange{
			topic:     topic,
			technique: tech,
			witness:   "she pauses.",
		})
	}
	return m, nil, false
}

func (m interrogationModel) View() string {
	header := lipgloss.JoinHorizontal(
		lipgloss.Top,
		styleTitle.Render("hearsay"),
		styleDim.Render("  ·  "+m.kase.ID),
	)

	dialogue := m.renderDialogue()
	topics := m.renderTopics()
	techs := m.renderTechniques()

	bottom := lipgloss.JoinHorizontal(lipgloss.Top, topics, techs)
	footer := styleDim.Render("enter ask · ↹ switch panel · esc back · q quit")

	body := lipgloss.JoinVertical(lipgloss.Left, header, "", dialogue, "", bottom, "", footer)
	return styleBorder.Render(body)
}

func (m interrogationModel) renderDialogue() string {
	if len(m.exchanges) == 0 {
		return styleMuted.Render("(she's waiting.)")
	}
	var b strings.Builder
	for i, ex := range m.exchanges {
		if i > 0 {
			b.WriteString("\n\n")
		}
		fmt.Fprintf(&b, "%s\n", styleDim.Render(fmt.Sprintf("> you (%s, %s)", ex.topic, ex.technique)))
		b.WriteString(ex.witness)
	}
	return b.String()
}

func (m interrogationModel) renderTopics() string {
	header := "ASK ABOUT"
	if m.focus == paneTopics {
		header = styleSelected.Render("ASK ABOUT")
	} else {
		header = styleDim.Render("ASK ABOUT")
	}
	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n")
	for i, t := range m.kase.Topics {
		line := "  " + t.Name
		if i == m.topicIdx {
			if m.focus == paneTopics {
				line = styleSelected.Render("▸ " + t.Name)
			} else {
				line = styleDim.Render("▸ " + t.Name)
			}
		} else {
			line = styleMuted.Render("  " + t.Name)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return lipgloss.NewStyle().Width(28).Render(b.String())
}

func (m interrogationModel) renderTechniques() string {
	var b strings.Builder
	if m.focus == paneTechniques {
		b.WriteString(styleSelected.Render("HOW"))
	} else {
		b.WriteString(styleDim.Render("HOW"))
	}
	b.WriteString("\n")
	for i, tech := range kase.AllTechniques {
		label := tech.Label()
		if i == m.techIdx {
			if m.focus == paneTechniques {
				b.WriteString(styleSelected.Render("▸ " + label))
			} else {
				b.WriteString(styleDim.Render("▸ " + label))
			}
		} else {
			b.WriteString(styleMuted.Render("  " + label))
		}
		b.WriteString("\n")
	}
	return lipgloss.NewStyle().Width(28).Render(b.String())
}
