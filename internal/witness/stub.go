package witness

import (
	"context"

	"github.com/jerkeyray/hearsay/internal/kase"
)

// StubDriver returns canned, deterministic lines indexed by
// (topic, technique). It does not call any LLM and does not write
// to an event log; it exists so the loop is playable without an
// API key during development. The lines are intentionally dry per
// PRD §6 so the register feels right when comparing to LiveDriver.
type StubDriver struct{}

// NewStubDriver constructs a StubDriver.
func NewStubDriver() *StubDriver { return &StubDriver{} }

// stubOutputTokensPerAsk is a synthetic deduction so the session
// clock still ticks during dev runs without an API key. Calibrated
// so a 50k-token session takes ~250 stub asks to exhaust — comfortably
// more than any plausible playthrough.
const stubOutputTokensPerAsk = 200

// Respond returns the canned line for (topic, technique) plus a
// synthetic token charge so the session clock animates in stub mode
// and a per-technique demeanor so the portrait line renders. The
// history argument is accepted for interface parity but ignored.
func (d *StubDriver) Respond(_ context.Context, topic string, technique kase.Technique, _ []HistoryItem) (Response, error) {
	line := stubFallback[technique]
	if v, ok := stubLines[stubKey{topic, technique}]; ok {
		line = v
	}
	return Response{
		Text:         line,
		Demeanor:     stubDemeanor[technique],
		OutputTokens: stubOutputTokensPerAsk,
	}, nil
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

// Branch returns a fresh StubDriver — there's no per-session state
// to copy.
func (d *StubDriver) Branch(_ string) (Driver, error) { return NewStubDriver(), nil }

// Close is a no-op for StubDriver.
func (d *StubDriver) Close() error { return nil }

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
