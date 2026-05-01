// Package game owns the engine-side session state: the current case,
// the conversation log, the witness driver that produces each turn,
// and the per-session token budget. The driver is also the seam where
// the live LLM attaches and where Starling event-log writes happen
// for live runs.
//
// The UI holds cursor state and renders; the engine owns everything
// that would survive a process restart.
package game

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/jerkeyray/hearsay/internal/kase"
	"github.com/jerkeyray/hearsay/internal/witness"
)

// Budget caps the session's total LLM consumption. Maps onto the
// session clock the player sees in the interrogation header. Zero
// values disable that axis. PRD §3.5.
type Budget struct {
	MaxOutputTokens int64
	MaxUSD          float64
}

// DefaultBudget mirrors witness.DefaultBudget for the session-level
// view and is used when the caller does not provide one.
var DefaultBudget = Budget{
	MaxOutputTokens: 50_000,
	MaxUSD:          0.40,
}

// ErrSessionEnded is returned by Ask when the budget is exhausted.
// The fictional reading is "the witness leaves" (PRD §2.3 / §3.5).
var ErrSessionEnded = errors.New("the witness leaves")

// Exchange is one turn in the conversation: what the player asked,
// what the witness said back, the demeanor signalled that turn, and
// the usage that turn consumed.
type Exchange struct {
	Turn         int
	Topic        string
	Technique    kase.Technique
	Witness      string
	Demeanor     kase.Demeanor
	OutputTokens int64
	CostUSD      float64
}

// Session is the per-case engine state. One Session per save file.
//
// Session is safe for concurrent access: Bubble Tea runs Ask off the
// main loop via tea.Cmd while View renders on the main loop. mu
// guards every field below it.
type Session struct {
	Case   kase.Case
	driver witness.Driver

	mu               sync.RWMutex
	log              []Exchange
	budget           Budget
	usedOutputTokens int64
	usedCostUSD      float64
}

// NewSession constructs a session over a case + witness driver and
// per-session budget. A zero Budget falls back to DefaultBudget. The
// session does not own the driver's resources — Close releases them.
func NewSession(_ context.Context, c kase.Case, d witness.Driver, b Budget) (*Session, error) {
	if b.MaxOutputTokens == 0 && b.MaxUSD == 0 {
		b = DefaultBudget
	}
	return &Session{Case: c, driver: d, budget: b}, nil
}

// Ask runs one turn: build conversation history, invoke the driver,
// accumulate usage, append the exchange. Returns ErrSessionEnded if
// the budget is already exhausted. Safe to call concurrently with
// reading methods (Log, RemainingOutputTokens, ClockDisplay, etc.).
func (s *Session) Ask(ctx context.Context, topic string, technique kase.Technique) (Exchange, error) {
	// Snapshot the history under the read lock. Releasing before the
	// driver call lets the View render the existing log while the LLM
	// is in flight.
	s.mu.RLock()
	if s.budgetExhausted() {
		s.mu.RUnlock()
		return Exchange{}, ErrSessionEnded
	}
	hist := make([]witness.HistoryItem, 0, len(s.log))
	for _, ex := range s.log {
		hist = append(hist, witness.HistoryItem{
			Topic:     ex.Topic,
			Technique: ex.Technique,
			Witness:   ex.Witness,
		})
	}
	s.mu.RUnlock()

	resp, err := s.driver.Respond(ctx, topic, technique, hist)
	if err != nil {
		return Exchange{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.usedOutputTokens += resp.OutputTokens
	s.usedCostUSD += resp.CostUSD
	ex := Exchange{
		Turn:         len(s.log),
		Topic:        topic,
		Technique:    technique,
		Witness:      resp.Text,
		Demeanor:     resp.Demeanor,
		OutputTokens: resp.OutputTokens,
		CostUSD:      resp.CostUSD,
	}
	s.log = append(s.log, ex)
	return ex, nil
}

// CurrentDemeanor returns the demeanor recorded on the latest
// exchange, or DemeanorEngaged if no exchanges yet (the witness
// arrives engaged).
func (s *Session) CurrentDemeanor() kase.Demeanor {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.log) == 0 {
		return kase.DemeanorEngaged
	}
	last := s.log[len(s.log)-1]
	if last.Demeanor == "" {
		return kase.DemeanorEngaged
	}
	return last.Demeanor
}

// Close releases the driver's resources. Safe to call multiple times.
func (s *Session) Close(_ context.Context) error {
	if s.driver == nil {
		return nil
	}
	err := s.driver.Close()
	s.driver = nil
	return err
}

// Log returns a copy of the exchanges in order. Safe to call
// concurrently with Ask.
func (s *Session) Log() []Exchange {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Exchange, len(s.log))
	copy(out, s.log)
	return out
}

// TurnCount is the number of completed turns.
func (s *Session) TurnCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.log)
}

// Budget returns the session's caps.
func (s *Session) Budget() Budget {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.budget
}

// UsedOutputTokens is the cumulative output-token consumption across
// all asks in this session.
func (s *Session) UsedOutputTokens() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.usedOutputTokens
}

// UsedCostUSD is the cumulative provider cost across all asks.
func (s *Session) UsedCostUSD() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.usedCostUSD
}

// RemainingOutputTokens reports how many output tokens are left
// before the session ends. Returns 0 when exhausted, math.MaxInt64
// when unbounded.
func (s *Session) RemainingOutputTokens() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.remainingOutputTokensLocked()
}

func (s *Session) remainingOutputTokensLocked() int64 {
	if s.budget.MaxOutputTokens <= 0 {
		return int64(^uint64(0) >> 1) // unbounded → MaxInt64
	}
	r := s.budget.MaxOutputTokens - s.usedOutputTokens
	if r < 0 {
		return 0
	}
	return r
}

// SessionEnded reports whether the budget is exhausted.
func (s *Session) SessionEnded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.budgetExhausted()
}

// ClockDisplay formats remaining output tokens as a "minutes the
// witness will give you" count-down. Mapping is 1000 tokens =
// 1 minute (so a 50k budget = 50:00 starting). PRD §3.5: "47:00".
// Returns "—:—" when the budget is unbounded.
func (s *Session) ClockDisplay() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.budget.MaxOutputTokens <= 0 {
		return "—:—"
	}
	remaining := s.remainingOutputTokensLocked()
	mins := remaining / 1000
	secs := (remaining % 1000) * 60 / 1000
	return fmt.Sprintf("%d:%02d", mins, secs)
}

// budgetExhausted is the lock-free predicate; callers must hold
// s.mu (read or write).
func (s *Session) budgetExhausted() bool {
	if s.budget.MaxOutputTokens > 0 && s.usedOutputTokens >= s.budget.MaxOutputTokens {
		return true
	}
	if s.budget.MaxUSD > 0 && s.usedCostUSD >= s.budget.MaxUSD {
		return true
	}
	return false
}
