package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/KirillSachkov/gvardia/internal/config"
	"github.com/KirillSachkov/gvardia/internal/model"
	"github.com/KirillSachkov/gvardia/internal/runs"
)

func readyWithRuns(t *testing.T) Model {
	t.Helper()
	m := New(testConfig())
	m, _ = step(m, tea.WindowSizeMsg{Width: 150, Height: 42})
	m, _ = step(m, fleetMsg{
		projects: testProjects(),
		runs: map[string][]runs.Run{
			"/r/alpha": {{
				ID: "run-1", Project: "alpha", ProjectPath: "/r/alpha",
				TaskTitle: "Build ops console", Runner: "claude", Tool: "claude",
				Status: runs.StatusReview, TmuxTarget: "gvardia-run-1",
				WorktreePath: "/r/alpha-wt", Branch: "gvardia/run-1",
				Report: "REPORT_READY", UpdatedAt: time.Now(),
				ChangeStat: model.ChangeStat{Files: 1, Added: 5, Removed: 2},
				Artifacts:  []model.Artifact{{Status: "M", Path: "internal/ui/view.go"}},
			}},
		},
	})
	return m
}

func TestRunsViewShowsRunsAndReport(t *testing.T) {
	m := readyWithRuns(t)
	if !m.showingRuns() {
		t.Fatal("model should prefer runs view when selected project has runs")
	}
	if len(m.sessions.Rows()) != 1 {
		t.Fatalf("runs table rows = %d, want 1", len(m.sessions.Rows()))
	}
	out := m.render()
	if !strings.Contains(out, "Build ops console") || !strings.Contains(out, "REPORT_READY") || !strings.Contains(out, "internal/ui/view.go") {
		t.Errorf("runs view should show task and report; render:\n%s", out)
	}
}

func TestRunKillConfirmation(t *testing.T) {
	m := readyWithRuns(t)
	m, _ = step(m, keyText("k"))
	if m.confirm == nil {
		t.Fatal("k on selected run should open confirmation")
	}
	if !strings.Contains(m.confirm.message, "run-1") {
		t.Errorf("confirm message = %q, want run id", m.confirm.message)
	}
}

func TestOpenRunReportReturnsCommand(t *testing.T) {
	m := readyWithRuns(t)
	_, cmd := step(m, keyText("o"))
	if cmd == nil {
		t.Fatal("o on selected run should return report viewer command")
	}
}

func testConfig() config.Config {
	return config.Default()
}
