package ui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jerkeyray/hearsay/internal/game"
)

// verdictModel renders the post-reconstruction reveal: per-question
// rows showing player vs witness vs truth with an error-kind
// classification, and a qualitative grade. PRD §7.2.7.
//
// Pressing v walks the SQLite event log and shows the hash-chain
// verify modal (PRD §3.8).
type verdictModel struct {
	verdict     game.Verdict
	savePath    string
	verifyOpen  bool
	verifyResult *game.VerifyResult
	verifyErr   string
}

func newVerdict(v game.Verdict, savePath string) verdictModel {
	return verdictModel{verdict: v, savePath: savePath}
}

func (m verdictModel) Init() tea.Cmd { return nil }

// Update returns (next, cmd, done). done=true returns to splash.
func (m verdictModel) Update(msg tea.Msg) (verdictModel, tea.Cmd, bool) {
	if k, ok := msg.(tea.KeyMsg); ok {
		s := k.String()
		if m.verifyOpen {
			switch s {
			case "esc", "v", "enter", " ", "space":
				m.verifyOpen = false
			case "q", "ctrl+c":
				return m, tea.Quit, false
			}
			return m, nil, false
		}
		switch s {
		case "q", "ctrl+c":
			return m, tea.Quit, false
		case "v":
			r, err := game.Verify(context.Background(), m.savePath)
			if err != nil {
				m.verifyErr = err.Error()
				m.verifyResult = nil
			} else {
				m.verifyErr = ""
				m.verifyResult = &r
			}
			m.verifyOpen = true
			return m, nil, false
		case "enter", "esc", " ", "space":
			return m, nil, true
		}
	}
	return m, nil, false
}

func (m verdictModel) View() string {
	if m.verifyOpen {
		return m.renderVerify()
	}

	var b strings.Builder
	b.WriteString(styleTitle.Render("verdict"))
	b.WriteString("\n\n")

	for _, item := range m.verdict.Items {
		b.WriteString(m.renderItem(item))
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "\n%s\n", styleSelected.Render(
		fmt.Sprintf("%d of %d correct.", m.verdict.Score, m.verdict.Total),
	))
	if m.verdict.Summary != "" {
		b.WriteString(styleDim.Render(m.verdict.Summary))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(styleDim.Render("v verify chain   ·   enter/esc return"))

	return styleBorder.Render(b.String())
}

func (m verdictModel) renderVerify() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("verify chain"))
	b.WriteString("\n\n")

	if m.verifyErr != "" {
		b.WriteString(styleDim.Render("could not verify:\n  " + m.verifyErr))
		b.WriteString("\n\n")
		b.WriteString(styleDim.Render("esc / v to return"))
		return styleBorder.Render(b.String())
	}
	r := m.verifyResult
	if r == nil {
		return styleBorder.Render("(no result)")
	}

	row := func(label, val string) {
		fmt.Fprintf(&b, "  %s  %s\n", styleMuted.Render(pad(label, 18)), val)
	}
	row("file:", r.Path)
	row("runs:", fmt.Sprintf("%d", r.RunCount))
	row("events:", fmt.Sprintf("%d", r.EventCount))
	row("first event seq:", fmt.Sprintf("%d", r.FirstSeq))
	row("first hash:", fmt.Sprintf("%x", r.FirstHash))
	row("first at:", r.FirstAt.UTC().Format("2006-01-02 15:04:05Z"))
	row("last event seq:", fmt.Sprintf("%d", r.LastSeq))
	row("last hash:", fmt.Sprintf("%x", r.LastHash))
	row("last at:", r.LastAt.UTC().Format("2006-01-02 15:04:05Z"))
	b.WriteString("\n")

	if r.OK {
		b.WriteString(styleSelected.Render("the chain is intact."))
	} else {
		b.WriteString(styleDim.Render("the chain is broken: " + r.Reason))
	}
	b.WriteString("\n\n")
	b.WriteString(styleDim.Render("esc / v to return"))
	return styleBorder.Render(b.String())
}

func (m verdictModel) renderItem(it game.VerdictItem) string {
	var b strings.Builder

	header := styleSelected.Render(it.Prompt)
	if !it.Correct && it.Error != game.NoCanonicalAnswer {
		header = styleDim.Render(it.Prompt)
	}
	b.WriteString(header)
	b.WriteString("\n")

	row := func(label, val string) {
		fmt.Fprintf(&b, "    %s %s\n", styleMuted.Render(pad(label, 12)), val)
	}
	row("you said:", it.Player)
	if it.Witness != "" {
		row("she said:", it.Witness)
	}
	if it.Truth != "" {
		row("the truth:", it.Truth)
	}

	verdictLine := it.Error.Label()
	if it.Correct {
		verdictLine = styleSelected.Render(verdictLine + "  (correct)")
	} else {
		verdictLine = styleDim.Render(verdictLine)
	}
	fmt.Fprintf(&b, "    %s\n", verdictLine)
	return b.String()
}

func pad(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}
