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

// Topic is a node in the case's topic graph (PRD §3.2). Surfacing rules
// land in M3; for M1 the list is fully visible from the start.
type Topic struct {
	Name             string
	InitiallyVisible bool
}
