package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jerkeyray/hearsay/internal/game"
	"github.com/jerkeyray/hearsay/internal/kase"
)

// reconstructionModel renders the post-session questionnaire (PRD
// §3.6 / §7.2.6). One question per row, all visible at once;
// up/down moves between questions, left/right cycles within choices,
// space toggles multi-select, "k" marks "don't know," typing fills
// free text.
type reconstructionModel struct {
	form     kase.Form
	answers  []reconAnswer
	cursor   int
	doneHint bool
}

// reconAnswer is the in-flight UI state for one question. Mirrors
// game.Answer but with extra UI state (which Choice index is
// highlighted for radio).
type reconAnswer struct {
	choiceIdx int             // for Radio: index into Choices, -1 = none yet
	picked    map[int]bool    // for MultiSelect: set of selected indices
	freeText  string          // for FreeText: the typed buffer
	dontKnow  bool
}

func newReconstruction(form kase.Form) reconstructionModel {
	answers := make([]reconAnswer, len(form.Questions))
	for i, q := range form.Questions {
		answers[i] = reconAnswer{choiceIdx: -1}
		if q.Type == kase.MultiSelect {
			answers[i].picked = make(map[int]bool, len(q.Choices))
		}
	}
	return reconstructionModel{form: form, answers: answers}
}

func (m reconstructionModel) Init() tea.Cmd { return nil }

// Update returns (next, cmd, submit, back). submit=true means the
// player wants the verdict; back=true returns to the previous screen.
func (m reconstructionModel) Update(msg tea.Msg) (reconstructionModel, tea.Cmd, bool, bool) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil, false, false
	}

	q := m.form.Questions[m.cursor]
	a := &m.answers[m.cursor]

	// Free-text fields capture printable characters. Reserve only
	// global navigation keys (esc, ctrl+c, tab, up/down).
	if q.Type == kase.FreeText && !a.dontKnow {
		s := k.String()
		switch s {
		case "ctrl+c":
			return m, tea.Quit, false, false
		case "esc":
			return m, nil, false, true
		case "up":
			m.move(-1)
			return m, nil, false, false
		case "down":
			m.move(1)
			return m, nil, false, false
		case "enter":
			if m.cursor == len(m.form.Questions)-1 {
				return m, nil, true, false
			}
			m.move(1)
			return m, nil, false, false
		case "backspace":
			if len(a.freeText) > 0 {
				a.freeText = a.freeText[:len(a.freeText)-1]
			}
			return m, nil, false, false
		case "tab":
			a.dontKnow = !a.dontKnow
			return m, nil, false, false
		default:
			if len(s) == 1 || s == "space" {
				if s == "space" {
					a.freeText += " "
				} else {
					a.freeText += s
				}
			}
			return m, nil, false, false
		}
	}

	// Non-free-text path.
	switch k.String() {
	case "q", "ctrl+c":
		return m, tea.Quit, false, false
	case "esc":
		return m, nil, false, true
	case "up", "k":
		m.move(-1)
	case "down", "j":
		m.move(1)
	case "left", "h":
		if !a.dontKnow && (q.Type == kase.Radio || q.Type == kase.MultiSelect) {
			if a.choiceIdx > 0 {
				a.choiceIdx--
			} else if a.choiceIdx == -1 && len(q.Choices) > 0 {
				a.choiceIdx = 0
			}
		}
	case "right", "l":
		if !a.dontKnow && (q.Type == kase.Radio || q.Type == kase.MultiSelect) {
			if a.choiceIdx == -1 && len(q.Choices) > 0 {
				a.choiceIdx = 0
			} else if a.choiceIdx < len(q.Choices)-1 {
				a.choiceIdx++
			}
		}
	case " ", "space":
		switch q.Type {
		case kase.Radio:
			// Pressing space on radio cycles like →.
			if !a.dontKnow {
				if a.choiceIdx < len(q.Choices)-1 {
					a.choiceIdx++
				} else {
					a.choiceIdx = 0
				}
			}
		case kase.MultiSelect:
			if !a.dontKnow {
				if a.choiceIdx < 0 {
					a.choiceIdx = 0
				}
				a.picked[a.choiceIdx] = !a.picked[a.choiceIdx]
			}
		}
	case "tab":
		// Toggle don't-know.
		a.dontKnow = !a.dontKnow
	case "enter", "s":
		// Submit if on last question, else move down.
		if m.cursor == len(m.form.Questions)-1 || k.String() == "s" {
			return m, nil, true, false
		}
		m.move(1)
	}
	return m, nil, false, false
}

