package kase

// Rubric tells the scoring engine the locked truth for each question
// on the reconstruction form. The model never sees this — it lives
// behind the verify chain (PRD §3.8) and is revealed at the verdict
// screen. Cases author this map alongside their Reconstruction Form.
type Rubric struct {
	// Items is keyed by Question.ID.
	Items map[string]RubricItem
}

// RubricItem is the scoring spec for one question.
//
// Truth is the canonical correct answer for Radio + FreeText questions.
// TruthSet is the required exact set for MultiSelect questions.
// WitnessTopic links the question back to a topic in the case's
// Beliefs map so the verdict can show what the witness said about
// it (the most recent recall) and classify the player's error by
// the underlying MemoryKind.
//
// Distractor questions have no canonical answer; the verdict shows
// the player's response with no truth column.
type RubricItem struct {
	Truth        string
	TruthSet     []string
	WitnessTopic string
	Distractor   bool
}
