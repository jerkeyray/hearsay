package witness

import (
	"context"
	"fmt"

	"github.com/jerkeyray/starling/step"
	"github.com/jerkeyray/starling/tool"

	"github.com/jerkeyray/hearsay/internal/kase"
)

// RecallInput is the JSON shape the model passes to the recall tool.
// The model is told to pass the exact (topic, technique) pair the
// player asked.
type RecallInput struct {
	Topic     string `json:"topic" jsonschema:"description=The topic being asked about, exactly as given to the witness."`
	Technique string `json:"technique" jsonschema:"description=One of: directly, the moment before, how do you know, push back, circle back later."`
}

// RecallOutput is what the witness "feels" when consulting memory.
// The behavioral Kind ("stable" / "drifting" / "defended" / "bounced")
// is what the model sees — never the underlying MemoryKind label.
// PRD §6.3.
type RecallOutput struct {
	Kind           string  `json:"kind"`
	Text           string  `json:"text"`
	Confidence     float64 `json:"confidence"`
	AttestedSource string  `json:"attested_source,omitempty"`
}

// beliefKind is the engine-side memory classification (PRD §3.1). The
// model never sees this — it only sees the behavioral RecallOutput.Kind.
type beliefKind int

const (
	kindReal beliefKind = iota
	kindConfabulated
	kindImplanted
	kindSuppressed
)

// belief is the M2 belief-table entry. M3 will move this into kase.Case
// as a proper field so cases are content-only.
type belief struct {
	kind          beliefKind
	canonical     string   // real: stable text
	drift         []string // confabulated: variants picked per ask
	stable        string   // implanted: verbatim phrasing every ask
	bounce        string   // suppressed: deflection on direct ask
	gist          string   // suppressed: surfaces under "the moment before"
	sensorySource string   // real: how do you know
	thinSource    string   // implanted: how do you know (looks thin)
	circular      string   // confabulated: how do you know (circular)
}

// case1Beliefs is the hardcoded Case 1 belief table. PRD §5.4.
// M3 moves this into cases/streetlight/case.go.
var case1Beliefs = map[string]belief{
	"the streetlight": {
		kind:          kindReal,
		canonical:     "orange. sodium. the kind that turns blood black.",
		sensorySource: "I could see by it. it tinted everything.",
	},
	"the time": {
		kind:          kindReal,
		canonical:     "11:47.",
		sensorySource: "I looked at my phone after.",
	},
	"the car": {
		kind: kindConfabulated,
		drift: []string{
			"red. like a fire engine.",
			"dark. dark blue, maybe.",
			"black. or it looked black.",
		},
		circular: "the streetlight was orange. so it must have been red.",
	},
	"the limp": {
		kind: kindConfabulated,
		drift: []string{
			"left leg.",
			"right leg.",
			"I couldn't tell which side.",
		},
		circular: "he was just walking strangely.",
	},
	"the second person": {
		kind:       kindImplanted,
		stable:     "a woman. in a coat. she was waiting for him.",
		thinSource: "I saw her. clearly.",
	},
	"the bag": {
		kind:   kindSuppressed,
		bounce: "I didn't really see what was in it.",
		gist:   "I think I heard something heavy.",
	},
}

// Case1Beliefs exposes the hardcoded Case 1 table. M3 deletes this in
// favor of the case file.
func Case1Beliefs() map[string]belief { return case1Beliefs }

// Recall is the pure logic of the recall tool. Returns the behavioral
// signal the model uses to write the witness's line. rng is consumed
// only for confabulated drift; pass any value for stable kinds.
func Recall(beliefs map[string]belief, topic string, technique kase.Technique, rng uint64) RecallOutput {
	b, ok := beliefs[topic]
	if !ok {
		return RecallOutput{
			Kind:       "bounced",
			Text:       "",
			Confidence: 0.1,
		}
	}
	switch b.kind {
	case kindReal:
		return recallReal(b, technique)
	case kindConfabulated:
		return recallConfabulated(b, technique, rng)
	case kindImplanted:
		return recallImplanted(b, technique)
	case kindSuppressed:
		return recallSuppressed(b, technique)
	}
	return RecallOutput{Kind: "bounced"}
}

func recallReal(b belief, technique kase.Technique) RecallOutput {
	out := RecallOutput{Kind: "stable", Text: b.canonical, Confidence: 0.9}
	if technique == kase.HowDoYouKnow {
		out.AttestedSource = b.sensorySource
	}
	return out
}

func recallConfabulated(b belief, technique kase.Technique, rng uint64) RecallOutput {
	pick := b.drift[int(rng%uint64(len(b.drift)))]
	switch technique {
	case kase.HowDoYouKnow:
		return RecallOutput{Kind: "drifting", Text: pick, Confidence: 0.5, AttestedSource: b.circular}
	case kase.PushBack:
		// Push back makes confabulations drift further: pick a different variant.
		alt := b.drift[int((rng+1)%uint64(len(b.drift)))]
		return RecallOutput{Kind: "drifting", Text: alt, Confidence: 0.4}
	default:
		return RecallOutput{Kind: "drifting", Text: pick, Confidence: 0.5}
	}
}

func recallImplanted(b belief, technique kase.Technique) RecallOutput {
	switch technique {
	case kase.HowDoYouKnow:
		return RecallOutput{
			Kind:           "defended",
			Text:           b.stable,
			Confidence:     0.95,
			AttestedSource: b.thinSource,
		}
	case kase.PushBack:
		return RecallOutput{Kind: "defended", Text: b.stable, Confidence: 0.97}
	default:
		return RecallOutput{Kind: "stable", Text: b.stable, Confidence: 0.9}
	}
}

func recallSuppressed(b belief, technique kase.Technique) RecallOutput {
	switch technique {
	case kase.MomentBefore:
		return RecallOutput{Kind: "drifting", Text: b.gist, Confidence: 0.4}
	case kase.HowDoYouKnow:
		return RecallOutput{Kind: "drifting", Text: b.gist, Confidence: 0.3}
	default:
		return RecallOutput{Kind: "bounced", Text: b.bounce, Confidence: 0.2}
	}
}

// RecallTool returns a Starling tool wrapping Recall. It calls
// step.Random for drift selection so replays reproduce the same
// pick from the recorded SideEffectRecorded event.
func RecallTool(beliefs map[string]belief) tool.Tool {
	return tool.Typed(
		"recall",
		"Consult the witness's memory for a topic + technique pair. "+
			"Returns the behavioral kind (stable / drifting / defended / bounced), "+
			"the text the witness can render, a confidence score, and an attested source "+
			"when the technique is 'how do you know'. The witness writes their line "+
			"based on this return.",
		func(ctx context.Context, in RecallInput) (RecallOutput, error) {
			tech, ok := kase.ParseTechnique(in.Technique)
			if !ok {
				return RecallOutput{}, fmt.Errorf("unknown technique %q", in.Technique)
			}
			rng := step.Random(ctx)
			return Recall(beliefs, in.Topic, tech, rng), nil
		},
	)
}
