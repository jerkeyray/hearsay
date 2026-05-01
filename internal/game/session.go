// Package game owns the engine-side session state: the current case,
// the conversation log, and the witness driver that produces each
// turn. The driver is also the seam where the live LLM (M2.b) attaches
// and where Starling event-log writes happen for live runs.
//
// The UI holds cursor state and renders; the engine owns everything
// that would survive a process restart.
package game

import (
	"context"

	"github.com/jerkeyray/hearsay/internal/kase"
	"github.com/jerkeyray/hearsay/internal/witness"
)

// Exchange is one turn in the conversation: what the player asked and
// what the witness said back.
type Exchange struct {
	Turn      int
	Topic     string
	Technique kase.Technique
	Witness   string
}

// Session is the per-case engine state. One Session per save file.
type Session struct {
	Case   kase.Case
	driver witness.Driver
	log    []Exchange
}

// NewSession constructs a session over a case + witness driver. The
// session does not own the driver's resources — it relies on Close to
// release them.
func NewSession(_ context.Context, c kase.Case, d witness.Driver) (*Session, error) {
	return &Session{Case: c, driver: d}, nil
}

// Ask runs one turn: build the conversation history, ask the driver,
// append the exchange. In the live path the driver writes Starling
// events to its event log under a per-ask RunID; in the stub path the
// driver is a pure function.
func (s *Session) Ask(ctx context.Context, topic string, technique kase.Technique) (Exchange, error) {
	hist := make([]witness.HistoryItem, 0, len(s.log))
	for _, ex := range s.log {
		hist = append(hist, witness.HistoryItem{
			Topic:     ex.Topic,
			Technique: ex.Technique,
			Witness:   ex.Witness,
		})
	}
	line, err := s.driver.Respond(ctx, topic, technique, hist)
	if err != nil {
		return Exchange{}, err
	}
	ex := Exchange{
		Turn:      len(s.log),
		Topic:     topic,
		Technique: technique,
		Witness:   line,
	}
	s.log = append(s.log, ex)
	return ex, nil
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

// Log returns the exchanges in order. Read-only view for renderers.
func (s *Session) Log() []Exchange { return s.log }

// TurnCount is the number of completed turns.
func (s *Session) TurnCount() int { return len(s.log) }
