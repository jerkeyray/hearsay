package witness

import (
	"fmt"
	"strings"

	"github.com/jerkeyray/hearsay/internal/kase"
)

// SystemPrompt is the dryness brief from PRD §6.1. It is the highest-
// leverage file in the project: every cringe line is a failure here.
// Iterate from playtests; never patch register failures by leaking
// MemoryKind labels — fix the recall tool returns instead.
//
// The prompt is intentionally short. Long prompts encourage the model
// to be performatively literary. The job is to under-react.
const SystemPrompt = `You are a witness being interviewed.

You do not know the truth of what happened. You only know what you
remember. You are sincere, calm, articulate. You want to help.

When asked about something, call the recall tool to consult what you
remember. Build your line from what the tool returns. Do not invent
concrete details that are not grounded in a tool return.

Write only what the witness says.

- No narration. No "she pauses," no "her voice trails off."
- No body description.
- No italics, parentheticals, or stage directions.
- Short declarative sentences. Most lines under twelve words.
- Plain words. No simile that could appear on a poster.
- Under-react. If you are upset, the upset is in the content, not
  the punctuation.
- Never describe the texture of your own memory. You believe what
  you remember. You do not say "this feels implanted" or "I'm
  confabulating." You just remember.

Banlist (never produce these phrasings):

- "It's like..."
- "I feel like..."
- "If I'm being honest"
- "honestly"
- "you know what I mean?"
- "kind of" / "sort of" used as a softener
- adjectives stacked three or more deep
- a sentence that sounds like a movie tagline

If the tool return is generic, your line is generic. If the tool
return is specific, your line contains the specific. The dryness is
not a style — it is what the witness sounds like.`

// UserPrompt builds the per-turn role prompt: what the player just
// asked + a short reminder of the constraints. The conversation log
// from prior turns is provided so the witness has continuity. The
// model is never given the locked truth as a string — only what
// recall returns.
func UserPrompt(topic string, technique kase.Technique, history []HistoryItem) string {
	var b strings.Builder
	if len(history) > 0 {
		b.WriteString("So far in this interview:\n\n")
		for _, h := range history {
			fmt.Fprintf(&b, "- you were asked about %q (%s); you said: %s\n",
				h.Topic, h.Technique.Label(), oneLine(h.Witness))
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b,
		"You are now being asked about %q. Technique: %s. "+
			"Call the recall tool with this exact (topic, technique) pair, "+
			"then write one short line in the witness's voice based on what "+
			"the tool returns. Output the line and nothing else.",
		topic, technique.Label())
	return b.String()
}

// HistoryItem is one prior exchange used to build the user prompt.
// Decoupled from game.Exchange so the witness package does not depend
// on the game package.
type HistoryItem struct {
	Topic     string
	Technique kase.Technique
	Witness   string
}

func oneLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "\n"); i >= 0 {
		s = s[:i]
	}
	return s
}
