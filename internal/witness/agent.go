// Package witness drives the witness's responses. In M1 this is a
// hardcoded lookup table; in M2 it is replaced with a real
// starling.Agent.Run wired to an LLM provider. The Respond signature
// is the seam — callers do not change.
package witness

import "github.com/jerkeyray/hearsay/internal/kase"

// Agent generates a witness line for a (topic, technique) pair. M1
// stub implementation; the real LLM driver lands in M2.
type Agent struct{}

func New() *Agent { return &Agent{} }

// Respond returns what the witness says when asked about topic with
// the given technique. M1: canned, deterministic, indexed by both.
// The line is intentionally dry per PRD §6.
func (a *Agent) Respond(topic string, technique kase.Technique) string {
	if line, ok := stubLines[stubKey{topic, technique}]; ok {
		return line
	}
	return stubFallback[technique]
}

type stubKey struct {
	topic     string
	technique kase.Technique
}

// stubLines is the M1 canned matrix. It exists to make the topic +
// technique loop feel real before the LLM lands; behavior here is not
// the spec for any MemoryKind. Real semantics arrive in M3.
var stubLines = map[stubKey]string{
	{"the car", kase.Directly}:        "she says it was red. she's sure.",
	{"the car", kase.HowDoYouKnow}:    `"the streetlight was orange. so it must have been red."`,
	{"the streetlight", kase.Directly}: "orange. sodium. the kind that turns blood black.",
	{"the bag", kase.Directly}:         `"a folder. just a folder. i didn't really see."`,
	{"the bag", kase.MomentBefore}:     `she's quiet. "i think i heard something heavy."`,
	{"the time", kase.Directly}:        "11:47.",
	{"the limp", kase.Directly}:        "he was walking strangely. left side, maybe.",
	{"the second person", kase.Directly}: `"a woman. in a coat. she was waiting for him."`,
	{"the second person", kase.HowDoYouKnow}: `"i saw her clearly."`,
}

var stubFallback = map[kase.Technique]string{
	kase.Directly:        "she pauses.",
	kase.MomentBefore:    "she looks somewhere else.",
	kase.HowDoYouKnow:    `"i just know."`,
	kase.PushBack:        "she holds her ground.",
	kase.CircleBackLater: "(noted.)",
}
