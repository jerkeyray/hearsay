package witness

import (
	"context"

	"github.com/jerkeyray/hearsay/internal/kase"
)

// Driver is the witness's voice. Implementations produce one line of
// dialogue per ask, given the topic, technique, and conversation
// history so far. M1/M2 ship two impls:
//
//   - StubDriver: canned lines indexed by (topic, technique). No LLM,
//     no event log writes. Used for development without an API key.
//   - LiveDriver: a real starling.Agent run per ask, writing events
//     to a shared SQLite log. Lands in step 12b.
//
// Drivers may hold provider connections, event-log handles, or
// other resources; callers must Close when the session ends.
type Driver interface {
	Respond(ctx context.Context, topic string, technique kase.Technique, history []HistoryItem) (string, error)
	Close() error
}
