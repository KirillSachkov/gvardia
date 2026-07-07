package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestAddPromptOpensAndCancels(t *testing.T) {
	m := ready(t)
	m, cmd := step(m, keyText("A"))
	if m.pathPrompt == nil || m.pathPrompt.mode != pathAdd {
		t.Fatal("A should open the add-project prompt")
	}
	if cmd == nil {
		t.Error("opening the prompt should focus the input (non-nil cmd)")
	}
	m, _ = step(m, keyText("x")) // type into the input
	if m.pathPrompt.input.Value() != "x" {
		t.Errorf("prompt input = %q, want x", m.pathPrompt.input.Value())
	}
	m, _ = step(m, keyPress(tea.KeyEscape))
	if m.pathPrompt != nil {
		t.Error("esc should cancel the path prompt")
	}
}

func TestAddPromptEnterIssuesTrack(t *testing.T) {
	m := ready(t)
	m, _ = step(m, keyText("A"))
	for _, r := range "repo" {
		m, _ = step(m, keyText(string(r)))
	}
	m2, cmd := step(m, keyPress(tea.KeyEnter))
	if m2.pathPrompt != nil {
		t.Error("enter should close the path prompt")
	}
	if cmd == nil {
		t.Error("enter with a non-empty path should issue a track command")
	}
}

func TestCreatePromptEmptyBanners(t *testing.T) {
	m := ready(t)
	m, _ = step(m, keyText("C"))
	if m.pathPrompt == nil || m.pathPrompt.mode != pathCreate {
		t.Fatal("C should open the create-project prompt")
	}
	m2, cmd := step(m, keyPress(tea.KeyEnter)) // empty path
	if m2.pathPrompt != nil {
		t.Error("enter should close the prompt even when empty")
	}
	if cmd != nil {
		t.Error("empty path should not issue a command")
	}
	if m2.banner == "" {
		t.Error("empty path should set a banner")
	}
}

func TestUntrackRequiresCurated(t *testing.T) {
	m := ready(t)
	// ready() is a roots scan (not curated) → X banners, no confirm.
	m, _ = step(m, keyText("X"))
	if m.confirm != nil {
		t.Fatal("X without curation should not open a confirm")
	}
	if m.banner == "" {
		t.Error("X without curation should hint to curate first")
	}
	// Once curated, X opens a confirm.
	m.curated = true
	m, _ = step(m, keyText("X"))
	if m.confirm == nil {
		t.Fatal("X while curated should open an untrack confirm")
	}
	if _, cmd := step(m, keyText("y")); cmd == nil {
		t.Error("confirming untrack should return an action command")
	}
}

func TestProjectsChangedReCollects(t *testing.T) {
	m := ready(t)
	if _, cmd := step(m, projectsChangedMsg{}); cmd == nil {
		t.Error("projectsChangedMsg should trigger a re-collect")
	}
}
