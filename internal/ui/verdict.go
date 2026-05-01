package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jerkeyray/hearsay/internal/game"
)

// verdictModel renders the post-reconstruction reveal: per-question
// rows showing player vs witness vs truth with an error-kind
// classification, and a qualitative grade. PRD §7.2.7.
//
// The verify-chain modal lands in M5 polish; this screen has a
// placeholder stub for it.
type verdictModel struct {
	verdict game.Verdict
}

func newVerdict(v game.Verdict) verdictModel { return verdictModel{verdict: v} }

func (m verdictModel) Init() tea.Cmd { return nil }

// Update returns (next, cmd, done). done=true returns to splash.
func (m verdictModel) Update(msg tea.Msg) (verdictModel, tea.Cmd, bool) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "q", "ctrl+c":
			return m, tea.Quit, false
		case "enter", "esc", " ", "space":
			return m, nil, true
		}
	}
	return m, nil, false
}

func (m verdictModel) View() string {
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
	b.WriteString(styleDim.Render("[ verify chain ] (M5 polish)   [ enter / esc to return ]"))

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
