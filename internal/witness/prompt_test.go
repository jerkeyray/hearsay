package witness_test

import (
	"strings"
	"testing"

	"github.com/jerkeyray/hearsay/internal/kase"
	"github.com/jerkeyray/hearsay/internal/witness"
)

func TestUserPrompt_FirstTurn(t *testing.T) {
	got := witness.UserPrompt("the bag", kase.MomentBefore, nil)
	if !strings.Contains(got, `"the bag"`) {
		t.Errorf("topic missing: %q", got)
	}
	if !strings.Contains(got, "the moment before") {
		t.Errorf("technique label missing: %q", got)
	}
	if strings.Contains(got, "So far in this interview") {
		t.Errorf("history block leaked into first-turn prompt")
	}
	if !strings.Contains(got, "recall tool") {
		t.Errorf("recall tool instruction missing")
	}
}

func TestUserPrompt_WithHistory(t *testing.T) {
	hist := []witness.HistoryItem{
		{Topic: "the car", Technique: kase.Directly, Witness: "she said red.\nsecond line trimmed"},
		{Topic: "the streetlight", Technique: kase.Directly, Witness: "orange."},
	}
	got := witness.UserPrompt("the bag", kase.HowDoYouKnow, hist)
	if !strings.Contains(got, "So far in this interview") {
		t.Errorf("history header missing")
	}
	if !strings.Contains(got, `"the car"`) || !strings.Contains(got, "she said red.") {
		t.Errorf("first prior turn missing or unclipped: %q", got)
	}
	if strings.Contains(got, "second line trimmed") {
		t.Errorf("history not clipped to one line: %q", got)
	}
	if !strings.Contains(got, `"the bag"`) || !strings.Contains(got, "how do you know") {
		t.Errorf("current question missing: %q", got)
	}
}

func TestSystemPrompt_NotEmpty(t *testing.T) {
	if len(witness.SystemPrompt) == 0 {
		t.Fatal("SystemPrompt is empty")
	}
	// Spot-check the banlist marker and the under-react directive.
	if !strings.Contains(witness.SystemPrompt, "Banlist") {
		t.Errorf("banlist section missing")
	}
	if !strings.Contains(witness.SystemPrompt, "Under-react") {
		t.Errorf("under-react directive missing")
	}
}
