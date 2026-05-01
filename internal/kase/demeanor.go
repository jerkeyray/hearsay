package kase

// Demeanor is the witness's current visible state, signalled by the
// note_demeanor tool. The model picks from a small fixed set rather
// than free text so the renderer can map it to a consistent portrait
// line. PRD §6.3.
type Demeanor string

const (
	// DemeanorEngaged is the default: cooperative, present, trying.
	DemeanorEngaged Demeanor = "engaged"
	// DemeanorUncomfortable: shifting, looking elsewhere, soft voice.
	DemeanorUncomfortable Demeanor = "uncomfortable"
	// DemeanorDefensive: holding ground, repeating, harder voice.
	DemeanorDefensive Demeanor = "defensive"
	// DemeanorTired: late-session, low affect, less detail.
	DemeanorTired Demeanor = "tired"
)

// AllDemeanors is the canonical set the tool will accept. Anything
// else returns an error from the tool.
var AllDemeanors = []Demeanor{
	DemeanorEngaged,
	DemeanorUncomfortable,
	DemeanorDefensive,
	DemeanorTired,
}

// ParseDemeanor returns the Demeanor matching s, or ("", false) if
// s is not in AllDemeanors.
func ParseDemeanor(s string) (Demeanor, bool) {
	for _, d := range AllDemeanors {
		if string(d) == s {
			return d, true
		}
	}
	return "", false
}
