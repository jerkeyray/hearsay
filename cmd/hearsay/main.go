package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jerkeyray/starling"
	"github.com/jerkeyray/starling/eventlog"
	"github.com/jerkeyray/starling/replay"

	"github.com/jerkeyray/hearsay/cases"
	"github.com/jerkeyray/hearsay/internal/kase"
	"github.com/jerkeyray/hearsay/internal/ui"
	"github.com/jerkeyray/hearsay/internal/witness"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "inspect":
			if err := runInspect(args[1:]); err != nil {
				fmt.Fprintln(os.Stderr, "hearsay inspect:", err)
				os.Exit(1)
			}
			return
		case "-h", "--help", "help":
			printUsage()
			return
		}
	}

	// Default: launch the TUI.
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

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  hearsay              # play (TUI)")
	fmt.Fprintln(os.Stderr, "  hearsay inspect <db> # open Starling's HTTP inspector against a save file")
}

// runInspect dispatches to Starling's bundled inspector. With an LLM
// provider configured in the environment, the Replay button works
// (the same agent build replays recorded runs); without one, the
// inspector runs read-only.
func runInspect(args []string) error {
	// Pull the case ID out of the save filename so we can configure
	// the replay agent's beliefs. Saves are named
	//   <caseID>-<sessionID>.db
	// or, for branches,
	//   <caseID>-<sessionID>-<parentTimeline>-branch.db
	var caseID string
	for _, a := range args {
		if strings.HasSuffix(a, ".db") || strings.Contains(a, ".db?") {
			caseID = caseIDFromPath(a)
			break
		}
	}

	var factory replay.Factory
	if caseID != "" {
		c, ok := cases.ByID(caseID)
		if ok {
			if live, err := witness.NewLiveProviderFromEnv(); err == nil {
				factory = func(_ context.Context) (replay.Agent, error) {
					return live.BuildReplayAgent(eventlog.NewInMemory(), c.Beliefs), nil
				}
			} else {
				fmt.Fprintln(os.Stderr,
					"hearsay inspect: no LLM provider configured; running read-only (Replay disabled).")
			}
		} else {
			fmt.Fprintf(os.Stderr,
				"hearsay inspect: case %q not found in this build; running read-only.\n", caseID)
		}
	}

	cmd := starling.InspectCommand(factory)
	cmd.Name = "hearsay inspect"
	return cmd.Run(args)
}

// caseIDFromPath strips the directory and extension and returns the
// case ID prefix (everything up to the first "-"). Returns "" on
// shapes it doesn't recognise.
func caseIDFromPath(p string) string {
	base := filepath.Base(p)
	base = strings.TrimSuffix(base, ".db")
	if i := strings.IndexByte(base, '-'); i > 0 {
		return base[:i]
	}
	return ""
}

// buildDriverFactory returns a DriverFactory and an optional notice
// message. When an LLM provider is configured via env, the factory
// produces a LiveDriver per session. Otherwise it falls back to the
// canned-line stub. Both write a SQLite save to the given path so
// the inspector + verify path work either way.
func buildDriverFactory() (ui.DriverFactory, string) {
	live, err := witness.NewLiveProviderFromEnv()
	if err != nil {
		notice := "hearsay: " + err.Error() + "\n" +
			"        running with the canned-line stub witness (saves still recorded for the inspector); set\n" +
			"        ANTHROPIC_API_KEY=...   (default, claude-sonnet-4-6)\n" +
			"        OPENAI_API_KEY=...      (or PROVIDER=openai)\n" +
			"        and re-run for a real interrogation."
		return func(_ context.Context, _ kase.Case, savePath string) (witness.Driver, error) {
			return witness.NewStubDriverWithSave(savePath)
		}, notice
	}
	return func(_ context.Context, c kase.Case, savePath string) (witness.Driver, error) {
		return live.NewDriver(savePath, c)
	}, ""
}
