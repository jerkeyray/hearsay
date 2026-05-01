package game

import "fmt"

// RewindTo truncates the in-memory conversation log to length turn+1
// (i.e. the exchange at index turn is the last surviving one).
//
// The SQLite event log is append-only and audit-faithful — events
// from rewound turns stay on disk so the verify chain can prove what
// happened. The "current timeline" the player sees and the verdict
// scores against is the truncated in-memory log.
//
// Rewinding clears the ended flag so the player can keep asking;
// recomputes accumulated token usage from the surviving exchanges;
// and rebuilds the visible-topic set from InitiallyVisible plus
// surfacing rules replayed across the surviving exchanges.
//
// turn must be in [-1, len(log)-1]. -1 means "back to before any
// asks" (empty log). Returns an error on out-of-range.
func (s *Session) RewindTo(turn int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if turn < -1 || turn >= len(s.log) {
		return fmt.Errorf("rewind: turn %d out of range [-1, %d]", turn, len(s.log)-1)
	}
	s.log = s.log[:turn+1]
	s.recomputeFromLogLocked()
	return nil
}

// recomputeFromLogLocked rebuilds usedOutputTokens, usedCostUSD, and
// the visible-topics set from the current s.log. Caller holds s.mu.Lock.
func (s *Session) recomputeFromLogLocked() {
	s.usedOutputTokens = 0
	s.usedCostUSD = 0
	s.ended = false

	visible := make(map[string]bool, len(s.Case.Topics))
	for _, t := range s.Case.Topics {
		if t.InitiallyVisible {
			visible[t.Name] = true
		}
	}
	s.visible = visible

	for _, ex := range s.log {
		s.usedOutputTokens += ex.OutputTokens
		s.usedCostUSD += ex.CostUSD
		s.applySurfacingLocked(ex.Topic, ex.Technique)
	}
}
