package ui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jerkeyray/hearsay/internal/game"
	"github.com/jerkeyray/hearsay/internal/kase"
)

type pane int

const (
	paneTopics pane = iota
	paneTechniques
)

type interrogationModel struct {
	session  *game.Session
	focus    pane
	topicIdx int
	techIdx  int
	lastErr  string
}

func newInterrogation(s *game.Session) interrogationModel {
	return interrogationModel{session: s, focus: paneTopics}
}

func (m interrogationModel) Init() tea.Cmd { return nil }

// Close releases the session's underlying journal. Safe to call once.
func (m interrogationModel) Close(ctx context.Context) error {
	if m.session == nil {
		return nil
	}
	return m.session.Close(ctx)
}

// Update returns (next, cmd, back). back=true asks the parent to leave
// the interrogation (return to splash for now).
func (m interrogationModel) Update(msg tea.Msg) (interrogationModel, tea.Cmd, bool) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil, false
	}
	topics := m.session.Case.Topics
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
			if m.topicIdx < len(topics)-1 {
				m.topicIdx++
			}
		case paneTechniques:
			if m.techIdx < len(kase.AllTechniques)-1 {
				m.techIdx++
			}
		}
	case "enter":
		if len(topics) == 0 {
			return m, nil, false
		}
		topic := topics[m.topicIdx].Name
		tech := kase.AllTechniques[m.techIdx]
		if _, err := m.session.Ask(context.Background(), topic, tech); err != nil {
			m.lastErr = err.Error()
		} else {
			m.lastErr = ""
		}
	}
	return m, nil, false
}

func (m interrogationModel) View() string {
	header := lipgloss.JoinHorizontal(
		lipgloss.Top,
		styleTitle.Render("hearsay"),
		styleDim.Render("  ·  "+m.session.Case.ID),
	)

	dialogue := m.renderDialogue()
	topics := m.renderTopics()
	techs := m.renderTechniques()

	bottom := lipgloss.JoinHorizontal(lipgloss.Top, topics, techs)
	footer := styleDim.Render("enter ask · ↹ switch panel · esc back · q quit")

	parts := []string{header, "", dialogue, "", bottom, "", footer}
	if m.lastErr != "" {
		parts = append(parts, styleDim.Render("err: "+m.lastErr))
	}
	body := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return styleBorder.Render(body)
}

func (m interrogationModel) renderDialogue() string {
	log := m.session.Log()
	if len(log) == 0 {
		return styleMuted.Render("(she's waiting.)")
	}
	var b strings.Builder
	for i, ex := range log {
		if i > 0 {
			b.WriteString("\n\n")
		}
		fmt.Fprintf(&b, "%s\n", styleDim.Render(fmt.Sprintf("> you (%s, %s)", ex.Topic, ex.Technique.Label())))
		b.WriteString(ex.Witness)
	}
	return b.String()
}

func (m interrogationModel) renderTopics() string {
	var header string
	if m.focus == paneTopics {
		header = styleSelected.Render("ASK ABOUT")
	} else {
		header = styleDim.Render("ASK ABOUT")
	}
	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n")
	for i, t := range m.session.Case.Topics {
		var line string
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
