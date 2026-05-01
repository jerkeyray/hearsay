package witness_test

import (
	"testing"

	"github.com/jerkeyray/hearsay/cases/streetlight"
	"github.com/jerkeyray/hearsay/internal/kase"
	"github.com/jerkeyray/hearsay/internal/witness"
)

// beliefs is the canonical Case 1 belief map under test. Cases own
// this data now; the witness package only knows how to read it.
var beliefs = streetlight.Case.Beliefs

func TestRecall_RealStableAcrossTechniques(t *testing.T) {
	for _, tech := range []kase.Technique{kase.Directly, kase.PushBack} {
		got := witness.Recall(beliefs, "the streetlight", tech, 0)
		if got.Kind != "stable" {
			t.Errorf("technique %s: kind = %q, want stable", tech.Label(), got.Kind)
		}
		if got.Text == "" {
			t.Errorf("technique %s: text empty", tech.Label())
		}
	}
}

func TestRecall_RealHowDoYouKnowAttestsSensorySource(t *testing.T) {
	got := witness.Recall(beliefs, "the time", kase.HowDoYouKnow, 0)
	if got.AttestedSource == "" {
		t.Errorf("expected sensory source, got empty")
	}
}

func TestRecall_ConfabulatedDriftsAcrossSeeds(t *testing.T) {
	a := witness.Recall(beliefs, "the car", kase.Directly, 0)
	b := witness.Recall(beliefs, "the car", kase.Directly, 1)
	c := witness.Recall(beliefs, "the car", kase.Directly, 2)
	if a.Text == b.Text && b.Text == c.Text {
		t.Errorf("expected drift across seeds, all returned %q", a.Text)
	}
	for _, out := range []witness.RecallOutput{a, b, c} {
		if out.Kind != "drifting" {
			t.Errorf("kind = %q, want drifting", out.Kind)
		}
	}
}

func TestRecall_ConfabulatedHowDoYouKnowReturnsCircularSource(t *testing.T) {
	got := witness.Recall(beliefs, "the car", kase.HowDoYouKnow, 0)
	if got.AttestedSource == "" {
		t.Error("expected circular source")
	}
}

func TestRecall_ImplantedStableAcrossAsks(t *testing.T) {
	a := witness.Recall(beliefs, "the second person", kase.Directly, 0)
	b := witness.Recall(beliefs, "the second person", kase.Directly, 999)
	if a.Text != b.Text {
		t.Errorf("implanted text not stable: %q vs %q", a.Text, b.Text)
	}
	if a.Kind != "stable" {
		t.Errorf("kind = %q, want stable", a.Kind)
	}
}

func TestRecall_ImplantedHowDoYouKnowAttestsThinSource(t *testing.T) {
	got := witness.Recall(beliefs, "the second person", kase.HowDoYouKnow, 0)
	if got.Kind != "defended" {
		t.Errorf("kind = %q, want defended", got.Kind)
	}
	if got.AttestedSource == "" {
		t.Error("expected thin source")
	}
}

func TestRecall_SuppressedBouncesOnDirect(t *testing.T) {
	got := witness.Recall(beliefs, "the bag", kase.Directly, 0)
	if got.Kind != "bounced" {
		t.Errorf("kind = %q, want bounced", got.Kind)
	}
}

func TestRecall_SuppressedSurfacesUnderMomentBefore(t *testing.T) {
	got := witness.Recall(beliefs, "the bag", kase.MomentBefore, 0)
	if got.Kind == "bounced" {
		t.Errorf("the moment before should surface a gist; got bounced")
	}
	if got.Text == "" {
		t.Error("expected gist text")
	}
}

func TestRecall_UnknownTopicBounces(t *testing.T) {
	got := witness.Recall(beliefs, "the moon", kase.Directly, 0)
	if got.Kind != "bounced" {
		t.Errorf("kind = %q, want bounced", got.Kind)
	}
}

func TestRecallTool_HasName(t *testing.T) {
	tt := witness.RecallTool(beliefs)
	if tt.Name() != "recall" {
		t.Errorf("name = %q, want recall", tt.Name())
	}
	if tt.Description() == "" {
		t.Error("description empty")
	}
	if len(tt.Schema()) == 0 {
		t.Error("schema empty")
	}
}
