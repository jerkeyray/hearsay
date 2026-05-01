package witness

import (
	"context"

	"github.com/jerkeyray/hearsay/internal/kase"
)

// Driver is the witness's voice. Implementations produce one line of
// dialogue per ask, given the topic, technique, and conversation
// history so far. The Response carries the text plus the token /
// cost usage of this turn so Session can accumulate session-level
// budget consumption.
//
// Two impls:
//
//   - StubDriver: canned lines indexed by (topic, technique). No LLM,
//     no event log writes. Used for development without an API key.
//   - LiveDriver: a real starling.Agent run per ask, writing events
//     to a shared SQLite log.
//
// Drivers may hold provider connections, event-log handles, or
// other resources; callers must Close when the session ends.
type Driver interface {
	Respond(ctx context.Context, topic string, technique kase.Technique, history []HistoryItem) (Response, error)
	Close() error
}

// Response is one turn's output: the witness's line, the demeanor
// the model signalled (if any), plus the usage numbers Session uses
// to drive the session clock.
type Response struct {
	Text         string
	Demeanor     kase.Demeanor
	InputTokens  int64
	OutputTokens int64
	CostUSD      float64
}
