package kase

// Technique is one of the five fixed interrogation techniques per PRD §3.3.
// The set is closed and global across cases.
type Technique int

const (
	Directly Technique = iota
	MomentBefore
	HowDoYouKnow
	PushBack
	CircleBackLater
)

func (t Technique) Label() string {
	switch t {
	case Directly:
		return "directly"
	case MomentBefore:
		return "the moment before"
	case HowDoYouKnow:
		return "how do you know"
	case PushBack:
		return "push back"
	case CircleBackLater:
		return "circle back later"
	}
	return ""
}

// AllTechniques is the canonical, ordered set used by the UI.
var AllTechniques = []Technique{
	Directly,
	MomentBefore,
	HowDoYouKnow,
	PushBack,
	CircleBackLater,
}

// ParseTechnique returns the Technique with the given Label, or
// (0, false) if no match. Used to validate the technique string the
// LLM passes to the recall tool.
func ParseTechnique(label string) (Technique, bool) {
	for _, t := range AllTechniques {
		if t.Label() == label {
			return t, true
		}
	}
	return 0, false
}

// Topic is a node in the case's topic graph (PRD §3.2). Topics with
// InitiallyVisible=false start hidden and only surface when one of
// the case's Surfaces rules fires from another topic.
type Topic struct {
	Name             string
	InitiallyVisible bool
	// Surfaces lists "asking THIS topic with that technique surfaces
	// the named topic." Order does not matter; rules are evaluated
	// after each turn and idempotent (re-firing has no effect).
	Surfaces []SurfaceRule
}

// SurfaceRule says: when the just-completed turn asked the parent
// topic using Technique, mark Topic visible. PRD §3.2 / §5.3.
//
// To surface one topic from multiple techniques, add multiple rules
// with the same Topic and different Technique values. To surface from
// any technique, list one rule per technique in AllTechniques.
type SurfaceRule struct {
	// Topic is the name of the topic to make visible.
	Topic string
	// Technique that triggers the rule.
	Technique Technique
}
