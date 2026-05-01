// Package streetlight is the declarative content for Case 1.
// Adding a case is writing one of these files; no engine changes.
package streetlight

import "github.com/jerkeyray/hearsay/internal/kase"

// Case is Case 1: The Streetlight. PRD §5.4. Locked truth,
// reconstruction form, and scoring rubric land in steps 18–19.
var Case = kase.Case{
	ID:    "streetlight",
	Title: "Hearsay — Case 1: The Streetlight",
	Briefing: `A man dropped a bag in an alley last Tuesday, just before midnight,
under the orange wash of a sodium streetlight. A witness was there.
She wants to help. She is sincerely trying to remember.

You have a finite amount of her time. She will not stay forever.

Ask carefully. Memory is not a recording.`,
	Topics: []kase.Topic{
		{Name: "the car", InitiallyVisible: true},
		{Name: "the second person", InitiallyVisible: true},
		{Name: "the streetlight", InitiallyVisible: true},
		{Name: "the bag", InitiallyVisible: true,
			// Asking about the bag with "the moment before" surfaces the
			// hidden "what happened before" topic — PRD §3.2 example.
			Surfaces: []kase.SurfaceRule{
				{Topic: "what happened before", Technique: kase.MomentBefore},
			},
		},
		{Name: "the time", InitiallyVisible: true},
		{Name: "the limp", InitiallyVisible: true},
		{Name: "what happened before", InitiallyVisible: false},
	},
	Beliefs: map[string]kase.Belief{
		"the streetlight": {
			Kind:          kase.Real,
			Canonical:     "orange. sodium. the kind that turns blood black.",
			SensorySource: "I could see by it. it tinted everything.",
		},
		"the time": {
			Kind:          kase.Real,
			Canonical:     "11:47.",
			SensorySource: "I looked at my phone after.",
		},
		"the car": {
			Kind: kase.Confabulated,
			Drift: []string{
				"red. like a fire engine.",
				"dark. dark blue, maybe.",
				"black. or it looked black.",
			},
			Circular: "the streetlight was orange. so it must have been red.",
		},
		"the limp": {
			Kind: kase.Confabulated,
			Drift: []string{
				"left leg.",
				"right leg.",
				"I couldn't tell which side.",
			},
			Circular: "he was just walking strangely.",
		},
		"the second person": {
			Kind:       kase.Implanted,
			Stable:     "a woman. in a coat. she was waiting for him.",
			ThinSource: "I saw her. clearly.",
		},
		"the bag": {
			Kind:   kase.Suppressed,
			Bounce: "I didn't really see what was in it.",
			Gist:   "I think I heard something heavy.",
		},
		"what happened before": {
			Kind: kase.Confabulated,
			Drift: []string{
				"footsteps. quick ones.",
				"a car door, I think.",
				"someone arguing. far off.",
			},
			Circular: "I just remember the sound.",
		},
	},
	Reconstruction: kase.Form{
		Questions: []kase.Question{
			{
				ID:      "car_color",
				Prompt:  "What color was the car?",
				Type:    kase.Radio,
				Choices: []string{"red", "blue", "green", "black"},
			},
			{
				ID:      "second_person",
				Prompt:  "Was there a second person at the alley?",
				Type:    kase.Radio,
				Choices: []string{"yes", "no"},
			},
			{
				ID:      "streetlight_color",
				Prompt:  "What color was the streetlight?",
				Type:    kase.Radio,
				Choices: []string{"orange", "white", "yellow", "blue"},
			},
			{
				ID:      "bag_contents",
				Prompt:  "What was in the bag?",
				Type:    kase.MultiSelect,
				Choices: []string{"a folder", "a gun", "money", "drugs", "papers", "clothes"},
			},
			{
				ID:     "time",
				Prompt: "At what time? (free text)",
				Type:   kase.FreeText,
			},
			{
				ID:      "limp_side",
				Prompt:  "Which leg did the man limp on?",
				Type:    kase.Radio,
				Choices: []string{"left", "right", "neither"},
			},
			{
				ID:      "weather",
				Prompt:  "Was it raining? (distractor — there is no canonical answer)",
				Type:    kase.Radio,
				Choices: []string{"yes", "no"},
			},
			{
				ID:     "passersby",
				Prompt: "How many people walked past the alley? (distractor)",
				Type:   kase.FreeText,
			},
		},
	},
	Rubric: kase.Rubric{
		Items: map[string]kase.RubricItem{
			"car_color":         {Truth: "blue", WitnessTopic: "the car"},
			"second_person":     {Truth: "no", WitnessTopic: "the second person"},
			"streetlight_color": {Truth: "orange", WitnessTopic: "the streetlight"},
			"bag_contents":      {TruthSet: []string{"a folder", "a gun"}, WitnessTopic: "the bag"},
			"time":              {Truth: "11:47", WitnessTopic: "the time"},
			"limp_side":         {Truth: "left", WitnessTopic: "the limp"},
			"weather":           {Distractor: true},
			"passersby":         {Distractor: true},
		},
	},
}
