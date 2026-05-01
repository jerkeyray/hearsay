package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// helpModel is a global help overlay reachable from anywhere via "?".
type helpModel struct{}

func newHelp() helpModel { return helpModel{} }

func (m helpModel) Init() tea.Cmd { return nil }

// Update returns done=true when the user dismisses the overlay.
func (m helpModel) Update(msg tea.Msg) (helpModel, tea.Cmd, bool) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "q", "ctrl+c":
			return m, tea.Quit, false
		case "esc", "?", "enter", " ", "space":
			return m, nil, true
		}
	}
	return m, nil, false
}

func (m helpModel) View() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("hearsay — help"))
	b.WriteString("\n\n")

	section := func(title string, rows [][2]string) {
		b.WriteString(styleSelected.Render(title))
		b.WriteString("\n")
		for _, r := range rows {
			b.WriteString("  ")
			b.WriteString(styleMuted.Render(pad(r[0], 14)))
			b.WriteString("  ")
			b.WriteString(styleDim.Render(r[1]))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	section("the five techniques", [][2]string{
		{"directly", "ask straight; the witness gives you what she most readily believes."},
		{"the moment before", "shift the anchor; suppressed memories sometimes surface."},
		{"how do you know", "force the source; implants thin, confabulations turn circular."},
		{"push back", "challenge; real holds, drift drifts further, implants double down."},
		{"circle back later", "mark the topic to ask again; useful with the inspector's diff view."},
	})

	section("interrogation", [][2]string{
		{"enter", "ask the selected (topic, technique)."},
		{"tab", "switch between topic and technique panes."},
		{"↑↓ / jk", "navigate within the focused pane."},
		{"r", "rewind to a prior turn (in this timeline)."},
		{"b", "fork a branch from a prior turn."},
		{"i", "open the inspector (read-only event log)."},
		{"d", "stop interrogating; go to reconstruction."},
		{"esc", "back to splash."},
	})

	section("after the session", [][2]string{
		{"reconstruction", "answer the form. ↑↓ navigate, ←→ pick, space toggle multi, tab don't-know, s submit."},
		{"verdict", "v verifies the hash chain; enter/esc returns to splash."},
	})

	section("global", [][2]string{
		{"?", "this help."},
		{"q / ctrl+c", "quit."},
	})

	section("watching the api", [][2]string{
		{"i (in-game)", "inline inspector — every starling event."},
		{"hearsay inspect <db>", "starling's web inspector + replay button."},
		{"HEARSAY_DEBUG=1", "stream slog → ~/.hearsay/debug.log (info)."},
		{"HEARSAY_DEBUG=2", "same, debug-level."},
	})

	b.WriteString(styleDim.Render("press ? or esc to return."))
	return b.String()
}
