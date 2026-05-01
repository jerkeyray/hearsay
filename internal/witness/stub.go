package witness

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/jerkeyray/starling/event"
	"github.com/jerkeyray/starling/eventlog"
	"github.com/jerkeyray/starling/merkle"

	"github.com/jerkeyray/hearsay/internal/kase"
)

// StubDriver returns canned, deterministic lines indexed by
// (topic, technique). It exists so the loop is playable without an
// API key during development. The lines are intentionally dry per
// PRD §6 so the register feels right when comparing to LiveDriver.
//
// When constructed via NewStubDriverWithSave, the stub also writes a
// minimal hash-chained Starling event log per ask — enough to make
// the inspector, verify chain, and dev exploration work without any
// live LLM. The chain is "in progress" (no terminal RunCompleted)
// because the merkle-root code lives in starling/internal/, but
// envelope-level validation passes.
type StubDriver struct {
	log      eventlog.EventLog
	savePath string
}

// NewStubDriver constructs a StubDriver with no event log. Used by
// tests and when no save path is available.
func NewStubDriver() *StubDriver { return &StubDriver{} }

// NewStubDriverWithSave opens (or creates) a SQLite event log at
// savePath and returns a stub driver that writes an audit chain per
// ask. The driver owns the log; Close releases it.
func NewStubDriverWithSave(savePath string) (*StubDriver, error) {
	log, err := eventlog.NewSQLite(savePath)
	if err != nil {
		return nil, fmt.Errorf("stub: open event log: %w", err)
	}
	return &StubDriver{log: log, savePath: savePath}, nil
}

// stubOutputTokensPerAsk is a synthetic deduction so the session
// clock still ticks during dev runs without an API key. Calibrated
// so a 50k-token session takes ~250 stub asks to exhaust — comfortably
// more than any plausible playthrough.
const stubOutputTokensPerAsk = 200

// Respond returns the canned line for (topic, technique) plus a
// synthetic token charge so the session clock animates in stub mode
// and a per-technique demeanor so the portrait line renders. When
// the stub has an event log (NewStubDriverWithSave), each Respond
// also writes a 4-event Run (with a real merkle root in the
// terminal event) into the log so the inspector + verify path work
// in dev.
func (d *StubDriver) Respond(ctx context.Context, topic string, technique kase.Technique, _ []HistoryItem) (Response, error) {
	line := stubFallback[technique]
	if v, ok := stubLines[stubKey{topic, technique}]; ok {
		line = v
	}
	var runID string
	if d.log != nil {
		// We log writes best-effort: a stub eventlog hiccup should
		// not interrupt gameplay.
		runID, _ = d.writeStubRun(ctx, topic, technique, line)
	}
	return Response{
		Text:         line,
		Demeanor:     stubDemeanor[technique],
		RunID:        runID,
		OutputTokens: stubOutputTokensPerAsk,
	}, nil
}

// Branch forks the underlying SQLite log to dstPath using
// eventlog.ForkSQLite. With no log, returns a fresh in-memory stub.
func (d *StubDriver) Branch(dstPath, anchorRunID string) (Driver, error) {
	if d.log == nil {
		return NewStubDriver(), nil
	}
	if anchorRunID == "" {
		return nil, fmt.Errorf("stub branch: anchor run id required")
	}
	if err := eventlog.ForkSQLite(context.Background(), d.savePath, dstPath, anchorRunID, 0); err != nil {
		return nil, fmt.Errorf("stub branch: fork save: %w", err)
	}
	log, err := eventlog.NewSQLite(dstPath)
	if err != nil {
		return nil, fmt.Errorf("stub branch: open copy: %w", err)
	}
	return &StubDriver{log: log, savePath: dstPath}, nil
}

// Close releases the event log if any.
func (d *StubDriver) Close() error {
	if d.log == nil {
		return nil
	}
	err := d.log.Close()
	d.log = nil
	return err
}

// SavePathHint exposes the SQLite path so callers (the inspector
// panel, the verify modal) can re-open the log read-only. Returns
// "" when this stub has no log.
func (d *StubDriver) SavePathHint() string { return d.savePath }

