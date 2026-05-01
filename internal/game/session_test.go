package game_test

import (
	"context"
	"testing"

	"github.com/jerkeyray/hearsay/internal/game"
	"github.com/jerkeyray/hearsay/internal/kase"
	"github.com/jerkeyray/hearsay/internal/witness"
)

// TestSession_StubRoundTrip exercises the engine wiring against the
// stub driver: three asks, expected exchanges in the in-memory log,
// Close idempotent. The SQLite-event-log round trip lives in the
// LiveDriver tests (step 12b) since the stub driver does not write
// events.
func TestSession_StubRoundTrip(t *testing.T) {
	c := kase.Case{
		ID: "test",
		Topics: []kase.Topic{
			{Name: "the bag", InitiallyVisible: true},
			{Name: "the streetlight", InitiallyVisible: true},
		},
	}

	ctx := context.Background()
	d := witness.NewStubDriver()
	s, err := game.NewSession(ctx, c, d, game.Budget{MaxOutputTokens: 1000})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	asks := []struct {
		topic string
		tech  kase.Technique
	}{
		{"the bag", kase.Directly},
		{"the streetlight", kase.Directly},
		{"the bag", kase.MomentBefore},
	}
	for i, a := range asks {
		ex, err := s.Ask(ctx, a.topic, a.tech)
		if err != nil {
			t.Fatalf("Ask %d: %v", i, err)
		}
		if ex.Turn != i {
			t.Errorf("ask %d: Turn = %d, want %d", i, ex.Turn, i)
		}
		if ex.Topic != a.topic {
			t.Errorf("ask %d: Topic = %q, want %q", i, ex.Topic, a.topic)
		}
		if ex.Witness == "" {
			t.Errorf("ask %d: Witness empty", i)
		}
	}

	if got := s.TurnCount(); got != 3 {
		t.Errorf("TurnCount = %d, want 3", got)
	}
	if got := len(s.Log()); got != 3 {
		t.Errorf("Log length = %d, want 3", got)
	}

	// Close is idempotent.
	if err := s.Close(ctx); err != nil {
		t.Errorf("Close: %v", err)
	}
	if err := s.Close(ctx); err != nil {
		t.Errorf("second Close: %v", err)
	}
}
