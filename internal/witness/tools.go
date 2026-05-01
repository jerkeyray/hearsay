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

// Recall is the pure logic of the recall tool. Returns the behavioral
// signal the model uses to write the witness's line. rng is consumed
// only for confabulated drift; pass any value for stable kinds.
func Recall(beliefs map[string]kase.Belief, topic string, technique kase.Technique, rng uint64) RecallOutput {
	b, ok := beliefs[topic]
	if !ok {
		return RecallOutput{
			Kind:       "bounced",
			Text:       "",
			Confidence: 0.1,
		}
	}
	switch b.Kind {
	case kase.Real:
		return recallReal(b, technique)
	case kase.Confabulated:
		return recallConfabulated(b, technique, rng)
	case kase.Implanted:
		return recallImplanted(b, technique)
	case kase.Suppressed:
		return recallSuppressed(b, technique)
	}
	return RecallOutput{Kind: "bounced"}
}

func recallReal(b kase.Belief, technique kase.Technique) RecallOutput {
	out := RecallOutput{Kind: "stable", Text: b.Canonical, Confidence: 0.9}
	if technique == kase.HowDoYouKnow {
		out.AttestedSource = b.SensorySource
	}
	return out
}

func recallConfabulated(b kase.Belief, technique kase.Technique, rng uint64) RecallOutput {
	if len(b.Drift) == 0 {
		return RecallOutput{Kind: "drifting", Text: "", Confidence: 0.3}
	}
	pick := b.Drift[int(rng%uint64(len(b.Drift)))]
	switch technique {
	case kase.HowDoYouKnow:
		return RecallOutput{Kind: "drifting", Text: pick, Confidence: 0.5, AttestedSource: b.Circular}
	case kase.PushBack:
		// Push back makes confabulations drift further: pick a different variant.
		alt := b.Drift[int((rng+1)%uint64(len(b.Drift)))]
		return RecallOutput{Kind: "drifting", Text: alt, Confidence: 0.4}
	default:
		return RecallOutput{Kind: "drifting", Text: pick, Confidence: 0.5}
	}
}

func recallImplanted(b kase.Belief, technique kase.Technique) RecallOutput {
	switch technique {
	case kase.HowDoYouKnow:
		return RecallOutput{
			Kind:           "defended",
			Text:           b.Stable,
			Confidence:     0.95,
			AttestedSource: b.ThinSource,
		}
	case kase.PushBack:
		return RecallOutput{Kind: "defended", Text: b.Stable, Confidence: 0.97}
	default:
		return RecallOutput{Kind: "stable", Text: b.Stable, Confidence: 0.9}
	}
}

func recallSuppressed(b kase.Belief, technique kase.Technique) RecallOutput {
	switch technique {
	case kase.MomentBefore:
		return RecallOutput{Kind: "drifting", Text: b.Gist, Confidence: 0.4}
	case kase.HowDoYouKnow:
		return RecallOutput{Kind: "drifting", Text: b.Gist, Confidence: 0.3}
	default:
		return RecallOutput{Kind: "bounced", Text: b.Bounce, Confidence: 0.2}
	}
}

// DemeanorInput is the JSON shape passed to note_demeanor. The model
// picks one of: engaged, uncomfortable, defensive, tired.
type DemeanorInput struct {
	State string `json:"state" jsonschema:"description=Witness demeanor — one of: engaged, uncomfortable, defensive, tired."`
}

// DemeanorOutput is a trivial ack so the model knows the signal
// landed.
type DemeanorOutput struct {
	Ack bool `json:"ack"`
}

// DemeanorTool returns the note_demeanor tool. set is invoked
// synchronously inside the tool call with the parsed Demeanor;
// callers typically wire it to a per-ask sink they read after Run.
func DemeanorTool(set func(kase.Demeanor)) tool.Tool {
	return tool.Typed(
		"note_demeanor",
		"Signal the witness's current demeanor. Call this when the "+
			"witness's visible state shifts during your line — when the "+
			"player's question has destabilized, defended, fatigued, or "+
			"steadied them. State must be one of: engaged, uncomfortable, "+
			"defensive, tired. The renderer uses this to update the "+
			"witness portrait; you cannot describe demeanor in your line.",
		func(_ context.Context, in DemeanorInput) (DemeanorOutput, error) {
			d, ok := kase.ParseDemeanor(in.State)
			if !ok {
				return DemeanorOutput{}, fmt.Errorf("unknown demeanor %q", in.State)
			}
			set(d)
			return DemeanorOutput{Ack: true}, nil
		},
	)
}

// RecallTool returns a Starling tool wrapping Recall. It calls
// step.Random for drift selection so replays reproduce the same
// pick from the recorded SideEffectRecorded event.
func RecallTool(beliefs map[string]kase.Belief) tool.Tool {
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
