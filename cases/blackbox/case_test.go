package blackbox_test

import (
	"testing"

	"github.com/jerkeyray/hearsay/cases/blackbox"
	"github.com/jerkeyray/hearsay/internal/game"
	"github.com/jerkeyray/hearsay/internal/kase"
)

// TestCase2_AllMemoryKindsPresent confirms the case exercises all
// four MemoryKinds in its Beliefs map. Acts as a content-side
// regression: lose a kind from the case, the test fails.
func TestCase2_AllMemoryKindsPresent(t *testing.T) {
	seen := map[kase.MemoryKind]bool{}
	for _, b := range blackbox.Case.Beliefs {
		seen[b.Kind] = true
	}
	for _, want := range []kase.MemoryKind{kase.Real, kase.Confabulated, kase.Implanted, kase.Suppressed} {
		if !seen[want] {
			t.Errorf("MemoryKind %v missing from Case 2 beliefs", want)
		}
	}
}

// TestCase2_RubricCoversFormQuestions makes sure every form question
// has either a rubric entry or is marked Distractor.
func TestCase2_RubricCoversFormQuestions(t *testing.T) {
	for _, q := range blackbox.Case.Reconstruction.Questions {
		ri, ok := blackbox.Case.Rubric.Items[q.ID]
		if !ok {
			t.Errorf("question %q has no rubric entry", q.ID)
			continue
		}
		if !ri.Distractor && ri.Truth == "" && len(ri.TruthSet) == 0 {
			t.Errorf("question %q rubric has no truth and isn't a distractor", q.ID)
		}
	}
}

// TestCase2_ScoresEndToEnd runs the scoring engine against a
// best-case answer set and confirms a perfect score on the
// non-distractor questions.
func TestCase2_ScoresEndToEnd(t *testing.T) {
	r := game.Reconstruction{
		Answers: []game.Answer{
			{QuestionID: "courier_height", Choice: "average"},
			{QuestionID: "van_color", Choice: "navy"},
			{QuestionID: "badge_logo", Choice: "no"},
			{QuestionID: "box_contents_signal", Choices: []string{"ticking"}},
			{QuestionID: "time", FreeText: "9:13"},
			{QuestionID: "raining", Choice: "yes"},
			{QuestionID: "doorman_called_super", Choice: "yes"},
		},
	}
	v := game.Score(blackbox.Case, nil, r)
	if v.Score != v.Total {
		t.Errorf("perfect Case 2 play: Score = %d, Total = %d", v.Score, v.Total)
	}
	if v.Total != 6 {
		t.Errorf("Total = %d, want 6 (non-distractors)", v.Total)
	}
}
