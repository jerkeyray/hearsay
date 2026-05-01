package witness

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jerkeyray/starling"
	"github.com/jerkeyray/starling/eventlog"
	"github.com/jerkeyray/starling/provider"
	"github.com/jerkeyray/starling/provider/anthropic"
	"github.com/jerkeyray/starling/provider/openai"
	"github.com/jerkeyray/starling/tool"

	"github.com/jerkeyray/hearsay/internal/kase"
)

// DefaultBudget is the per-session token / USD cap. The session-clock
// math in game.Session maps 1000 output tokens → 1 minute of game
// time, so 3000 tokens reads as a 3:00 starting clock. WallClock is
// kept generous so a slow LLM call doesn't kill an in-flight ask.
var DefaultBudget = &starling.Budget{
	MaxOutputTokens: 3_000,
	MaxUSD:          0.05,
	MaxWallClock:    10 * time.Minute,
}

// LiveProvider holds the provider-level config that lasts across
// sessions: the LLM connection, the model id, and the budget. One
// LiveProvider is built at app start; per-session LiveDrivers are
// minted from it.
type LiveProvider struct {
	Provider   provider.Provider
	ProviderID string // "anthropic" | "openai"
	Model      string
	Budget     *starling.Budget
}

// NewLiveProviderFromEnv reads PROVIDER / ANTHROPIC_API_KEY /
// OPENAI_API_KEY / MODEL and returns a configured LiveProvider.
// Defaults: Anthropic + claude-sonnet-4-6 if ANTHROPIC_API_KEY is
// set, OpenAI + gpt-4o-mini if OPENAI_API_KEY is set, error otherwise.
// PRD §6.4 / §8.2.
func NewLiveProviderFromEnv() (*LiveProvider, error) {
	choice := os.Getenv("PROVIDER")
	if choice == "" {
		switch {
		case os.Getenv("ANTHROPIC_API_KEY") != "":
			choice = "anthropic"
		case os.Getenv("OPENAI_API_KEY") != "":
			choice = "openai"
		default:
			return nil, errors.New("no LLM provider configured: set ANTHROPIC_API_KEY or OPENAI_API_KEY")
		}
	}
	switch choice {
	case "anthropic":
		key := os.Getenv("ANTHROPIC_API_KEY")
		if key == "" {
			return nil, errors.New("ANTHROPIC_API_KEY not set")
		}
		model := os.Getenv("MODEL")
		if model == "" {
			model = "claude-sonnet-4-6"
		}
		p, err := anthropic.New(anthropic.WithAPIKey(key))
		if err != nil {
			return nil, fmt.Errorf("anthropic.New: %w", err)
		}
		return &LiveProvider{Provider: p, ProviderID: "anthropic", Model: model, Budget: DefaultBudget}, nil
	case "openai":
		key := os.Getenv("OPENAI_API_KEY")
		if key == "" {
			return nil, errors.New("OPENAI_API_KEY not set")
		}
		model := os.Getenv("MODEL")
		if model == "" {
			model = "gpt-4o-mini"
		}
		opts := []openai.Option{openai.WithAPIKey(key)}
		if u := os.Getenv("OPENAI_BASE_URL"); u != "" {
			opts = append(opts, openai.WithBaseURL(u))
		}
		p, err := openai.New(opts...)
		if err != nil {
			return nil, fmt.Errorf("openai.New: %w", err)
		}
		return &LiveProvider{Provider: p, ProviderID: "openai", Model: model, Budget: DefaultBudget}, nil
	default:
		return nil, fmt.Errorf("unknown PROVIDER %q (want anthropic | openai)", choice)
	}
}

// NewDriver opens a per-session SQLite event log at savePath and
// returns a LiveDriver wired to the given case's beliefs. The
// returned driver owns the event log; Close releases it.
func (p *LiveProvider) NewDriver(savePath string, c kase.Case) (*LiveDriver, error) {
	log, err := eventlog.NewSQLite(savePath)
	if err != nil {
		return nil, fmt.Errorf("open event log: %w", err)
	}
	return newLiveDriver(p.Provider, p.Model, p.Budget, log, c.Beliefs, savePath, p), nil
}

// Branch copies the underlying SQLite log (and any WAL/SHM
// sidecars) to dstPath, opens it, and returns a sibling LiveDriver
// that writes future runs into the copy. The original driver is
// unaffected.
func (d *LiveDriver) Branch(dstPath string) (Driver, error) {
	if d.savePath == "" || d.owner == nil {
		return nil, fmt.Errorf("branch: live driver has no save path (in-memory log?)")
	}
	if err := copySQLite(d.savePath, dstPath); err != nil {
		return nil, fmt.Errorf("branch: copy save: %w", err)
	}
	log, err := eventlog.NewSQLite(dstPath)
	if err != nil {
		return nil, fmt.Errorf("branch: open copy: %w", err)
	}
	return newLiveDriver(d.owner.Provider, d.owner.Model, d.owner.Budget, log, d.beliefs, dstPath, d.owner), nil
}

