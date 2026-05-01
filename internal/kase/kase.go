package kase

// Case bundles everything that defines a case: the briefing copy,
// the visible topic graph, the witness's beliefs the recall tool
// consults, the post-session reconstruction form, and the scoring
// rubric used to produce the verdict.
type Case struct {
	ID       string
	Title    string
	Briefing string
	Topics   []Topic
	// Beliefs maps a topic name to what the witness believes about
	// that topic. Cases author this map directly — this is the
	// content/engine split: adding a case is writing one of these.
	Beliefs map[string]Belief
	// Reconstruction is the questionnaire shown after the session.
	Reconstruction Form
	// Rubric scores the reconstruction against locked truth.
	Rubric Rubric
}
