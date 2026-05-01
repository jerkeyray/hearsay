package kase

// QuestionType discriminates how a Question is rendered and answered.
type QuestionType int

const (
	// Radio: one Choice from Choices, or "don't know."
	Radio QuestionType = iota
	// MultiSelect: any subset of Choices.
	MultiSelect
	// FreeText: a single line of text.
	FreeText
)

// Question is one item in the reconstruction form. ID is the stable
// key the scoring rubric (step 19) uses to match an Answer to its
// rule.
type Question struct {
	ID      string
	Prompt  string
	Type    QuestionType
	Choices []string // unused for FreeText
}

// Form is the per-case reconstruction questionnaire shown after the
// interrogation. PRD §3.6 / §5.1.
type Form struct {
	Questions []Question
}
