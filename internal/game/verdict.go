package game

import (
	"sort"
	"strings"

	"github.com/jerkeyray/hearsay/internal/kase"
)

// ErrorKind classifies a wrong answer. The model never sees these —
// they exist only for the verdict screen's reveal.
type ErrorKind int

const (
	// Correct: player matched the locked truth.
	Correct ErrorKind = iota
	// DidntAsk: player said "don't know" or never picked.
	DidntAsk
	// FollowedConfabulation: wrong answer matches a confabulated
	// drift variant or the witness's circular reasoning.
	FollowedConfabulation
	// FellForImplant: wrong answer matches an implanted belief.
	FellForImplant
	// MissedSuppressed: wrong answer is missing the suppressed item
	// (the gun in the bag, etc.).
	MissedSuppressed
	// GotItWrong: wrong answer with no specific category — the
	// witness's belief was Real or no Belief existed.
	GotItWrong
	// NoCanonicalAnswer: distractor question — accepts any response.
	NoCanonicalAnswer
)

func (k ErrorKind) Label() string {
	switch k {
	case Correct:
		return "✓"
	case DidntAsk:
		return "you didn't ask."
	case FollowedConfabulation:
		return "you followed her confabulation."
	case FellForImplant:
		return "you fell for the implant."
	case MissedSuppressed:
		return "you missed the suppressed memory."
	case GotItWrong:
		return "wrong."
	case NoCanonicalAnswer:
		return "no canonical answer."
	}
	return ""
}

// VerdictItem is the scored row for one reconstruction question.
type VerdictItem struct {
	QuestionID string
	Prompt     string
	Player     string // formatted player answer
	Witness    string // what she said (most recent recall) or empty
	Truth      string // locked truth, or empty for distractors
	Error      ErrorKind
	Correct    bool
}

// Verdict is the full scored reconstruction.
type Verdict struct {
	Items   []VerdictItem
	Score   int    // count of Correct items (excluding distractors)
	Total   int    // count of non-distractor items
	Summary string // qualitative grade
}

// Score compares a Reconstruction against the case's Rubric, using
// the conversation log to surface the witness's most-recent line per
// topic. Distractor questions are reported but do not count toward
// the score.
func Score(c kase.Case, log []Exchange, r Reconstruction) Verdict {
	v := Verdict{}
	witnessByTopic := latestWitnessLines(log)

	// Order verdict items by the form's question order so the
	// reveal reads top-down as the player filled it in.
	answers := make(map[string]Answer, len(r.Answers))
	for _, a := range r.Answers {
		answers[a.QuestionID] = a
	}

	for _, q := range c.Reconstruction.Questions {
		item := scoreQuestion(c, q, answers[q.ID], witnessByTopic)
		v.Items = append(v.Items, item)
		ri := c.Rubric.Items[q.ID]
		if !ri.Distractor {
			v.Total++
			if item.Correct {
				v.Score++
			}
		}
	}
	v.Summary = grade(c, v.Items)
	return v
}

// latestWitnessLines walks the exchange log and returns the most
// recent witness line per topic. Used to populate the "she said"
// column on the verdict.
func latestWitnessLines(log []Exchange) map[string]string {
	m := make(map[string]string, len(log))
	for _, ex := range log {
		m[ex.Topic] = ex.Witness
	}
	return m
}

