package kase

// Case bundles everything that defines a case: the briefing copy
// shown before play, the visible topic graph, and the witness's
// beliefs the recall tool consults. Locked truth, reconstruction
// form, and scoring rubric are added in steps 18–19.
type Case struct {
	ID       string
	Title    string
	Briefing string
	Topics   []Topic
	// Beliefs maps a topic name to what the witness believes about
	// that topic. Cases author this map directly — this is the
	// content/engine split: adding a case is writing one of these.
	Beliefs map[string]Belief
}
