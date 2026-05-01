// Package cases exposes the registry of compiled-in cases. Adding a
// new case is two lines here plus one new file in cases/<name>/.
package cases

import (
	"github.com/jerkeyray/hearsay/cases/blackbox"
	"github.com/jerkeyray/hearsay/cases/streetlight"
	"github.com/jerkeyray/hearsay/internal/kase"
)

// All is the ordered list of cases this build ships with. The first
// entry is the default offered to first-time players.
var All = []kase.Case{
	streetlight.Case,
	blackbox.Case,
}

// ByID returns the case with the given ID, or zero + false.
func ByID(id string) (kase.Case, bool) {
	for _, c := range All {
		if c.ID == id {
			return c, true
		}
	}
	return kase.Case{}, false
}