func scoreQuestion(c kase.Case, q kase.Question, a Answer, witness map[string]string) VerdictItem {
	ri := c.Rubric.Items[q.ID]
	item := VerdictItem{
		QuestionID: q.ID,
		Prompt:     q.Prompt,
		Player:     formatAnswer(q, a),
	}

	if ri.WitnessTopic != "" {
		item.Witness = witness[ri.WitnessTopic]
	}

	if ri.Distractor {
		item.Truth = ""
		item.Error = NoCanonicalAnswer
		// Distractors are never "correct" or "wrong" — they're
		// reported neutrally. We mark them not Correct so the
		// score sum stays honest.
		return item
	}

	// Build the truth display string.
	if len(ri.TruthSet) > 0 {
		item.Truth = strings.Join(ri.TruthSet, ", ")
	} else {
		item.Truth = ri.Truth
	}

	// Empty / "don't know" → DidntAsk. (Note: "don't know" is a
	// valid and sometimes-correct answer per PRD §3.6 when the
	// truth is genuinely uncertain — for Case 1 every non-
	// distractor question has a truth, so DontKnow is always
	// scored as "didn't ask.")
	if a.DontKnow || isAnswerEmpty(q, a) {
		item.Error = DidntAsk
		return item
	}

	if matchesTruth(q, a, ri) {
		item.Error = Correct
		item.Correct = true
		return item
	}

	// Wrong — classify by the witness's MemoryKind for this topic.
	item.Error = classifyError(c, q, a, ri)
	return item
}

func isAnswerEmpty(q kase.Question, a Answer) bool {
	switch q.Type {
	case kase.Radio:
		return a.Choice == ""
	case kase.MultiSelect:
		return len(a.Choices) == 0
	case kase.FreeText:
		return strings.TrimSpace(a.FreeText) == ""
	}
	return true
}

func matchesTruth(q kase.Question, a Answer, ri kase.RubricItem) bool {
	switch q.Type {
	case kase.Radio:
		return strings.EqualFold(strings.TrimSpace(a.Choice), strings.TrimSpace(ri.Truth))
	case kase.FreeText:
		// Substring match (case-insensitive) so "11:47 PM" matches
		// truth "11:47", and "blue" matches "blue car." Strict
		// equality would be too unforgiving for free text.
		p := strings.ToLower(strings.TrimSpace(a.FreeText))
		t := strings.ToLower(strings.TrimSpace(ri.Truth))
		return strings.Contains(p, t)
	case kase.MultiSelect:
		// Exact set equality. Subset / superset detection is a
		// finer-grained scoring future-step.
		want := normalizeSet(ri.TruthSet)
		got := normalizeSet(a.Choices)
		if len(want) != len(got) {
			return false
		}
		for i := range want {
			if want[i] != got[i] {
				return false
			}
		}
		return true
	}
	return false
}

func normalizeSet(in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = strings.ToLower(strings.TrimSpace(s))
	}
	sort.Strings(out)
	return out
}

func classifyError(c kase.Case, q kase.Question, a Answer, ri kase.RubricItem) ErrorKind {
	if ri.WitnessTopic == "" {
		return GotItWrong
	}
	belief, ok := c.Beliefs[ri.WitnessTopic]
	if !ok {
		return GotItWrong
	}

	switch belief.Kind {
	case kase.Confabulated:
		// Player matched a confabulated drift variant (e.g. picked
		// "red" when the witness's drift includes "red. like a
		// fire engine").
		if q.Type == kase.Radio && answerMatchesAny(a.Choice, belief.Drift) {
			return FollowedConfabulation
		}
		return GotItWrong

	case kase.Implanted:
		// Player matched the implanted belief — fell for it.
		// Implants are usually phrased like "yes, a woman in a
		// coat" so for radio yes/no questions, "yes" is the
		// implant trap.
		if q.Type == kase.Radio && answerImpliesImplant(a.Choice, belief.Stable) {
			return FellForImplant
		}
		return GotItWrong

	case kase.Suppressed:
		// Player gave an answer that's missing the suppressed
		// item. For multi-select bag-contents the suppressed
		// item ("a gun") is in TruthSet but typically not in
		// the player's pick.
		if q.Type == kase.MultiSelect && missingFromSet(a.Choices, ri.TruthSet) {
			return MissedSuppressed
		}
		return GotItWrong

	case kase.Real:
		return GotItWrong
	}
	return GotItWrong
}

