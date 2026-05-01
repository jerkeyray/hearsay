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

// branchedMsg is delivered when Session.Branch finishes. The router
// (app.go) handles it by swapping the active session.
type branchedMsg struct {
	session *game.Session
	err     error
}

type interrogationModel struct {
	session  *game.Session
	focus    pane
	topicIdx int
	techIdx  int
	pending  *pendingAsk
	lastErr  string

	// rewindOpen is true while the rewind/branch picker overlays
	// the interrogation pane; rewindIdx is the cursor inside it.
	// rewindMode: "rewind" or "branch"; affects the action taken
	// on enter.
	rewindOpen bool
	rewindIdx  int
	rewindMode string

	// inspectorOpen overlays the inspector panel. inspector holds
	// its model (loaded lazily on each open).
	inspectorOpen bool
	inspector     inspectorModel
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

// updateRewind handles keys while the rewind/branch picker is open.
// ↑↓ navigate; enter rewinds or branches to the highlighted turn;
// esc cancels. "Before any asks" is selectable as the last item.
func (m interrogationModel) updateRewind(k tea.KeyMsg) (interrogationModel, tea.Cmd, bool, bool) {
	turns := m.session.TurnCount()
	maxIdx := turns // 0..turns-1 = surviving exchange index; turns = "before any asks"
	switch k.String() {
	case "esc", "r", "b":
		m.rewindOpen = false
		return m, nil, false, false
	case "q", "ctrl+c":
		return m, tea.Quit, false, false
	case "up", "k":
		if m.rewindIdx > 0 {
			m.rewindIdx--
		}
	case "down", "j":
		if m.rewindIdx < maxIdx {
			m.rewindIdx++
		}
	case "enter":
		target := m.rewindIdx
		if m.rewindIdx == turns {
			target = -1
		}
		if m.rewindMode == "branch" {
			cmd := branchCmd(m.session, target, m.session.Case.ID)
			m.rewindOpen = false
			return m, cmd, false, false
		}
		if err := m.session.RewindTo(target); err != nil {
			m.lastErr = err.Error()
		} else {
			m.lastErr = ""
		}
		m.rewindOpen = false
		if topics := m.session.VisibleTopics(); m.topicIdx >= len(topics) && len(topics) > 0 {
			m.topicIdx = len(topics) - 1
		}
		return m, nil, false, false
	}
	return m, nil, false, false
}

// branchCmd dispatches Session.Branch off-thread (the SQLite copy
// can take a moment) and returns a branchedMsg the router handles.
func branchCmd(s *game.Session, turn int, caseID string) tea.Cmd {
	return func() tea.Msg {
		dst, err := newBranchSavePath(caseID, s.Timeline)
		if err != nil {
			return branchedMsg{err: err}
		}
		child, err := s.Branch(turn, dst)
		return branchedMsg{session: child, err: err}
	}
}

// askCmd dispatches Session.Ask in a goroutine so the TUI loop stays
// responsive while the LLM is in flight.
func askCmd(s *game.Session, topic string, technique kase.Technique) tea.Cmd {
	return func() tea.Msg {
		_, err := s.Ask(context.Background(), topic, technique)
		return witnessRespondedMsg{err: err}
	}
}

// Update returns (next, cmd, back, done). back=true returns to
// splash; done=true transitions to the reconstruction screen.
func (m interrogationModel) Update(msg tea.Msg) (interrogationModel, tea.Cmd, bool, bool) {
	// Async response from a previous ask.
	if r, ok := msg.(witnessRespondedMsg); ok {
		m.pending = nil
		if r.err != nil {
			if errors.Is(r.err, game.ErrSessionEnded) {
				m.lastErr = "the witness leaves"
				// Budget tripped during this ask: auto-advance to
				// reconstruction so the player isn't stranded.
				return m, nil, false, true
			}
			m.lastErr = r.err.Error()
		} else {
			m.lastErr = ""
		}
		return m, nil, false, false
	}

	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil, false, false
	}

	// Rewind picker captures all keys while open.
	if m.rewindOpen {
		return m.updateRewind(k)
	}
	// Inspector overlay captures all keys while open.
	if m.inspectorOpen {
		next, cmd, done := m.inspector.Update(msg)
		m.inspector = next
		if done {
			m.inspectorOpen = false
		}
		return m, cmd, false, false
	}

	topics := m.session.VisibleTopics()
	if m.topicIdx >= len(topics) && len(topics) > 0 {
		m.topicIdx = len(topics) - 1
	}
	switch k.String() {
	case "q", "ctrl+c":
		return m, tea.Quit, false, false
	case "esc":
		return m, nil, true, false
	case "d":
		// Player chose "I'm done" — end the session and transition
		// to reconstruction. PRD §2.3.
		m.session.EndSession()
		return m, nil, false, true
	case "r":
		if m.pending != nil || m.session.TurnCount() == 0 {
			return m, nil, false, false
		}
		m.rewindOpen = true
		m.rewindMode = "rewind"
		m.rewindIdx = m.session.TurnCount() - 1
		return m, nil, false, false
	case "b":
		if m.pending != nil || m.session.TurnCount() == 0 {
			return m, nil, false, false
		}
		m.rewindOpen = true
		m.rewindMode = "branch"
		m.rewindIdx = m.session.TurnCount() - 1
		return m, nil, false, false
	case "i":
		// Open the inspector over the current save file.
		m.inspector = newInspector(m.session.SavePath())
		m.inspectorOpen = true
		return m, nil, false, false
	case "tab":
		if m.focus == paneTopics {
			m.focus = paneTechniques
		} else {
			m.focus = paneTopics
		}
		return m, nil, false, false
	case "left", "h":
		m.focus = paneTopics
		return m, nil, false, false
	case "right", "l":
		m.focus = paneTechniques
		return m, nil, false, false
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
			return m, nil, false, false
		}
		topic := topics[m.topicIdx].Name
		tech := kase.AllTechniques[m.techIdx]
		m.pending = &pendingAsk{topic: topic, technique: tech}
		m.lastErr = ""
		return m, askCmd(m.session, topic, tech), false, false
	}
	return m, nil, false, false
}

