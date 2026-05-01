package game

// Answer is the player's response to one Question on the
// reconstruction form. Exactly one of {Choice, Choices, FreeText} is
// populated based on the Question's Type — except when DontKnow is
// true, which overrides any selection.
type Answer struct {
	QuestionID string
	Choice     string   // Radio
	Choices    []string // MultiSelect
	FreeText   string   // FreeText
	DontKnow   bool
}

// Reconstruction is the player's full set of answers to the form.
// Stored on Session at submit time; consumed by the scoring rubric
// (step 19) to produce the verdict.
type Reconstruction struct {
	Answers []Answer
}
