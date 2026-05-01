package game_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jerkeyray/hearsay/internal/game"
	"github.com/jerkeyray/hearsay/internal/kase"
	"github.com/jerkeyray/hearsay/internal/witness"
)

func TestSession_BranchStubDriverPrefillsLog(t *testing.T) {
	ctx := context.Background()
	c := kase.Case{
		ID:     "test",
		Topics: []kase.Topic{{Name: "x", InitiallyVisible: true}},
	}
	parent, err := game.NewSession(ctx, c, witness.NewStubDriver(), game.Budget{MaxOutputTokens: 10_000})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	for i := 0; i < 4; i++ {
		if _, err := parent.Ask(ctx, "x", kase.Directly); err != nil {
			t.Fatalf("ask %d: %v", i, err)
		}
	}
	tmp := t.TempDir()
	child, err := parent.Branch(1, filepath.Join(tmp, "child.db"))
	if err != nil {
		t.Fatalf("Branch: %v", err)
	}
	defer child.Close(ctx)
	defer parent.Close(ctx)

	if got := child.TurnCount(); got != 2 {
		t.Errorf("child TurnCount = %d, want 2 (prefix [0..1])", got)
	}
	if got := parent.TurnCount(); got != 4 {
		t.Errorf("parent TurnCount mutated = %d, want 4", got)
	}
	if child.Timeline != "A.1" {
		t.Errorf("child Timeline = %q, want A.1", child.Timeline)
	}
	// Child should be independently askable.
	if _, err := child.Ask(ctx, "x", kase.Directly); err != nil {
		t.Errorf("child Ask after branch: %v", err)
	}
	if got := parent.TurnCount(); got != 4 {
		t.Errorf("child Ask leaked into parent: parent TurnCount = %d", got)
	}
}

func TestSession_BranchTimelineLabelChain(t *testing.T) {
	ctx := context.Background()
	c := kase.Case{ID: "test", Topics: []kase.Topic{{Name: "x", InitiallyVisible: true}}}
	parent, err := game.NewSession(ctx, c, witness.NewStubDriver(), game.Budget{MaxOutputTokens: 10_000})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	for i := 0; i < 3; i++ {
		if _, err := parent.Ask(ctx, "x", kase.Directly); err != nil {
			t.Fatalf("ask: %v", err)
		}
	}
	tmp := t.TempDir()
	a1, err := parent.Branch(1, filepath.Join(tmp, "a1.db"))
	if err != nil {
		t.Fatalf("first branch: %v", err)
	}
	a2, err := parent.Branch(0, filepath.Join(tmp, "a2.db"))
	if err != nil {
		t.Fatalf("second branch: %v", err)
	}
	if a1.Timeline != "A.1" {
		t.Errorf("first branch label = %q, want A.1", a1.Timeline)
	}
	if a2.Timeline != "A.2" {
		t.Errorf("second branch label = %q, want A.2", a2.Timeline)
	}
	for _, ask := range []int{0, 1} {
		if _, err := a1.Ask(ctx, "x", kase.Directly); err != nil {
			t.Fatalf("a1 ask %d: %v", ask, err)
		}
	}
	a11, err := a1.Branch(0, filepath.Join(tmp, "a11.db"))
	if err != nil {
		t.Fatalf("nested branch: %v", err)
	}
	if a11.Timeline != "A.1.1" {
		t.Errorf("nested branch label = %q, want A.1.1", a11.Timeline)
	}
}

func TestSession_BranchOutOfRange(t *testing.T) {
	ctx := context.Background()
	c := kase.Case{ID: "test", Topics: []kase.Topic{{Name: "x", InitiallyVisible: true}}}
	s, err := game.NewSession(ctx, c, witness.NewStubDriver(), game.Budget{MaxOutputTokens: 10_000})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if _, err := s.Ask(ctx, "x", kase.Directly); err != nil {
		t.Fatalf("ask: %v", err)
	}
	tmp := t.TempDir()
	if _, err := s.Branch(5, filepath.Join(tmp, "out.db")); err == nil {
		t.Errorf("expected out-of-range error")
	}
}
