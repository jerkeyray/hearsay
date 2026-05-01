package witness

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jerkeyray/starling"
	"github.com/jerkeyray/starling/eventlog"
	"github.com/jerkeyray/starling/provider"
	"github.com/jerkeyray/starling/provider/anthropic"
	"github.com/jerkeyray/starling/provider/openai"
	"github.com/jerkeyray/starling/tool"

	"github.com/jerkeyray/hearsay/internal/kase"
)

// DefaultBudget is the per-session token / USD cap from PRD §3.5 / §8.5.
// Mapped onto the session clock in step 13.
var DefaultBudget = &starling.Budget{
	MaxOutputTokens: 50_000,
	MaxUSD:          0.40,
	MaxWallClock:    30 * time.Minute,
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
// returns a LiveDriver that will write all run events into it. The
// returned driver owns the event log; Close releases it.
func (p *LiveProvider) NewDriver(savePath string) (*LiveDriver, error) {
	log, err := eventlog.NewSQLite(savePath)
	if err != nil {
		return nil, fmt.Errorf("open event log: %w", err)
	}
	return newLiveDriver(p.Provider, p.Model, p.Budget, log), nil
}

// LiveDriver runs a starling.Agent per ask against a real (or scripted)
// provider, writing events to a per-session SQLite log. Each Respond
// invocation mints a fresh RunID; all runs share the underlying log.
type LiveDriver struct {
	agent *starling.Agent
	log   eventlog.EventLog
}

// NewLiveDriverWith is a low-level constructor used by tests with a
// scripted provider and an in-memory event log. Production code uses
// LiveProvider.NewDriver.
func NewLiveDriverWith(p provider.Provider, model string, budget *starling.Budget, log eventlog.EventLog) *LiveDriver {
	return newLiveDriver(p, model, budget, log)
}

func newLiveDriver(p provider.Provider, model string, budget *starling.Budget, log eventlog.EventLog) *LiveDriver {
	a := &starling.Agent{
		Provider: p,
		Tools:    []tool.Tool{RecallTool(case1Beliefs)},
		Log:      log,
		Budget:   budget,
		Config: starling.Config{
			Model:        model,
			SystemPrompt: SystemPrompt,
			MaxTurns:     4,
		},
	}
	return &LiveDriver{agent: a, log: log}
}

// Respond runs one starling.Agent.Run for the given (topic, technique)
// pair and returns the witness's final line plus the usage numbers
// from the run. The conversation history is embedded in the user
// prompt so each Run is self-contained.
func (d *LiveDriver) Respond(ctx context.Context, topic string, technique kase.Technique, history []HistoryItem) (Response, error) {
	goal := UserPrompt(topic, technique, history)
	res, err := d.agent.Run(ctx, goal)
	if err != nil {
		return Response{}, err
	}
	return Response{
		Text:         res.FinalText,
		InputTokens:  res.InputTokens,
		OutputTokens: res.OutputTokens,
		CostUSD:      res.TotalCostUSD,
	}, nil
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