// move shifts the cursor by d, clamped.
func (m *reconstructionModel) move(d int) {
	n := len(m.form.Questions)
	if n == 0 {
		return
	}
	m.cursor += d
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= n {
		m.cursor = n - 1
	}
}

func (m reconstructionModel) View() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("reconstruct what happened"))
	b.WriteString("\n\n")

	for i, q := range m.form.Questions {
		focused := i == m.cursor
		b.WriteString(m.renderQuestion(q, &m.answers[i], focused))
		b.WriteString("\n\n")
	}

	footer := styleDim.Render(
		"↑↓ question · ←→/space choice · tab don't know · s/enter submit · esc back · q quit",
	)
	b.WriteString(footer)
	return b.String()
}

func (m reconstructionModel) renderQuestion(q kase.Question, a *reconAnswer, focused bool) string {
	var b strings.Builder
	prompt := q.Prompt
	if focused {
		b.WriteString(styleSelected.Render("▸ " + prompt))
	} else {
		b.WriteString(styleDim.Render("  " + prompt))
	}
	b.WriteString("\n")

	switch q.Type {
	case kase.Radio:
		b.WriteString("    ")
		for i, c := range q.Choices {
			marker := "( ) "
			if !a.dontKnow && a.choiceIdx == i {
				marker = "(•) "
			}
			label := marker + c
			if focused && a.choiceIdx == i && !a.dontKnow {
				label = styleSelected.Render(label)
			} else {
				label = styleMuted.Render(label)
			}
			b.WriteString(label)
			if i < len(q.Choices)-1 {
				b.WriteString("   ")
			}
		}
	case kase.MultiSelect:
		for i, c := range q.Choices {
			marker := "[ ] "
			if !a.dontKnow && a.picked[i] {
				marker = "[x] "
			}
			label := marker + c
			if focused && a.choiceIdx == i && !a.dontKnow {
				label = styleSelected.Render(label)
			} else {
				label = styleMuted.Render(label)
			}
			b.WriteString("    ")
			b.WriteString(label)
			b.WriteString("\n")
		}
		// Trim trailing newline for consistent spacing.
		s := b.String()
		b.Reset()
		b.WriteString(strings.TrimRight(s, "\n"))
	case kase.FreeText:
		text := a.freeText
		if focused && !a.dontKnow {
			text += "▏" // cursor
		}
		if text == "" {
			text = styleMuted.Render("(type your answer)")
		}
		b.WriteString("    " + text)
	}

	if a.dontKnow {
		b.WriteString("\n    " + styleSelected.Render("[ don't know ]"))
	} else {
		b.WriteString("    " + styleMuted.Render("· tab to mark don't know"))
	}
	return b.String()
}

// Result returns the player's answers as a game.Reconstruction.
func (m reconstructionModel) Result() game.Reconstruction {
	out := game.Reconstruction{
		Answers: make([]game.Answer, 0, len(m.form.Questions)),
	}
	for i, q := range m.form.Questions {
		a := m.answers[i]
		ans := game.Answer{QuestionID: q.ID, DontKnow: a.dontKnow}
		if a.dontKnow {
			out.Answers = append(out.Answers, ans)
			continue
		}
		switch q.Type {
		case kase.Radio:
			if a.choiceIdx >= 0 && a.choiceIdx < len(q.Choices) {
				ans.Choice = q.Choices[a.choiceIdx]
			}
		case kase.MultiSelect:
			for j, c := range q.Choices {
				if a.picked[j] {
					ans.Choices = append(ans.Choices, c)
				}
			}
		case kase.FreeText:
			ans.FreeText = a.freeText
		}
		out.Answers = append(out.Answers, ans)
	}
	return out
}

