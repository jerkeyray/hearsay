package kase

// MemoryKind classifies how a witness's belief about a fact behaves
// under questioning. The model never sees this label — it only sees
// the behavioral RecallOutput the recall tool returns. Engine-side,
// MemoryKind is the spec for that behavior. PRD §3.1.
type MemoryKind int

const (
	// Real memories are stable. Asked twice they return the same
	// substance; the source is sensory grounding.
	Real MemoryKind = iota
	// Confabulated memories drift. Each ask regenerates from a gist
	// with stochastic detail-filling. Sources are circular.
	Confabulated
	// Implanted memories are too clean. The same exact phrasing
	// every ask, and the source cracks under "how do you know."
	Implanted
	// Suppressed memories bounce on direct asks. They surface only
	// under "the moment before" or via topic-adjacency.
	Suppressed
)

// Belief is one fact the witness believes about a topic. Cases
// declare beliefs; the recall tool consults them. The model never
// receives a Belief directly — it gets the behavioral output of
// applying a Technique to a Belief.
//
// Field meanings vary by Kind:
//
//   - Real:         Canonical + SensorySource.
//   - Confabulated: Drift (variants picked per ask) + Circular.
//   - Implanted:    Stable + ThinSource.
//   - Suppressed:   Bounce (direct deflection) + Gist (what surfaces
//                   under "the moment before").
//
// Unset fields are zero-valued; the recall handler only reads what's
// relevant to the Kind.
type Belief struct {
	Kind MemoryKind

	// Real
	Canonical     string
	SensorySource string

	// Confabulated
	Drift    []string
	Circular string

	// Implanted
	Stable     string
	ThinSource string

	// Suppressed
	Bounce string
	Gist   string
}
