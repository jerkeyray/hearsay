package game_test

import (
	"context"
	"testing"

	"github.com/jerkeyray/hearsay/internal/game"
	"github.com/jerkeyray/hearsay/internal/kase"
	"github.com/jerkeyray/hearsay/internal/witness"
)

func rewindCase() kase.Case {
	return kase.Case{
		ID: "rewind-test",
		Topics: []kase.Topic{
			{Name: "the bag", InitiallyVisible: true,
				Surfaces: []kase.SurfaceRule{
					{Topic: "what happened before", Technique: kase.MomentBefore},
				},
			},
			{Name: "the car", InitiallyVisible: true},
			{Name: "what happened before", InitiallyVisible: false},
		},
	}
}

func TestSession_RewindTruncatesLog(t *testing.T) {
	ctx := context.Background()
	s, err := game.NewSession(ctx, rewindCase(), witness.NewStubDriver(), game.Budget{MaxOutputTokens: 10_000})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	for i := 0; i < 4; i++ {
		if _, err := s.Ask(ctx, "the bag", kase.Directly); err != nil {
			t.Fatalf("ask %d: %v", i, err)
		}
	}
	if err := s.RewindTo(1); err != nil {
		t.Fatalf("RewindTo(1): %v", err)
	}
	if got := s.TurnCount(); got != 2 {
		t.Errorf("TurnCount after RewindTo(1) = %d, want 2", got)
	}
}

func TestSession_RewindRecomputesBudget(t *testing.T) {
	ctx := context.Background()
	s, err := game.NewSession(ctx, rewindCase(), witness.NewStubDriver(), game.Budget{MaxOutputTokens: 10_000})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	for i := 0; i < 5; i++ {
		if _, err := s.Ask(ctx, "the car", kase.Directly); err != nil {
			t.Fatalf("ask %d: %v", i, err)
		}
	}
	// Stub charges 200 per ask → 1000 used.
	if got := s.UsedOutputTokens(); got != 1000 {
		t.Fatalf("UsedOutputTokens = %d, want 1000", got)
	}
	if err := s.RewindTo(1); err != nil {
		t.Fatalf("RewindTo: %v", err)
	}
	// 2 surviving asks → 400 used.
	if got := s.UsedOutputTokens(); got != 400 {
		t.Errorf("UsedOutputTokens after rewind = %d, want 400", got)
	}
}

func TestSession_RewindRebuildsSurfacing(t *testing.T) {
	ctx := context.Background()
	s, err := game.NewSession(ctx, rewindCase(), witness.NewStubDriver(), game.Budget{MaxOutputTokens: 10_000})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	// Surface "what happened before" via MomentBefore on "the bag".
	if _, err := s.Ask(ctx, "the bag", kase.MomentBefore); err != nil {
		t.Fatalf("ask: %v", err)
	}
	// Then ask another turn.
	if _, err := s.Ask(ctx, "the car", kase.Directly); err != nil {
		t.Fatalf("ask: %v", err)
	}
	if got := len(s.VisibleTopics()); got != 3 {
		t.Fatalf("visible after surfacing = %d, want 3", got)
	}
	// Rewind before the surfacing turn.
	if err := s.RewindTo(-1); err != nil {
		t.Fatalf("RewindTo(-1): %v", err)
	}
	if got := len(s.VisibleTopics()); got != 2 {
		t.Errorf("visible after rewind to -1 = %d, want 2 (surfacing undone)", got)
	}
	for _, top := range s.VisibleTopics() {
		if top.Name == "what happened before" {
			t.Errorf("hidden topic still visible after rewind")
		}
	}
}

func TestSession_RewindClearsEndedFlag(t *testing.T) {
	ctx := context.Background()
	s, err := game.NewSession(ctx, rewindCase(), witness.NewStubDriver(), game.Budget{MaxOutputTokens: 400})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	for i := 0; i < 2; i++ {
		if _, err := s.Ask(ctx, "the car", kase.Directly); err != nil {
			t.Fatalf("ask %d: %v", i, err)
		}
	}
	if !s.IsEnded() {
		t.Fatalf("expected session ended after budget exhaustion")
	}
	if err := s.RewindTo(0); err != nil {
		t.Fatalf("RewindTo: %v", err)
	}
	if s.IsEnded() {
		t.Errorf("session still ended after rewind")
	}
	// Should be able to Ask again.
	if _, err := s.Ask(ctx, "the car", kase.Directly); err != nil {
		t.Errorf("Ask after rewind: %v", err)
	}
}

func TestSession_RewindOutOfRange(t *testing.T) {
	ctx := context.Background()
	s, err := game.NewSession(ctx, rewindCase(), witness.NewStubDriver(), game.Budget{MaxOutputTokens: 10_000})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if _, err := s.Ask(ctx, "the car", kase.Directly); err != nil {
		t.Fatalf("ask: %v", err)
	}
	if err := s.RewindTo(5); err == nil {
		t.Errorf("expected out-of-range error")
	}
	if err := s.RewindTo(-2); err == nil {
		t.Errorf("expected out-of-range error for -2")
	}
}

func TestSession_RewindToMinusOneEmpties(t *testing.T) {
	ctx := context.Background()
	s, err := game.NewSession(ctx, rewindCase(), witness.NewStubDriver(), game.Budget{MaxOutputTokens: 10_000})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	for i := 0; i < 3; i++ {
		if _, err := s.Ask(ctx, "the car", kase.Directly); err != nil {
			t.Fatalf("ask %d: %v", i, err)
		}
	}
	if err := s.RewindTo(-1); err != nil {
		t.Fatalf("RewindTo(-1): %v", err)
	}
	if got := s.TurnCount(); got != 0 {
		t.Errorf("TurnCount after RewindTo(-1) = %d, want 0", got)
	}
	if got := s.UsedOutputTokens(); got != 0 {
		t.Errorf("UsedOutputTokens after RewindTo(-1) = %d, want 0", got)
	}
}
