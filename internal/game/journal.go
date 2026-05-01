package game

import (
	"context"
	"fmt"
	"time"

	"github.com/jerkeyray/starling/event"
	"github.com/jerkeyray/starling/eventlog"
)

// Journal is a thin wrapper over a Starling EventLog that maintains
// the per-run hash chain (Seq, PrevHash, Timestamp) so callers only
// supply Kind + typed payload. One Journal per session save file.
//
// The journal is intentionally minimal: M2 will replace direct
// Append calls with a real starling.Agent.Run that owns the chain
// internally. For M1 with a stub witness, going manual lets us
// produce a real, hash-chained, replay-ready audit log without a
// live LLM provider.
type Journal struct {
	log      eventlog.EventLog
	runID    string
	seq      uint64
	prevHash []byte
}

// OpenJournal opens (or creates) a SQLite event log at path and binds
// it to runID. Caller must Close when the session ends.
func OpenJournal(ctx context.Context, path, runID string) (*Journal, error) {
	log, err := eventlog.NewSQLite(path)
	if err != nil {
		return nil, fmt.Errorf("open journal: %w", err)
	}
	j := &Journal{log: log, runID: runID}
	// Resume support: if the log already has events for this runID,
	// replay state so subsequent Appends chain correctly.
	prior, err := log.Read(ctx, runID)
	if err != nil {
		log.Close()
		return nil, fmt.Errorf("read journal: %w", err)
	}
	if len(prior) > 0 {
		last := prior[len(prior)-1]
		j.seq = last.Seq
		marshaled, err := event.Marshal(last)
		if err != nil {
			log.Close()
			return nil, fmt.Errorf("rehash journal tail: %w", err)
		}
		j.prevHash = event.Hash(marshaled)
	}
	return j, nil
}

// RunID is the run identifier this journal writes under.
func (j *Journal) RunID() string { return j.runID }

// Append builds an event for the given kind and payload, computes
// Seq/PrevHash/Timestamp, and appends it to the underlying log.
func appendPayload[T any](ctx context.Context, j *Journal, kind event.Kind, payload T) error {
	enc, err := event.EncodePayload(payload)
	if err != nil {
		return fmt.Errorf("encode %s: %w", kind, err)
	}
	ev := event.Event{
		RunID:     j.runID,
		Seq:       j.seq + 1,
		PrevHash:  j.prevHash,
		Timestamp: time.Now().UnixNano(),
		Kind:      kind,
		Payload:   enc,
	}
	if err := j.log.Append(ctx, j.runID, ev); err != nil {
		return fmt.Errorf("append %s: %w", kind, err)
	}
	marshaled, err := event.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshal %s for chain: %w", kind, err)
	}
	j.seq = ev.Seq
	j.prevHash = event.Hash(marshaled)
	return nil
}

// AppendRunStarted writes the M1 RunStarted with stub provider/model
// fields. M2 swaps in real provider IDs and prompt hashes.
func (j *Journal) AppendRunStarted(ctx context.Context, caseID string) error {
	return appendPayload(ctx, j, event.KindRunStarted, event.RunStarted{
		SchemaVersion: event.SchemaVersion,
		Goal:          "hearsay:" + caseID,
		ProviderID:    "stub",
		ModelID:       "stub",
	})
}

// AppendTurnStarted records the start of one player turn.
func (j *Journal) AppendTurnStarted(ctx context.Context, turnID string) error {
	return appendPayload(ctx, j, event.KindTurnStarted, event.TurnStarted{
		TurnID: turnID,
	})
}

// AppendAssistantMessage records the witness's reply text. In M1 the
// token counts and cost are zero; M2 fills them in from real provider
// usage.
func (j *Journal) AppendAssistantMessage(ctx context.Context, turnID, text string) error {
	return appendPayload(ctx, j, event.KindAssistantMessageCompleted, event.AssistantMessageCompleted{
		TurnID:     turnID,
		Text:       text,
		StopReason: "stub",
	})
}

// Close releases the underlying event log. M1 does not write a
// terminal RunCompleted: that event must carry a Merkle root over the
// pre-terminal chain, and starling/internal/merkle is not exported.
// M2 will replace this manual journal with starling.Agent.Run, which
// owns terminal events natively.
func (j *Journal) Close() error { return j.log.Close() }