// writeStubRun writes a 4-event Run (RunStarted + TurnStarted +
// AssistantMessageCompleted + RunCompleted) into d.log, with a fresh
// RunID and proper Seq/PrevHash linkage. The terminal event embeds a
// merkle.Root over the prior leaves so the chain fully validates —
// the inspector shows "chain valid" on stub-mode saves, same as
// live ones.
func (d *StubDriver) writeStubRun(ctx context.Context, topic string, technique kase.Technique, line string) (string, error) {
	runID := newStubULID()
	turnID := newStubULID()

	// Collect leaf hashes as we go so the terminal event can commit
	// to the chain via a real merkle root.
	leaves := make([][]byte, 0, 3)

	appendEv := func(ev event.Event) error {
		if err := d.log.Append(ctx, runID, ev); err != nil {
			return err
		}
		bs, err := event.Marshal(ev)
		if err != nil {
			return fmt.Errorf("stub: marshal seq %d: %w", ev.Seq, err)
		}
		leaves = append(leaves, event.Hash(bs))
		return nil
	}

	rs := event.RunStarted{
		SchemaVersion: event.SchemaVersion,
		Goal:          fmt.Sprintf("ask %q (%s)", topic, technique.Label()),
		ProviderID:    "stub",
		ModelID:       "stub",
	}
	rsEnc, err := event.EncodePayload(rs)
	if err != nil {
		return runID, err
	}
	ev1 := event.Event{
		RunID:     runID,
		Seq:       1,
		PrevHash:  nil,
		Timestamp: time.Now().UnixNano(),
		Kind:      event.KindRunStarted,
		Payload:   rsEnc,
	}
	if err := appendEv(ev1); err != nil {
		return runID, err
	}

	ts := event.TurnStarted{TurnID: turnID}
	tsEnc, err := event.EncodePayload(ts)
	if err != nil {
		return runID, err
	}
	ev2 := event.Event{
		RunID:     runID,
		Seq:       2,
		PrevHash:  leaves[0],
		Timestamp: time.Now().UnixNano(),
		Kind:      event.KindTurnStarted,
		Payload:   tsEnc,
	}
	if err := appendEv(ev2); err != nil {
		return runID, err
	}

	am := event.AssistantMessageCompleted{
		TurnID:       turnID,
		Text:         line,
		StopReason:   "stub",
		OutputTokens: stubOutputTokensPerAsk,
	}
	amEnc, err := event.EncodePayload(am)
	if err != nil {
		return runID, err
	}
	ev3 := event.Event{
		RunID:     runID,
		Seq:       3,
		PrevHash:  leaves[1],
		Timestamp: time.Now().UnixNano(),
		Kind:      event.KindAssistantMessageCompleted,
		Payload:   amEnc,
	}
	if err := appendEv(ev3); err != nil {
		return runID, err
	}

	// Terminal event with a real merkle root over the three prior
	// leaves. eventlog.Validate now accepts this chain.
	rc := event.RunCompleted{
		FinalText:         line,
		TurnCount:         1,
		ToolCallCount:     0,
		TotalOutputTokens: stubOutputTokensPerAsk,
		MerkleRoot:        merkle.Root(leaves),
	}
	rcEnc, err := event.EncodePayload(rc)
	if err != nil {
		return runID, err
	}
	ev4 := event.Event{
		RunID:     runID,
		Seq:       4,
		PrevHash:  leaves[2],
		Timestamp: time.Now().UnixNano(),
		Kind:      event.KindRunCompleted,
		Payload:   rcEnc,
	}
	return runID, d.log.Append(ctx, runID, ev4)
}

// newStubULID mints a fresh ULID for stub run / turn IDs. The
// LiveDriver path uses Starling's internal mint via Agent.Run; the
// stub runs without that, so it generates its own.
func newStubULID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}

// stubDemeanor maps each technique to a plausible witness state so
// the portrait line in the renderer animates in stub mode. Real
// demeanor signalling lives in the live driver via note_demeanor.
var stubDemeanor = map[kase.Technique]kase.Demeanor{
	kase.Directly:        kase.DemeanorEngaged,
	kase.MomentBefore:    kase.DemeanorUncomfortable,
	kase.HowDoYouKnow:    kase.DemeanorEngaged,
	kase.PushBack:        kase.DemeanorDefensive,
	kase.CircleBackLater: kase.DemeanorEngaged,
}

type stubKey struct {
	topic     string
	technique kase.Technique
}

// stubLines is the M1 canned matrix. Behavior here is not the spec
// for any MemoryKind; it is just enough to feel real before the
// LiveDriver lands. Real semantics arrive in M3 and are driven by the
// recall tool, not by hand-tuned strings.
var stubLines = map[stubKey]string{
	{"the car", kase.Directly}:               "she says it was red. she's sure.",
	{"the car", kase.HowDoYouKnow}:           `"the streetlight was orange. so it must have been red."`,
	{"the streetlight", kase.Directly}:       "orange. sodium. the kind that turns blood black.",
	{"the bag", kase.Directly}:               `"a folder. just a folder. i didn't really see."`,
	{"the bag", kase.MomentBefore}:           `she's quiet. "i think i heard something heavy."`,
	{"the time", kase.Directly}:              "11:47.",
	{"the limp", kase.Directly}:              "he was walking strangely. left side, maybe.",
	{"the second person", kase.Directly}:     `"a woman. in a coat. she was waiting for him."`,
	{"the second person", kase.HowDoYouKnow}: `"i saw her clearly."`,
}

var stubFallback = map[kase.Technique]string{
	kase.Directly:        "she pauses.",
	kase.MomentBefore:    "she looks somewhere else.",
	kase.HowDoYouKnow:    `"i just know."`,
	kase.PushBack:        "she holds her ground.",
	kase.CircleBackLater: "(noted.)",
}
