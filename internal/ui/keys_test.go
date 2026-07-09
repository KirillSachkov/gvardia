package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/KirillSachkov/gvardia/internal/config"
	"github.com/KirillSachkov/gvardia/internal/model"
)

func TestNormalizeKey(t *testing.T) {
	cases := map[string]string{
		"в": "d", "ф": "a", "к": "r", "н": "y", "т": "n", "ш": "i",
		"К": "R", "Ч": "X", "Ф": "A", "С": "C", // uppercase → uppercase Latin
		"d": "d", "R": "R", // already Latin passes through
		"enter": "enter", "esc": "esc", "tab": "tab", "backspace": "backspace",
		"ctrl+c": "ctrl+c", "up": "up",
	}
	for in, want := range cases {
		if got := normalizeKey(in); got != want {
			t.Errorf("normalizeKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEnterDrillsAndEscClimbs(t *testing.T) {
	m := ready(t)
	if m.level != levelProjects {
		t.Fatalf("start level = %d, want projects", m.level)
	}
	m, _ = step(m, keyPress(tea.KeyEnter)) // projects → work
	if m.level != levelWork {
		t.Fatalf("after enter, level = %d, want work", m.level)
	}
	m, _ = step(m, keyPress(tea.KeyEnter)) // work → detail (alpha a1 selectable)
	if m.level != levelDetail {
		t.Fatalf("after 2nd enter, level = %d, want detail", m.level)
	}
	m, _ = step(m, keyPress(tea.KeyEscape)) // detail → work
	if m.level != levelWork {
		t.Fatalf("after esc, level = %d, want work", m.level)
	}
	m, _ = step(m, keyPress(tea.KeyEscape)) // work → projects
	if m.level != levelProjects {
		t.Fatalf("after 2nd esc, level = %d, want projects", m.level)
	}
}

// TestCyrillicDIssuesDiff proves keybinds fire under a Russian layout: the
// physical D key reports "в", which must behave like "d".
func TestCyrillicDIssuesDiff(t *testing.T) {
	m := ready(t)
	_, cmd := step(m, keyText("в"))
	if cmd == nil {
		t.Fatal("Cyrillic 'в' (physical d) should issue a diff command like 'd'")
	}
}

// TestDetailPaneShowsSummary is the regression guard for the empty-detail bug:
// a ready model must render the selected session's summary in the detail pane.
func TestDetailPaneShowsSummary(t *testing.T) {
	m := New(config.Default())
	m, _ = step(m, tea.WindowSizeMsg{Width: 140, Height: 40})
	projs := []model.Project{{
		Name: "alpha", Path: "/r/alpha",
		Worktrees: []model.Worktree{{Path: "/r/alpha", Branch: "main", IsPrimary: true}},
		WorkSessions: []model.Session{{
			Harness: "claude", Name: "a1", SessionID: "s1", Live: true,
			Status: model.StatusBusy, Branch: "main", WorktreePath: "/r/alpha",
			Summary: "REGRESSION_SUMMARY_MARKER",
		}},
	}}
	m, _ = step(m, fleetMsg{projects: projs})
	if out := m.render(); !strings.Contains(out, "REGRESSION_SUMMARY_MARKER") {
		t.Errorf("detail pane should show the session summary; render:\n%s", out)
	}
}

// TestEmptyDetailNotBlank guards that a project with no sessions still renders a
// helpful placeholder rather than a blank pane.
func TestEmptyDetailNotBlank(t *testing.T) {
	m := New(config.Default())
	m, _ = step(m, tea.WindowSizeMsg{Width: 140, Height: 40})
	projs := []model.Project{{Name: "empty", Path: "/r/empty",
		Worktrees: []model.Worktree{{Path: "/r/empty", Branch: "main", IsPrimary: true}}}}
	m, _ = step(m, fleetMsg{projects: projs})
	if out := m.render(); !strings.Contains(out, "nothing selected in this tab") {
		t.Errorf("empty project detail should hint about the current tab; render:\n%s", out)
	}
}
