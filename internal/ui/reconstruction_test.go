package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jerkeyray/hearsay/internal/kase"
)

func formForTest() kase.Form {
	return kase.Form{
		Questions: []kase.Question{
			{ID: "color", Prompt: "color?", Type: kase.Radio, Choices: []string{"red", "blue", "green"}},
			{ID: "items", Prompt: "items?", Type: kase.MultiSelect, Choices: []string{"folder", "gun", "money"}},
			{ID: "time", Prompt: "time?", Type: kase.FreeText},
		},
	}
}

// keyMsg is a helper that builds a tea.KeyMsg from a single rune.
func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "down", "up", "left", "right", "tab", "enter", "space", "esc", "backspace":
		return tea.KeyMsg{Type: keyType(s)}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func keyType(s string) tea.KeyType {
	switch s {
	case "down":
		return tea.KeyDown
	case "up":
		return tea.KeyUp
	case "left":
		return tea.KeyLeft
	case "right":
		return tea.KeyRight
	case "tab":
		return tea.KeyTab
	case "enter":
		return tea.KeyEnter
	case "space":
		return tea.KeySpace
	case "esc":
		return tea.KeyEsc
	case "backspace":
		return tea.KeyBackspace
	}
	return tea.KeyRunes
}

func TestReconstruction_RadioRightArrowSelectsFirstChoice(t *testing.T) {
	m := newReconstruction(formForTest())
	// Right arrow on a fresh radio with -1 index sets index 0.
	m, _, _, _ = m.Update(keyMsg("right"))
	if m.answers[0].choiceIdx != 0 {
		t.Errorf("choiceIdx after right = %d, want 0", m.answers[0].choiceIdx)
	}
	// Another right → 1.
	m, _, _, _ = m.Update(keyMsg("right"))
	if m.answers[0].choiceIdx != 1 {
		t.Errorf("choiceIdx after 2nd right = %d, want 1", m.answers[0].choiceIdx)
	}
}

func TestReconstruction_DownAdvancesQuestion(t *testing.T) {
	m := newReconstruction(formForTest())
	m, _, _, _ = m.Update(keyMsg("down"))
	if m.cursor != 1 {
		t.Errorf("cursor after down = %d, want 1", m.cursor)
	}
	m, _, _, _ = m.Update(keyMsg("down"))
	m, _, _, _ = m.Update(keyMsg("down"))
	if m.cursor != 2 {
		t.Errorf("cursor clamped at last = %d, want 2", m.cursor)
	}
}

func TestReconstruction_TabTogglesDontKnow(t *testing.T) {
	m := newReconstruction(formForTest())
	m, _, _, _ = m.Update(keyMsg("tab"))
	if !m.answers[0].dontKnow {
		t.Errorf("dontKnow not set after tab")
	}
	m, _, _, _ = m.Update(keyMsg("tab"))
	if m.answers[0].dontKnow {
		t.Errorf("dontKnow not unset after second tab")
	}
}

func TestReconstruction_MultiSelectSpaceTogglesPick(t *testing.T) {
	m := newReconstruction(formForTest())
	// Move to question 1 (multi-select).
	m, _, _, _ = m.Update(keyMsg("down"))
	// Space toggles index 0 (default cursor).
	m, _, _, _ = m.Update(keyMsg("space"))
	if !m.answers[1].picked[0] {
		t.Errorf("picked[0] = false after first space, want true")
	}
	m, _, _, _ = m.Update(keyMsg("space"))
	if m.answers[1].picked[0] {
		t.Errorf("picked[0] = true after second space, want false")
	}
}

func TestReconstruction_FreeTextAcceptsRunes(t *testing.T) {
	m := newReconstruction(formForTest())
	// Move to question 2 (free text).
	m, _, _, _ = m.Update(keyMsg("down"))
	m, _, _, _ = m.Update(keyMsg("down"))
	m, _, _, _ = m.Update(keyMsg("1"))
	m, _, _, _ = m.Update(keyMsg("1"))
	m, _, _, _ = m.Update(keyMsg(":"))
	m, _, _, _ = m.Update(keyMsg("4"))
	m, _, _, _ = m.Update(keyMsg("7"))
	if m.answers[2].freeText != "11:47" {
		t.Errorf("freeText = %q, want %q", m.answers[2].freeText, "11:47")
	}
	// Backspace.
	m, _, _, _ = m.Update(keyMsg("backspace"))
	if m.answers[2].freeText != "11:4" {
		t.Errorf("freeText after backspace = %q, want %q", m.answers[2].freeText, "11:4")
	}
}

func TestReconstruction_ResultBuildsAnswers(t *testing.T) {
	m := newReconstruction(formForTest())
	// Q0 radio → blue (idx 1).
	m, _, _, _ = m.Update(keyMsg("right"))
	m, _, _, _ = m.Update(keyMsg("right"))
	// Q1 multi-select → toggle indexes 0 and 2.
	m, _, _, _ = m.Update(keyMsg("down"))
	m, _, _, _ = m.Update(keyMsg("space"))
	m, _, _, _ = m.Update(keyMsg("right"))
	m, _, _, _ = m.Update(keyMsg("right"))
	m, _, _, _ = m.Update(keyMsg("space"))
	// Q2 free text — type "11:47".
	m, _, _, _ = m.Update(keyMsg("down"))
	for _, r := range []string{"1", "1", ":", "4", "7"} {
		m, _, _, _ = m.Update(keyMsg(r))
	}

	res := m.Result()
	if len(res.Answers) != 3 {
		t.Fatalf("answers len = %d, want 3", len(res.Answers))
	}
	if res.Answers[0].Choice != "blue" {
		t.Errorf("Q0 Choice = %q, want blue", res.Answers[0].Choice)
	}
	if got := res.Answers[1].Choices; len(got) != 2 || got[0] != "folder" || got[1] != "money" {
		t.Errorf("Q1 Choices = %v, want [folder money]", got)
	}
	if res.Answers[2].FreeText != "11:47" {
		t.Errorf("Q2 FreeText = %q, want 11:47", res.Answers[2].FreeText)
	}
}

func TestReconstruction_DontKnowInResult(t *testing.T) {
	m := newReconstruction(formForTest())
	m, _, _, _ = m.Update(keyMsg("right"))
	m, _, _, _ = m.Update(keyMsg("tab"))
	res := m.Result()
	if !res.Answers[0].DontKnow {
		t.Errorf("DontKnow = false, want true")
	}
	if res.Answers[0].Choice != "" {
		t.Errorf("Choice = %q, want empty when dontKnow", res.Answers[0].Choice)
	}
}
