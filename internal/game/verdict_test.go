package game_test

import (
	"testing"

	"github.com/jerkeyray/hearsay/cases/streetlight"
	"github.com/jerkeyray/hearsay/internal/game"
	"github.com/jerkeyray/hearsay/internal/kase"
)

// scoreCase1 is a helper that scores a Reconstruction against the
// real Case 1 with no conversation log (no "she said" lines). The
// rubric scoring path doesn't depend on the witness's actual
// utterances — it classifies on Belief.Kind.
func scoreCase1(answers []game.Answer) game.Verdict {
	return game.Score(streetlight.Case, nil, game.Reconstruction{Answers: answers})
}

func TestScore_PerfectPlay(t *testing.T) {
	v := scoreCase1([]game.Answer{
		{QuestionID: "car_color", Choice: "blue"},
		{QuestionID: "second_person", Choice: "no"},
		{QuestionID: "streetlight_color", Choice: "orange"},
		{QuestionID: "bag_contents", Choices: []string{"a folder", "a gun"}},
		{QuestionID: "time", FreeText: "11:47"},
		{QuestionID: "limp_side", Choice: "left"},
		{QuestionID: "weather", Choice: "yes"}, // distractor — doesn't count
		{QuestionID: "passersby", FreeText: "two"}, // distractor
	})
	if v.Score != v.Total {
		t.Errorf("perfect play: Score = %d, Total = %d", v.Score, v.Total)
	}
	if v.Total != 6 {
		t.Errorf("Total = %d, want 6 (non-distractors)", v.Total)
	}
}

func TestScore_FollowedConfabulation(t *testing.T) {
	v := scoreCase1([]game.Answer{
		// Picking "red" follows the witness's confabulated drift.
		{QuestionID: "car_color", Choice: "red"},
	})
	got := findItem(v, "car_color")
	if got.Error != game.FollowedConfabulation {
		t.Errorf("car_color error = %v, want FollowedConfabulation", got.Error)
	}
	if got.Correct {
		t.Errorf("car_color marked correct on red")
	}
}

func TestScore_FellForImplant(t *testing.T) {
	v := scoreCase1([]game.Answer{
		{QuestionID: "second_person", Choice: "yes"},
	})
	got := findItem(v, "second_person")
	if got.Error != game.FellForImplant {
		t.Errorf("second_person error = %v, want FellForImplant", got.Error)
	}
}

func TestScore_MissedSuppressed(t *testing.T) {
	v := scoreCase1([]game.Answer{
		// Picked only the folder — missed the suppressed gun.
		{QuestionID: "bag_contents", Choices: []string{"a folder"}},
	})
	got := findItem(v, "bag_contents")
	if got.Error != game.MissedSuppressed {
		t.Errorf("bag_contents error = %v, want MissedSuppressed", got.Error)
	}
}

func TestScore_DontKnowIsDidntAsk(t *testing.T) {
	v := scoreCase1([]game.Answer{
		{QuestionID: "limp_side", DontKnow: true},
	})
	got := findItem(v, "limp_side")
	if got.Error != game.DidntAsk {
		t.Errorf("limp_side dontknow error = %v, want DidntAsk", got.Error)
	}
}

func TestScore_FreeTextSubstringMatch(t *testing.T) {
	v := scoreCase1([]game.Answer{
		{QuestionID: "time", FreeText: "around 11:47 PM"},
	})
	got := findItem(v, "time")
	if !got.Correct {
		t.Errorf("'around 11:47 PM' should match truth '11:47'; got error %v", got.Error)
	}
}

func TestScore_DistractorReportedNoCanonical(t *testing.T) {
	v := scoreCase1([]game.Answer{
		{QuestionID: "weather", Choice: "yes"},
	})
	got := findItem(v, "weather")
	if got.Error != game.NoCanonicalAnswer {
		t.Errorf("weather (distractor) error = %v, want NoCanonicalAnswer", got.Error)
	}
	if got.Truth != "" {
		t.Errorf("weather Truth = %q, want empty for distractor", got.Truth)
	}
}

func TestScore_ProducesQualitativeGrade(t *testing.T) {
	// Player who fell for everything bad.
	v := scoreCase1([]game.Answer{
		{QuestionID: "car_color", Choice: "red"},
		{QuestionID: "second_person", Choice: "yes"},
		{QuestionID: "streetlight_color", Choice: "orange"},
		{QuestionID: "bag_contents", Choices: []string{"a folder"}},
		{QuestionID: "time", FreeText: "11:47"},
		{QuestionID: "limp_side", Choice: "left"},
	})
	if v.Summary == "" {
		t.Errorf("Summary empty after mixed play")
	}
	if !contains(v.Summary, "implant") || !contains(v.Summary, "suppressed") {
		t.Errorf("summary missing implant/suppressed mention: %q", v.Summary)
	}
}

func TestScore_WitnessLineFromExchanges(t *testing.T) {
	exchanges := []game.Exchange{
		{Topic: "the car", Witness: "she said red."},
		{Topic: "the bag", Witness: "she said folder."},
		{Topic: "the car", Witness: "she said dark blue."},
	}
	v := game.Score(streetlight.Case, exchanges, game.Reconstruction{
		Answers: []game.Answer{{QuestionID: "car_color", Choice: "red"}},
	})
	got := findItem(v, "car_color")
	// Witness column should reflect the most recent line for "the car".
	if got.Witness != "she said dark blue." {
		t.Errorf("Witness = %q, want most recent 'she said dark blue.'", got.Witness)
	}
}

// helpers ----------------------------------------------------------

func findItem(v game.Verdict, id string) game.VerdictItem {
	for _, it := range v.Items {
		if it.QuestionID == id {
			return it
		}
	}
	return game.VerdictItem{}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})())
}

// guard against unused-import in case future edits drop kase usage.
var _ = kase.Real
