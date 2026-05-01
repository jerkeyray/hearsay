package game_test

import (
	"context"
	"testing"

	"github.com/jerkeyray/hearsay/internal/game"
	"github.com/jerkeyray/hearsay/internal/kase"
	"github.com/jerkeyray/hearsay/internal/witness"
)

// caseWithSurfacing is the minimal case that exercises the surfacing
// rule: visible "the bag" surfaces hidden "what happened before"
// when asked with "the moment before."
func caseWithSurfacing() kase.Case {
	return kase.Case{
		ID: "surf-test",
		Topics: []kase.Topic{
			{Name: "the bag", InitiallyVisible: true,
				Surfaces: []kase.SurfaceRule{
					{Topic: "what happened before", Technique: kase.MomentBefore},
				},
			},
			{Name: "what happened before", InitiallyVisible: false},
		},
	}
}

func TestSession_VisibleTopics_HidesByDefault(t *testing.T) {
	ctx := context.Background()
	s, err := game.NewSession(ctx, caseWithSurfacing(), witness.NewStubDriver(), game.Budget{MaxOutputTokens: 10_000})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	got := s.VisibleTopics()
	if len(got) != 1 || got[0].Name != "the bag" {
		t.Errorf("initial visible topics = %+v, want [the bag]", got)
	}
}

func TestSession_Surfacing_FiresOnMatchingTechnique(t *testing.T) {
	ctx := context.Background()
	s, err := game.NewSession(ctx, caseWithSurfacing(), witness.NewStubDriver(), game.Budget{MaxOutputTokens: 10_000})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if _, err := s.Ask(ctx, "the bag", kase.MomentBefore); err != nil {
		t.Fatalf("ask: %v", err)
	}
	got := s.VisibleTopics()
	names := []string{}
	for _, t := range got {
		names = append(names, t.Name)
	}
	if len(got) != 2 || got[0].Name != "the bag" || got[1].Name != "what happened before" {
		t.Errorf("after MomentBefore, visible = %v, want [the bag, what happened before]", names)
	}
}

func TestSession_Surfacing_DoesNotFireOnOtherTechnique(t *testing.T) {
	ctx := context.Background()
	s, err := game.NewSession(ctx, caseWithSurfacing(), witness.NewStubDriver(), game.Budget{MaxOutputTokens: 10_000})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if _, err := s.Ask(ctx, "the bag", kase.Directly); err != nil {
		t.Fatalf("ask: %v", err)
	}
	if got := len(s.VisibleTopics()); got != 1 {
		t.Errorf("after Directly, visible count = %d, want 1 (no surfacing)", got)
	}
}

func TestSession_Surfacing_Idempotent(t *testing.T) {
	ctx := context.Background()
	s, err := game.NewSession(ctx, caseWithSurfacing(), witness.NewStubDriver(), game.Budget{MaxOutputTokens: 10_000})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	for i := 0; i < 3; i++ {
		if _, err := s.Ask(ctx, "the bag", kase.MomentBefore); err != nil {
			t.Fatalf("ask %d: %v", i, err)
		}
	}
	if got := len(s.VisibleTopics()); got != 2 {
		t.Errorf("after 3 surfacing asks, visible count = %d, want 2 (no duplicates)", got)
	}
}

// TestSession_Surfacing_StreetlightCase1 exercises the real Case 1
// surfacing rule from cases/streetlight, verifying the case's
// declarative data wires through correctly.
func TestSession_Surfacing_StreetlightCase1(t *testing.T) {
	ctx := context.Background()
	// Inline the streetlight Case to avoid an import cycle in tests.
	c := kase.Case{
		ID: "test",
		Topics: []kase.Topic{
			{Name: "the bag", InitiallyVisible: true,
				Surfaces: []kase.SurfaceRule{
					{Topic: "what happened before", Technique: kase.MomentBefore},
				},
			},
			{Name: "what happened before", InitiallyVisible: false},
		},
	}
	s, err := game.NewSession(ctx, c, witness.NewStubDriver(), game.Budget{MaxOutputTokens: 10_000})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if _, err := s.Ask(ctx, "the bag", kase.MomentBefore); err != nil {
		t.Fatalf("ask: %v", err)
	}
	visible := s.VisibleTopics()
	found := false
	for _, t := range visible {
		if t.Name == "what happened before" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("'what happened before' not surfaced after MomentBefore on 'the bag'")
	}
}
