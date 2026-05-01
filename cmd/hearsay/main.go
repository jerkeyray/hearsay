package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jerkeyray/hearsay/cases"
	"github.com/jerkeyray/hearsay/internal/kase"
	"github.com/jerkeyray/hearsay/internal/ui"
	"github.com/jerkeyray/hearsay/internal/witness"
)

func main() {
	factory, notice := buildDriverFactory()
	if notice != "" {
		fmt.Fprintln(os.Stderr, notice)
	}
	p := tea.NewProgram(ui.New(factory, cases.All), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "hearsay:", err)
		os.Exit(1)
	}
}

// buildDriverFactory returns a DriverFactory and an optional notice
// message. When an LLM provider is configured via env, the factory
// produces a LiveDriver per session. Otherwise it falls back to the
// canned-line stub and the notice tells the user what they're
// missing.
func buildDriverFactory() (ui.DriverFactory, string) {
	live, err := witness.NewLiveProviderFromEnv()
	if err != nil {
		notice := "hearsay: " + err.Error() + "\n" +
			"        running with the canned-line stub witness; set\n" +
			"        ANTHROPIC_API_KEY=...   (default, claude-sonnet-4-6)\n" +
			"        OPENAI_API_KEY=...      (or PROVIDER=openai)\n" +
			"        and re-run for a real interrogation."
		return func(_ context.Context, _ kase.Case, _ string) (witness.Driver, error) {
			return witness.NewStubDriver(), nil
		}, notice
	}
	return func(_ context.Context, c kase.Case, savePath string) (witness.Driver, error) {
		return live.NewDriver(savePath, c)
	}, ""
}
