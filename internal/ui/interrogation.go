package ui

import (
	"context"
	"errors"
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

// pendingAsk is the in-flight ask currently being run by the witness
// driver. Held in UI state (not Session) so the View can render the
// player line + typing indicator without touching the session log.
type pendingAsk struct {
	topic     string
	technique kase.Technique
}

// witnessRespondedMsg is delivered when the async ask completes.
// game.Session.Ask has already appended the exchange to its log; the
// UI just needs to clear the pending state.
type witnessRespondedMsg struct {
	err error
}

type interrogationModel struct {
	session  *game.Session
	focus    pane
	topicIdx int
	techIdx  int
	pending  *pendingAsk
	lastErr  string
}

func newInterrogation(s *game.Session) interrogationModel {
	return interrogationModel{session: s, focus: paneTopics}
}

func (m interrogationModel) Init() tea.Cmd { return nil }

// Close releases the session's underlying driver. Safe to call once.
func (m interrogationModel) Close(ctx context.Context) error {
	if m.session == nil {
		return nil
	}
	return m.session.Close(ctx)
}

// askCmd dispatches Session.Ask in a goroutine so the TUI loop stays
// responsive while the LLM is in flight.
func askCmd(s *game.Session, topic string, technique kase.Technique) tea.Cmd {
	return func() tea.Msg {
		_, err := s.Ask(context.Background(), topic, technique)
		return witnessRespondedMsg{err: err}
	}
}

// Update returns (next, cmd, back). back=true asks the parent to leave
// the interrogation (return to splash for now).
func (m interrogationModel) Update(msg tea.Msg) (interrogationModel, tea.Cmd, bool) {
	// Async response from a previous ask.
	if r, ok := msg.(witnessRespondedMsg); ok {
		m.pending = nil
		if r.err != nil {
			if errors.Is(r.err, game.ErrSessionEnded) {
				m.lastErr = "the witness leaves"
			} else {
				m.lastErr = r.err.Error()
			}
		} else {
			m.lastErr = ""
		}
		return m, nil, false
	}

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
		if m.pending != nil || len(topics) == 0 || m.session.SessionEnded() {
			return m, nil, false
		}
		topic := topics[m.topicIdx].Name
		tech := kase.AllTechniques[m.techIdx]
		m.pending = &pendingAsk{topic: topic, technique: tech}
		m.lastErr = ""
		return m, askCmd(m.session, topic, tech), false
	}
	return m, nil, false
}

func (m interrogationModel) View() string {
	header := lipgloss.JoinHorizontal(
		lipgloss.Top,
		styleTitle.Render("hearsay"),
		styleDim.Render("  ·  "+m.session.Case.ID),
		styleDim.Render("  ·  "+m.session.ClockDisplay()),
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
	if len(log) == 0 && m.pending == nil {
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
	if m.pending != nil {
		if len(log) > 0 {
			b.WriteString("\n\n")
		}
		fmt.Fprintf(&b, "%s\n", styleDim.Render(fmt.Sprintf("> you (%s, %s)", m.pending.topic, m.pending.technique.Label())))
		b.WriteString(styleMuted.Render("—"))
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
