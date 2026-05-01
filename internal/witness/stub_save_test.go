package witness_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jerkeyray/starling/event"
	"github.com/jerkeyray/starling/eventlog"

	"github.com/jerkeyray/hearsay/internal/kase"
	"github.com/jerkeyray/hearsay/internal/witness"
)

// TestStubDriver_WritesSQLiteEvents confirms the stub driver writes
// a 3-event Run per ask when constructed with a save path. Three
// asks → three runs, each with [RunStarted, TurnStarted,
// AssistantMessageCompleted] in order, properly chained.
func TestStubDriver_WritesSQLiteEvents(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "stub.db")

	d, err := witness.NewStubDriverWithSave(path)
	if err != nil {
		t.Fatalf("NewStubDriverWithSave: %v", err)
	}
	for i := 0; i < 3; i++ {
		if _, err := d.Respond(context.Background(), "the bag", kase.Directly, nil); err != nil {
			t.Fatalf("respond %d: %v", i, err)
		}
	}
	if err := d.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	log, err := eventlog.NewSQLite(path, eventlog.WithReadOnly())
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	defer log.Close()

	lister, ok := log.(eventlog.RunLister)
	if !ok {
		t.Fatal("not a RunLister")
	}
	runs, err := lister.ListRuns(context.Background())
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(runs) != 3 {
		t.Fatalf("run count = %d, want 3", len(runs))
	}
	wantKinds := []event.Kind{
		event.KindRunStarted,
		event.KindTurnStarted,
		event.KindAssistantMessageCompleted,
		event.KindRunCompleted,
	}
	for _, r := range runs {
		evs, err := log.Read(context.Background(), r.RunID)
		if err != nil {
			t.Fatalf("Read %s: %v", r.RunID, err)
		}
		if len(evs) != 4 {
			t.Errorf("run %s: %d events, want 4", r.RunID, len(evs))
			continue
		}
		for i, ev := range evs {
			if ev.Kind != wantKinds[i] {
				t.Errorf("run %s seq %d: kind %s, want %s", r.RunID, ev.Seq, ev.Kind, wantKinds[i])
			}
		}
		// With a real merkle root the chain fully validates now.
		if err := eventlog.Validate(evs); err != nil {
			t.Errorf("run %s: chain validate: %v", r.RunID, err)
		}
	}
}
