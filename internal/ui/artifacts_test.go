package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/KirillSachkov/gvardia/internal/config"
	"github.com/KirillSachkov/gvardia/internal/model"
)

func TestDetailShowsReportAndArtifacts(t *testing.T) {
	m := New(config.Default())
	m, _ = step(m, tea.WindowSizeMsg{Width: 140, Height: 40})
	projs := []model.Project{{
		Name: "alpha", Path: "/r/alpha",
		Worktrees: []model.Worktree{{Path: "/r/alpha", Branch: "main", IsPrimary: true}},
		WorkSessions: []model.Session{{
			Harness: "claude", Name: "a1", SessionID: "s1", Live: true,
			Status: model.StatusBusy, Branch: "main", WorktreePath: "/r/alpha",
			Summary: "session summary", Report: "REPORT_MARKER done",
			Artifacts: []model.Artifact{{Status: "M", Path: "internal/ui/view.go"}},
		}},
	}}
	m, _ = step(m, fleetMsg{projects: projs})

	out := m.render()
	if !strings.Contains(out, "REPORT_MARKER") {
		t.Errorf("detail should show the report; render:\n%s", out)
	}
	if !strings.Contains(out, "internal/ui/view.go") {
		t.Errorf("detail should list the changed file; render:\n%s", out)
	}
	if !strings.Contains(out, "artifacts (1)") {
		t.Errorf("detail should show the artifacts count; render:\n%s", out)
	}
}
