package game_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jerkeyray/hearsay/internal/game"
	"github.com/jerkeyray/hearsay/internal/kase"
	"github.com/jerkeyray/hearsay/internal/witness"
)

// TestSession_BudgetAccumulatesAndEnds runs asks until the
// session-level output-token cap exhausts, then confirms the next
// ask is rejected with ErrSessionEnded.
func TestSession_BudgetAccumulatesAndEnds(t *testing.T) {
	ctx := context.Background()
	c := kase.Case{
		ID: "test",
		Topics: []kase.Topic{
			{Name: "the bag", InitiallyVisible: true},
		},
	}
	// StubDriver charges 200 output tokens per ask. Cap at 500 so the
	// 3rd ask trips the budget.
	s, err := game.NewSession(ctx, c, witness.NewStubDriver(), game.Budget{MaxOutputTokens: 500})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	for i := 0; i < 2; i++ {
		if _, err := s.Ask(ctx, "the bag", kase.Directly); err != nil {
			t.Fatalf("ask %d: %v", i, err)
		}
	}
	if got := s.UsedOutputTokens(); got != 400 {
		t.Errorf("UsedOutputTokens = %d, want 400", got)
	}
	if s.SessionEnded() {
		t.Errorf("session should not be ended after 2 asks (used 400 / 500)")
	}
	// 3rd ask: stub charges another 200, pushing used to 600 ≥ 500.
	if _, err := s.Ask(ctx, "the bag", kase.Directly); err != nil {
		t.Fatalf("ask 3: %v", err)
	}
	if !s.SessionEnded() {
		t.Errorf("session should be ended after 3rd ask (used %d / 500)", s.UsedOutputTokens())
	}
	// 4th ask is rejected.
	_, err = s.Ask(ctx, "the bag", kase.Directly)
	if !errors.Is(err, game.ErrSessionEnded) {
		t.Fatalf("ask 4: err = %v, want ErrSessionEnded", err)
	}
}

func TestSession_ClockDisplay(t *testing.T) {
	ctx := context.Background()
	c := kase.Case{ID: "test", Topics: []kase.Topic{{Name: "x", InitiallyVisible: true}}}

	cases := []struct {
		name   string
		budget game.Budget
		asks   int
		want   string
	}{
		// 50000 cap, 0 used → 50:00
		{"fresh50k", game.Budget{MaxOutputTokens: 50_000}, 0, "50:00"},
		// 50000 cap, 1 ask = 200 used → remaining 49800 → 49:48
		{"after1ask", game.Budget{MaxOutputTokens: 50_000}, 1, "49:48"},
		// 1000 cap, 1 ask = 200 → 800 → 0:48
		{"smallcap", game.Budget{MaxOutputTokens: 1_000}, 1, "0:48"},
		// Unbounded
		{"unbounded", game.Budget{MaxUSD: 1.0}, 0, "—:—"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := game.NewSession(ctx, c, witness.NewStubDriver(), tc.budget)
			if err != nil {
				t.Fatalf("NewSession: %v", err)
			}
			for i := 0; i < tc.asks; i++ {
				if _, err := s.Ask(ctx, "x", kase.Directly); err != nil {
					t.Fatalf("ask %d: %v", i, err)
				}
			}
			if got := s.ClockDisplay(); got != tc.want {
				t.Errorf("ClockDisplay = %q, want %q (used %d)", got, tc.want, s.UsedOutputTokens())
			}
		})
	}
}

func TestSession_ZeroBudgetUsesDefault(t *testing.T) {
	ctx := context.Background()
	c := kase.Case{ID: "test"}
	s, err := game.NewSession(ctx, c, witness.NewStubDriver(), game.Budget{})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if got := s.Budget(); got != game.DefaultBudget {
		t.Errorf("Budget = %+v, want DefaultBudget %+v", got, game.DefaultBudget)
	}
}
