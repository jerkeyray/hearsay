package game_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jerkeyray/hearsay/internal/game"
	"github.com/jerkeyray/hearsay/internal/kase"
	"github.com/jerkeyray/hearsay/internal/witness"
)

func TestSession_EndSessionRejectsAsk(t *testing.T) {
	ctx := context.Background()
	c := kase.Case{ID: "test", Topics: []kase.Topic{{Name: "x", InitiallyVisible: true}}}
	s, err := game.NewSession(ctx, c, witness.NewStubDriver(), game.Budget{MaxOutputTokens: 10_000})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	s.EndSession()
	if !s.IsEnded() {
		t.Errorf("IsEnded = false after EndSession")
	}
	_, err = s.Ask(ctx, "x", kase.Directly)
	if !errors.Is(err, game.ErrSessionEnded) {
		t.Fatalf("Ask after end: err = %v, want ErrSessionEnded", err)
	}
}

func TestSession_SubmitReconstructionStoresAnswers(t *testing.T) {
	ctx := context.Background()
	c := kase.Case{ID: "test", Topics: []kase.Topic{{Name: "x", InitiallyVisible: true}}}
	s, err := game.NewSession(ctx, c, witness.NewStubDriver(), game.Budget{MaxOutputTokens: 10_000})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if s.Reconstruction() != nil {
		t.Errorf("Reconstruction = non-nil before submit")
	}
	r := game.Reconstruction{
		Answers: []game.Answer{
			{QuestionID: "color", Choice: "blue"},
			{QuestionID: "second_person", DontKnow: true},
		},
	}
	s.SubmitReconstruction(r)
	got := s.Reconstruction()
	if got == nil {
		t.Fatal("Reconstruction nil after submit")
	}
	if len(got.Answers) != 2 {
		t.Fatalf("answers len = %d, want 2", len(got.Answers))
	}
	if got.Answers[0].Choice != "blue" {
		t.Errorf("Answers[0].Choice = %q, want blue", got.Answers[0].Choice)
	}
	if !got.Answers[1].DontKnow {
		t.Errorf("Answers[1].DontKnow = false, want true")
	}
}
