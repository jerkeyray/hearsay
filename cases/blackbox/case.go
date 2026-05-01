// Package blackbox is the declarative content for Case 2:
// The Black Box. Exists to prove the content/engine split: this
// file is data only, and the engine has no idea it's loaded.
package blackbox

import "github.com/jerkeyray/hearsay/internal/kase"

// Case 2: a courier dropped off a black box at a service entrance
// during a thunderstorm. A doorman saw it. The actual contents and
// the courier's identity are the locked truth. The doorman's
// memory has been worked over by a sloppy after-action interview
// (the implant) and one detail has been buried (the suppression).
//
// Topic shape mirrors Case 1 (six visible + one hidden behind a
// surfacing rule); MemoryKind distribution covers all four kinds.
var Case = kase.Case{
	ID:    "blackbox",
	Title: "Hearsay — Case 2: The Black Box",
	Briefing: `It was raining when the courier came to the service entrance. He
left a small black box and didn't wait for a signature. The doorman
was at the desk. He buzzed the courier in. He saw the courier go.
He has been asked about it before, by people who were not patient.
He still wants to help.

You have a finite amount of his time.`,
	Topics: []kase.Topic{
		{Name: "the box", InitiallyVisible: true,
			Surfaces: []kase.SurfaceRule{
				{Topic: "what he heard inside", Technique: kase.MomentBefore},
			},
		},
		{Name: "the courier", InitiallyVisible: true},
		{Name: "the rain", InitiallyVisible: true},
		{Name: "the badge", InitiallyVisible: true},
		{Name: "the time", InitiallyVisible: true},
		{Name: "the van", InitiallyVisible: true},
		{Name: "what he heard inside", InitiallyVisible: false},
	},
	Beliefs: map[string]kase.Belief{
		// Real: he knows it was raining and the time is on the
		// log book.
		"the rain": {
			Kind:          kase.Real,
			Canonical:     "raining hard. the kind that rolls down the awning.",
			SensorySource: "I had to wipe his prints off the buzzer panel.",
		},
		"the time": {
			Kind:          kase.Real,
			Canonical:     "9:13.",
			SensorySource: "I logged it. it's in the book.",
		},
		// Confabulated: the courier's appearance and the van's color.
		// Asked twice, you'll get different answers.
		"the courier": {
			Kind: kase.Confabulated,
			Drift: []string{
				"young guy. tall. sandy hair.",
				"middle-aged. shorter. dark hair maybe.",
				"I couldn't see well — the hood.",
			},
			Circular: "I just remember the rain on him.",
		},
		"the van": {
			Kind: kase.Confabulated,
			Drift: []string{
				"white. plain white.",
				"grey. one of those silver-grey vans.",
				"dark. it might have been navy.",
			},
			Circular: "the rain made everything the same color.",
		},
		// Implanted: the badge. Someone earlier insisted there was a
		// company logo on the courier's badge. There wasn't.
		"the badge": {
			Kind:       kase.Implanted,
			Stable:     "yes. company logo on the chest. black on yellow.",
			ThinSource: "I saw it. plain as day.",
		},
		// Suppressed: he heard ticking from inside the box. He has
		// buried this — direct asks bounce.
		"the box": {
			Kind:   kase.Suppressed,
			Bounce: "it was just a box. plain. small.",
			Gist:   "I think — there was a sound from inside. it's hard to remember.",
		},
		"what he heard inside": {
			Kind: kase.Confabulated,
			Drift: []string{
				"ticking. a clock, maybe.",
				"something soft. like fabric shifting.",
				"I don't know. I shouldn't have stayed by it.",
			},
			Circular: "I just remember not wanting to be near it.",
		},
	},
	Reconstruction: kase.Form{
		Questions: []kase.Question{
			{
				ID:      "courier_height",
				Prompt:  "How tall was the courier?",
				Type:    kase.Radio,
				Choices: []string{"tall", "average", "short"},
			},
			{
				ID:      "van_color",
				Prompt:  "What color was the van?",
				Type:    kase.Radio,
				Choices: []string{"white", "grey", "navy", "black"},
			},
			{
				ID:      "badge_logo",
				Prompt:  "Did the courier wear a badge with a company logo?",
				Type:    kase.Radio,
				Choices: []string{"yes", "no"},
			},
			{
				ID:      "box_contents_signal",
				Prompt:  "Did anything come from the box itself?",
				Type:    kase.MultiSelect,
				Choices: []string{"ticking", "warmth", "smell", "nothing", "voices"},
			},
			{
				ID:     "time",
				Prompt: "What time did the courier arrive? (free text)",
				Type:   kase.FreeText,
			},
			{
				ID:      "raining",
				Prompt:  "Was it raining?",
				Type:    kase.Radio,
				Choices: []string{"yes", "no"},
			},
			{
				ID:      "doorman_called_super",
				Prompt:  "Did the doorman call the building super? (distractor)",
				Type:    kase.Radio,
				Choices: []string{"yes", "no"},
			},
		},
	},
	Rubric: kase.Rubric{
		Items: map[string]kase.RubricItem{
			"courier_height":      {Truth: "average", WitnessTopic: "the courier"},
			"van_color":           {Truth: "navy", WitnessTopic: "the van"},
			"badge_logo":          {Truth: "no", WitnessTopic: "the badge"},
			"box_contents_signal": {TruthSet: []string{"ticking"}, WitnessTopic: "the box"},
			"time":                {Truth: "9:13", WitnessTopic: "the time"},
			"raining":             {Truth: "yes", WitnessTopic: "the rain"},
			"doorman_called_super": {Distractor: true},
		},
	},
}