func (m interrogationModel) View() string {
	if m.inspectorOpen {
		return m.inspector.View()
	}
	if m.rewindOpen {
		return m.renderRewindPicker()
	}

	header := lipgloss.JoinHorizontal(
		lipgloss.Top,
		styleTitle.Render("hearsay"),
		styleDim.Render("  ·  "+m.session.Case.ID),
		styleDim.Render("  ·  "+m.session.Timeline),
		styleDim.Render("  ·  "+m.session.ClockDisplay()),
	)

	demeanor := styleMuted.Render("she is " + string(m.session.CurrentDemeanor()) + ".")

	dialogue := m.renderDialogue()
	topics := m.renderTopics()
	techs := m.renderTechniques()

	bottom := lipgloss.JoinHorizontal(lipgloss.Top, topics, techs)
	footer := styleDim.Render("↑↓ choose · ←→ pane · enter ask · r rewind · b branch · i inspector · d done · esc back · q quit")

	parts := []string{header, demeanor, "", dialogue, "", bottom, "", footer}
	if m.lastErr != "" {
		parts = append(parts, styleDim.Render("err: "+m.lastErr))
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// dialogueWindow is how many recent exchanges the dialogue pane
// renders. Older turns are summarised with a "(N earlier turns)"
// header so the topic + technique panes stay visible without
// scrolling. The full history is always available via the inspector.
const dialogueWindow = 6

func (m interrogationModel) renderDialogue() string {
	log := m.session.Log()
	if len(log) == 0 && m.pending == nil {
		return styleMuted.Render("(she's waiting.)")
	}

	start := 0
	if len(log) > dialogueWindow {
		start = len(log) - dialogueWindow
	}

	var b strings.Builder
	if start > 0 {
		fmt.Fprintf(&b, "%s\n\n",
			styleMuted.Render(fmt.Sprintf("(%d earlier turns — open the inspector with i)", start)))
	}
	for i := start; i < len(log); i++ {
		ex := log[i]
		if i > start {
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
	focused := m.focus == paneTopics
	var b strings.Builder
	if focused {
		b.WriteString(styleTitle.Render("ASK ABOUT"))
	} else {
		b.WriteString(styleMuted.Render("ASK ABOUT"))
	}
	b.WriteString("\n")
	for i, t := range m.session.VisibleTopics() {
		b.WriteString(renderRow(t.Name, i == m.topicIdx, focused))
		b.WriteString("\n")
	}
	return lipgloss.NewStyle().Width(28).Render(b.String())
}

// renderRewindPicker shows the past exchanges as a selectable list
// plus a "before any asks" entry. Selecting an entry rewinds (or
// branches at) that point. PRD §7.2.4.
func (m interrogationModel) renderRewindPicker() string {
	title := "rewind to..."
	hint := "↑↓ select · enter rewind here · esc cancel"
	if m.rewindMode == "branch" {
		title = "branch from..."
		hint = "↑↓ select · enter branch here · esc cancel"
	}
	var b strings.Builder
	b.WriteString(styleTitle.Render(title))
	b.WriteString("\n\n")

	log := m.session.Log()
	rows := make([]string, 0, len(log)+1)
	for i, ex := range log {
		preview := ex.Witness
		if len(preview) > 60 {
			preview = preview[:57] + "..."
		}
		row := fmt.Sprintf("turn %d · %s (%s)", i, ex.Topic, ex.Technique.Label())
		if preview != "" {
			row += "  " + styleMuted.Render("— "+preview)
		}
		rows = append(rows, row)
	}
	rows = append(rows, styleMuted.Render("(before any asks)"))

	for i, row := range rows {
		if i == m.rewindIdx {
			b.WriteString(styleSelected.Render("▸ " + row))
		} else {
			b.WriteString(styleDim.Render("  " + row))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styleDim.Render(hint))
	return b.String()
}

func (m interrogationModel) renderTechniques() string {
	focused := m.focus == paneTechniques
	var b strings.Builder
	if focused {
		b.WriteString(styleTitle.Render("HOW"))
	} else {
		b.WriteString(styleMuted.Render("HOW"))
	}
	b.WriteString("\n")
	for i, tech := range kase.AllTechniques {
		b.WriteString(renderRow(tech.Label(), i == m.techIdx, focused))
		b.WriteString("\n")
	}
	return lipgloss.NewStyle().Width(28).Render(b.String())
}

// renderRow draws one option row inside a pane. Only the focused
// pane shows the ▸ cursor; unfocused panes are flat lists. The
// selected-but-unfocused item gets a quieter dot mark so the
// player can see where they'll land when focus returns.
func renderRow(label string, selected, focused bool) string {
	switch {
	case selected && focused:
		return styleSelected.Render("▸ " + label)
	case selected && !focused:
		return styleDim.Render("· " + label)
	default:
		return styleMuted.Render("  " + label)
	}
}