// copySQLite copies a SQLite database file plus its -wal and -shm
// sidecars (if present) to a parallel destination. WAL mode means
// recent writes may live entirely in the .db-wal file until a
// checkpoint, so a naive single-file copy can miss them.
func copySQLite(src, dst string) error {
	if err := copyOne(src, dst); err != nil {
		return err
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		s := src + suffix
		if _, err := os.Stat(s); err != nil {
			continue
		}
		if err := copyOne(s, dst+suffix); err != nil {
			return err
		}
	}
	return nil
}

func copyOne(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// LiveDriver runs a starling.Agent per ask against a real (or scripted)
// provider, writing events to a per-session SQLite log. Each Respond
// invocation builds a fresh agent so the per-ask demeanor sink is
// isolated; all runs share the underlying event log.
type LiveDriver struct {
	provider provider.Provider
	model    string
	budget   *starling.Budget
	log      eventlog.EventLog
	beliefs  map[string]kase.Belief
	// savePath is the on-disk path of the SQLite event log, captured
	// so Branch can copy the file. Empty when log is in-memory (the
	// scripted-provider tests path).
	savePath string
	// owner remembers which factory built us so Branch can rebuild
	// itself with a fresh eventlog at a new path. nil for tests.
	owner *LiveProvider
}

// NewLiveDriverWith is a low-level constructor used by tests with a
// scripted provider and an in-memory event log. Production code uses
// LiveProvider.NewDriver.
func NewLiveDriverWith(p provider.Provider, model string, budget *starling.Budget, log eventlog.EventLog, beliefs map[string]kase.Belief) *LiveDriver {
	return newLiveDriver(p, model, budget, log, beliefs, "", nil)
}

func newLiveDriver(p provider.Provider, model string, budget *starling.Budget, log eventlog.EventLog, beliefs map[string]kase.Belief, savePath string, owner *LiveProvider) *LiveDriver {
	return &LiveDriver{
		provider: p,
		model:    model,
		budget:   budget,
		log:      log,
		beliefs:  beliefs,
		savePath: savePath,
		owner:    owner,
	}
}

// Respond runs one starling.Agent.Run for the given (topic, technique)
// pair and returns the witness's final line, any demeanor the model
// signalled, and the usage numbers from the run. The conversation
// history is embedded in the user prompt so each Run is self-contained.
func (d *LiveDriver) Respond(ctx context.Context, topic string, technique kase.Technique, history []HistoryItem) (Response, error) {
	// Per-ask sink for note_demeanor. The closure runs on whichever
	// goroutine the agent's tool executor uses; a mutex keeps the
	// write/read pair safe.
	var (
		mu       sync.Mutex
		demeanor kase.Demeanor
	)
	setDemeanor := func(s kase.Demeanor) {
		mu.Lock()
		demeanor = s
		mu.Unlock()
	}

	agent := &starling.Agent{
		Provider: d.provider,
		Tools: []tool.Tool{
			RecallTool(d.beliefs),
			DemeanorTool(setDemeanor),
		},
		Log:    d.log,
		Budget: d.budget,
		Config: starling.Config{
			Model:        d.model,
			SystemPrompt: SystemPrompt,
			MaxTurns:     4,
			Logger:       debugLogger(),
		},
	}

	goal := UserPrompt(topic, technique, history)
	res, err := agent.Run(ctx, goal)
	if err != nil {
		return Response{}, err
	}

	mu.Lock()
	d2 := demeanor
	mu.Unlock()

	return Response{
		Text:         res.FinalText,
		Demeanor:     d2,
		InputTokens:  res.InputTokens,
		OutputTokens: res.OutputTokens,
		CostUSD:      res.TotalCostUSD,
	}, nil
}

// SavePathHint exposes the SQLite path so callers (the inspector
// panel) can re-open the log read-only without going through the
// Driver interface. Returns "" when the log is in-memory.
func (d *LiveDriver) SavePathHint() string { return d.savePath }

// DebugLogPath returns the path the HEARSAY_DEBUG log writes to.
// Honors HEARSAY_HOME, falls back to $HOME/.hearsay/debug.log.
func DebugLogPath() (string, error) {
	if h := os.Getenv("HEARSAY_HOME"); h != "" {
		return filepath.Join(h, "debug.log"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".hearsay", "debug.log"), nil
}

// debugLogger returns a slog.Logger writing to HEARSAY_HOME/debug.log
// when HEARSAY_DEBUG=1. Returns nil otherwise; nil disables Starling's
// logging pipeline at no cost.
//
// Logs are appended (not truncated) so multiple sessions accumulate;
// readers tail the file in another terminal.
func debugLogger() *slog.Logger {
	if os.Getenv("HEARSAY_DEBUG") != "1" {
		return nil
	}
	path, err := DebugLogPath()
	if err != nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil
	}
	level := slog.LevelInfo
	if os.Getenv("HEARSAY_DEBUG") == "2" {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: level}))
}

// Close releases the underlying event log.
func (d *LiveDriver) Close() error {
	if d.log == nil {
		return nil
	}
	err := d.log.Close()
	d.log = nil
	return err
}
