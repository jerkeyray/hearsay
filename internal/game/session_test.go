package game_test

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/jerkeyray/starling/event"
	"github.com/jerkeyray/starling/eventlog"

	"github.com/jerkeyray/hearsay/internal/game"
	"github.com/jerkeyray/hearsay/internal/kase"
	"github.com/jerkeyray/hearsay/internal/witness"
)

// TestSession_RoundTrip starts a session, runs three asks, closes it,
// then re-opens the SQLite log and confirms the chain validates and
// the expected event kinds are present in order. Exercises the M1 wiring
// end-to-end without the TUI.
func TestSession_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HEARSAY_HOME", tmp)

	c := kase.Case{
		ID: "test",
		Topics: []kase.Topic{
			{Name: "the bag", InitiallyVisible: true},
		},
	}

	ctx := context.Background()
	s, err := game.NewSession(ctx, c, witness.New())
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	for i := 0; i < 3; i++ {
		if _, err := s.Ask(ctx, "the bag", kase.Directly); err != nil {
			t.Fatalf("Ask %d: %v", i, err)
		}
	}
	if err := s.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Re-open the SQLite log and read the chain back.
	path := s.SavePath()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("save file missing at %s: %v", path, err)
	}
	log, err := eventlog.NewSQLite(path, eventlog.WithReadOnly())
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	defer log.Close()

	events, err := log.Read(ctx, s.RunID)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	// M1 chains in-progress runs (no RunCompleted yet — see journal.go).
	// Validate seq monotonicity and prev-hash linkage by hand; the full
	// merkle-root validation arrives once M2's Agent.Run owns terminal
	// events.
	if events[0].Seq != 1 {
		t.Errorf("event[0].Seq = %d, want 1", events[0].Seq)
	}
	if len(events[0].PrevHash) != 0 {
		t.Errorf("event[0].PrevHash should be empty")
	}
	for i := 1; i < len(events); i++ {
		if events[i].Seq != events[i-1].Seq+1 {
			t.Errorf("seq gap at %d: %d → %d", i, events[i-1].Seq, events[i].Seq)
		}
		prevBytes, err := event.Marshal(events[i-1])
		if err != nil {
			t.Fatalf("marshal event[%d]: %v", i-1, err)
		}
		want := event.Hash(prevBytes)
		if !bytes.Equal(events[i].PrevHash, want) {
			t.Errorf("prev-hash mismatch at %d", i)
		}
	}

	want := []event.Kind{
		event.KindRunStarted,
		event.KindTurnStarted, event.KindAssistantMessageCompleted,
		event.KindTurnStarted, event.KindAssistantMessageCompleted,
		event.KindTurnStarted, event.KindAssistantMessageCompleted,
	}
	if len(events) != len(want) {
		t.Fatalf("event count: got %d want %d", len(events), len(want))
	}
	for i, ev := range events {
		if ev.Kind != want[i] {
			t.Errorf("event[%d]: got %s want %s", i, ev.Kind, want[i])
		}
	}
}
