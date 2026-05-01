package game_test

import (
	"context"
	"testing"

	"github.com/jerkeyray/hearsay/internal/game"
	"github.com/jerkeyray/hearsay/internal/kase"
	"github.com/jerkeyray/hearsay/internal/witness"
)

func TestSession_CurrentDemeanor_DefaultsToEngaged(t *testing.T) {
	ctx := context.Background()
	c := kase.Case{ID: "test", Topics: []kase.Topic{{Name: "x", InitiallyVisible: true}}}
	s, err := game.NewSession(ctx, c, witness.NewStubDriver(), game.Budget{MaxOutputTokens: 10_000})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if got := s.CurrentDemeanor(); got != kase.DemeanorEngaged {
		t.Errorf("CurrentDemeanor on fresh session = %q, want engaged", got)
	}
}

func TestSession_CurrentDemeanor_TracksLatestExchange(t *testing.T) {
	ctx := context.Background()
	c := kase.Case{ID: "test", Topics: []kase.Topic{{Name: "x", InitiallyVisible: true}}}
	s, err := game.NewSession(ctx, c, witness.NewStubDriver(), game.Budget{MaxOutputTokens: 10_000})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	// Stub maps PushBack → defensive.
	if _, err := s.Ask(ctx, "x", kase.PushBack); err != nil {
		t.Fatalf("ask: %v", err)
	}
	if got := s.CurrentDemeanor(); got != kase.DemeanorDefensive {
		t.Errorf("after PushBack, CurrentDemeanor = %q, want defensive", got)
	}
	// MomentBefore → uncomfortable.
	if _, err := s.Ask(ctx, "x", kase.MomentBefore); err != nil {
		t.Fatalf("ask: %v", err)
	}
	if got := s.CurrentDemeanor(); got != kase.DemeanorUncomfortable {
		t.Errorf("after MomentBefore, CurrentDemeanor = %q, want uncomfortable", got)
	}
}
