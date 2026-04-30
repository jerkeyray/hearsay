package kase

// Case is the minimal shape used by the M1 engine spike: enough for the
// splash → briefing → interrogation flow to thread a real value through.
// The full PRD §5 schema (LockedTruth, Beliefs, Topics, Reconstruction,
// Rubric, Budget) lands in M3.
type Case struct {
	ID       string
	Title    string
	Briefing string
}