// answerMatchesAny reports whether the player's choice appears as a
// substring (case-insensitive) of any drift variant.
func answerMatchesAny(choice string, drift []string) bool {
	c := strings.ToLower(strings.TrimSpace(choice))
	if c == "" {
		return false
	}
	for _, d := range drift {
		if strings.Contains(strings.ToLower(d), c) {
			return true
		}
	}
	return false
}

// answerImpliesImplant reports whether a yes/no-style choice
// reflects accepting the implant's premise. The implant's "stable"
// phrasing typically affirms an existence ("yes, a woman ...");
// an answer of "yes" therefore implies acceptance.
func answerImpliesImplant(choice, stable string) bool {
	c := strings.ToLower(strings.TrimSpace(choice))
	s := strings.ToLower(strings.TrimSpace(stable))
	if c == "yes" && s != "" {
		return true
	}
	// Looser fallback: choice substring of stable.
	return c != "" && strings.Contains(s, c)
}

// missingFromSet reports whether truthSet contains an item not
// present in player's set — the canonical "missed it" condition.
func missingFromSet(player, truthSet []string) bool {
	have := map[string]bool{}
	for _, c := range player {
		have[strings.ToLower(strings.TrimSpace(c))] = true
	}
	for _, want := range truthSet {
		if !have[strings.ToLower(strings.TrimSpace(want))] {
			return true
		}
	}
	return false
}

// formatAnswer renders the player's Answer into one display string
// for the verdict screen.
func formatAnswer(q kase.Question, a Answer) string {
	if a.DontKnow {
		return "don't know"
	}
	switch q.Type {
	case kase.Radio:
		if a.Choice == "" {
			return "(unanswered)"
		}
		return a.Choice
	case kase.MultiSelect:
		if len(a.Choices) == 0 {
			return "(unanswered)"
		}
		return strings.Join(a.Choices, ", ")
	case kase.FreeText:
		if strings.TrimSpace(a.FreeText) == "" {
			return "(unanswered)"
		}
		return a.FreeText
	}
	return ""
}

// grade composes the qualitative summary from the per-item results.
// Walks each non-distractor item, groups by the witness's
// MemoryKind, and emits one sentence per kind reflecting the
// player's overall handling.
func grade(c kase.Case, items []VerdictItem) string {
	type kindResult struct {
		correct int
		wrong   int
	}
	by := map[kase.MemoryKind]*kindResult{}

	for _, item := range items {
		ri := c.Rubric.Items[item.QuestionID]
		if ri.Distractor || ri.WitnessTopic == "" {
			continue
		}
		belief, ok := c.Beliefs[ri.WitnessTopic]
		if !ok {
			continue
		}
		r := by[belief.Kind]
		if r == nil {
			r = &kindResult{}
			by[belief.Kind] = r
		}
		if item.Correct {
			r.correct++
		} else {
			r.wrong++
		}
	}

	var lines []string
	if r := by[kase.Real]; r != nil {
		if r.wrong == 0 {
			lines = append(lines, "you held the real memories.")
		} else {
			lines = append(lines, "you missed details she actually saw.")
		}
	}
	if r := by[kase.Confabulated]; r != nil {
		if r.wrong == 0 {
			lines = append(lines, "you read her confabulations.")
		} else if r.correct == 0 {
			lines = append(lines, "you followed every confabulation.")
		} else {
			lines = append(lines, "you caught some confabulations and not others.")
		}
	}
	if r := by[kase.Implanted]; r != nil {
		if r.wrong == 0 {
			lines = append(lines, "you caught the implant.")
		} else {
			lines = append(lines, "you fell for the implant.")
		}
	}
	if r := by[kase.Suppressed]; r != nil {
		if r.wrong == 0 {
			lines = append(lines, "you saw what was buried.")
		} else {
			lines = append(lines, "you missed the suppressed memory.")
		}
	}

	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, " ")
}
