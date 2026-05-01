package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jerkeyray/starling/event"
	"github.com/jerkeyray/starling/eventlog"
)

// inspectorModel is a read-only TUI panel over the session's SQLite
// event log. Each row is one event in the chain across all runs in
// the file. The user can scroll, expand a row to read its payload,
// and toggle between flat and grouped views. PRD §3.8 / §7.2.5.
//
// Reading is done through eventlog.NewSQLite(WithReadOnly) so the
// panel cannot mutate the audit log even on a misbehaving Update.
type inspectorModel struct {
	savePath string
	events   []event.Event // flat across all runs
	cursor   int
	expanded bool
	loadErr  string
}

// newInspector opens the SQLite log at savePath read-only and loads
// every event across every run. Failure surfaces in the model's
// loadErr; the model is still renderable so the user gets a
// diagnostic instead of a silent crash.
func newInspector(savePath string) inspectorModel {
	m := inspectorModel{savePath: savePath}
	if savePath == "" {
		m.loadErr = "no save file (stub session?)"
		return m
	}
	log, err := eventlog.NewSQLite(savePath, eventlog.WithReadOnly())
	if err != nil {
		m.loadErr = err.Error()
		return m
	}
	defer log.Close()

	ctx := context.Background()
	lister, ok := log.(eventlog.RunLister)
	if !ok {
		m.loadErr = "log does not implement RunLister"
		return m
	}
	runs, err := lister.ListRuns(ctx)
	if err != nil {
		m.loadErr = "list runs: " + err.Error()
		return m
	}
	for _, r := range runs {
		evs, err := log.Read(ctx, r.RunID)
		if err != nil {
			m.loadErr = "read " + r.RunID + ": " + err.Error()
			return m
		}
		m.events = append(m.events, evs...)
	}
	return m
}

func (m inspectorModel) Init() tea.Cmd { return nil }

// Update returns (next, cmd, done). done=true asks the parent to
// close the inspector and return to the underlying screen.
func (m inspectorModel) Update(msg tea.Msg) (inspectorModel, tea.Cmd, bool) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil, false
	}
	switch k.String() {
	case "q", "ctrl+c":
		return m, tea.Quit, false
	case "esc", "i":
		return m, nil, true
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.events)-1 {
			m.cursor++
		}
	case "enter":
		m.expanded = !m.expanded
	case "g":
		m.cursor = 0
	case "G":
		if len(m.events) > 0 {
			m.cursor = len(m.events) - 1
		}
	}
	return m, nil, false
}

func (m inspectorModel) View() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("inspector"))
	b.WriteString(styleDim.Render(fmt.Sprintf("  ·  %d events", len(m.events))))
	b.WriteString("\n\n")

	if m.loadErr != "" {
		b.WriteString(styleDim.Render("could not read log:\n  " + m.loadErr))
		b.WriteString("\n\n")
		b.WriteString(styleDim.Render("esc back · q quit"))
		return b.String()
	}

	if len(m.events) == 0 {
		b.WriteString(styleMuted.Render("(no events yet)\n\n"))
		b.WriteString(styleDim.Render("esc back · q quit"))
		return b.String()
	}

	// Render a windowed view of ~20 rows centered on the cursor so
	// the panel is bounded regardless of log size.
	const window = 20
	start := m.cursor - window/2
	if start < 0 {
		start = 0
	}
	end := start + window
	if end > len(m.events) {
		end = len(m.events)
		start = end - window
		if start < 0 {
			start = 0
		}
	}

	for i := start; i < end; i++ {
		ev := m.events[i]
		row := fmt.Sprintf("  %4d  %-22s  %s",
			ev.Seq,
			truncate(ev.Kind.String(), 22),
			truncate(shortRunID(ev.RunID), 14),
		)
		if i == m.cursor {
			b.WriteString(styleSelected.Render("▸" + row[1:]))
		} else {
			b.WriteString(styleMuted.Render(row))
		}
		b.WriteString("\n")
	}

	if m.expanded {
		ev := m.events[m.cursor]
		b.WriteString("\n")
		b.WriteString(styleDim.Render(strings.Repeat("─", 50)))
		b.WriteString("\n")
		b.WriteString(styleDim.Render(fmt.Sprintf("seq:        %d", ev.Seq)))
		b.WriteString("\n")
		b.WriteString(styleDim.Render(fmt.Sprintf("run:        %s", ev.RunID)))
		b.WriteString("\n")
		b.WriteString(styleDim.Render(fmt.Sprintf("kind:       %s", ev.Kind)))
		b.WriteString("\n")
		b.WriteString(styleDim.Render(fmt.Sprintf("ts:         %s", time.Unix(0, ev.Timestamp).UTC().Format(time.RFC3339Nano))))
		b.WriteString("\n")
		b.WriteString(styleDim.Render(fmt.Sprintf("prev_hash:  %x", ev.PrevHash)))
		b.WriteString("\n")
		b.WriteString(styleDim.Render(fmt.Sprintf("payload:    %d bytes (cbor)", len(ev.Payload))))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styleDim.Render("↑↓ scroll · enter expand · g/G top/bottom · esc back · q quit"))
	return b.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n < 1 {
		return s
	}
	return s[:n-1] + "…"
}

// shortRunID renders the last 12 characters of a (possibly long)
// run id so the inspector row stays narrow.
func shortRunID(id string) string {
	if len(id) <= 12 {
		return id
	}
	return "…" + id[len(id)-12:]
}
