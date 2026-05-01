package game

import (
	"context"
	"fmt"
	"time"

	"github.com/jerkeyray/starling/event"
	"github.com/jerkeyray/starling/eventlog"
)

// VerifyResult is the outcome of walking a session's hash chain.
// The fictional reading: did the locked truth survive intact?
//
// In the fully-developed PRD, the "locked truth" is committed in a
// turn-0 event whose hash is shown to the player; the chain verifies
// that nothing has been re-written after the fact. Until M5 proper
// commits a TruthCommitted event, the verify routine just confirms
// every Run's chain validates and reports the hash and timestamp
// of the very first event in the file (typically RunStarted of
// the first ask) and the last event of the last run.
type VerifyResult struct {
	Path         string
	OK           bool
	Reason       string // populated on failure
	FirstSeq     uint64
	FirstHash    []byte // hash of the first event (PrevHash of the second event)
	FirstAt      time.Time
	LastSeq      uint64
	LastHash     []byte
	LastAt       time.Time
	RunCount     int
	EventCount   int
}

// Verify reads the SQLite log at savePath read-only, walks every
// run's events, and validates each chain. Returns OK=true with the
// summary when intact; OK=false + Reason when any run fails.
func Verify(ctx context.Context, savePath string) (VerifyResult, error) {
	if savePath == "" {
		return VerifyResult{}, fmt.Errorf("verify: no save path")
	}
	log, err := eventlog.NewSQLite(savePath, eventlog.WithReadOnly())
	if err != nil {
		return VerifyResult{}, fmt.Errorf("verify: open: %w", err)
	}
	defer log.Close()

	lister, ok := log.(eventlog.RunLister)
	if !ok {
		return VerifyResult{}, fmt.Errorf("verify: log does not implement RunLister")
	}
	runs, err := lister.ListRuns(ctx)
	if err != nil {
		return VerifyResult{}, fmt.Errorf("verify: list runs: %w", err)
	}

	result := VerifyResult{Path: savePath, OK: true, RunCount: len(runs)}
	if len(runs) == 0 {
		result.OK = false
		result.Reason = "no runs in file"
		return result, nil
	}

	var firstEv, lastEv event.Event
	firstSet := false

	for _, r := range runs {
		evs, err := log.Read(ctx, r.RunID)
		if err != nil {
			result.OK = false
			result.Reason = fmt.Sprintf("read %s: %v", r.RunID, err)
			return result, nil
		}
		result.EventCount += len(evs)

		// Validate the chain. Validate requires a terminal event;
		// in-progress runs fail it but their envelope chain is
		// still well-formed, so we run a lighter check on those.
		if len(evs) > 0 && evs[len(evs)-1].Kind.IsTerminal() {
			if err := eventlog.Validate(evs); err != nil {
				result.OK = false
				result.Reason = fmt.Sprintf("run %s: %v", r.RunID, err)
				return result, nil
			}
		} else {
			if err := validateEnvelopes(evs); err != nil {
				result.OK = false
				result.Reason = fmt.Sprintf("run %s: %v", r.RunID, err)
				return result, nil
			}
		}

		if !firstSet && len(evs) > 0 {
			firstEv = evs[0]
			firstSet = true
		}
		if len(evs) > 0 {
			lastEv = evs[len(evs)-1]
		}
	}

	if firstSet {
		result.FirstSeq = firstEv.Seq
		bs, _ := event.Marshal(firstEv)
		result.FirstHash = event.Hash(bs)
		result.FirstAt = time.Unix(0, firstEv.Timestamp)
	}
	if lastEv.Seq > 0 {
		result.LastSeq = lastEv.Seq
		bs, _ := event.Marshal(lastEv)
		result.LastHash = event.Hash(bs)
		result.LastAt = time.Unix(0, lastEv.Timestamp)
	}
	return result, nil
}

// validateEnvelopes is a slimmer chain check used on in-progress
// runs (no terminal event yet, so eventlog.Validate refuses them).
// Verifies seq monotonicity and prev-hash linkage; nothing else.
func validateEnvelopes(events []event.Event) error {
	if len(events) == 0 {
		return fmt.Errorf("no events")
	}
	if events[0].Seq != 1 {
		return fmt.Errorf("event[0].Seq = %d, want 1", events[0].Seq)
	}
	if len(events[0].PrevHash) != 0 {
		return fmt.Errorf("event[0].PrevHash non-empty")
	}
	for i := 1; i < len(events); i++ {
		prev := events[i-1]
		cur := events[i]
		if cur.Seq != prev.Seq+1 {
			return fmt.Errorf("seq gap at %d: %d → %d", i, prev.Seq, cur.Seq)
		}
		bs, _ := event.Marshal(prev)
		want := event.Hash(bs)
		if !bytesEqual(cur.PrevHash, want) {
			return fmt.Errorf("prev-hash mismatch at %d", i)
		}
	}
	return nil
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
