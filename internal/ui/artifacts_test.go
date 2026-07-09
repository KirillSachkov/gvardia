package ui

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/KirillSachkov/gvardia/internal/config"
	"github.com/KirillSachkov/gvardia/internal/model"
	"github.com/KirillSachkov/gvardia/internal/runs"
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
	if strings.Contains(out, "internal/ui/view.go") {
		t.Errorf("changed files must not be rendered as artifacts; render:\n%s", out)
	}
	if !strings.Contains(out, "1 files") || !strings.Contains(out, "d open diff") {
		t.Errorf("detail should show a compact changes summary; render:\n%s", out)
	}
}

func TestArtifactPathStaysInsideRunDirectory(t *testing.T) {
	runDir := t.TempDir()
	run := runs.Run{ID: "run-1", ProjectPath: "/repo", MetaPath: filepath.Join(runDir, "meta.json")}

	got, err := artifactPath(run, runs.RunArtifact{Path: "artifacts/plan.md"})
	if err != nil {
		t.Fatalf("artifactPath valid: %v", err)
	}
	if got != filepath.Join(runDir, "artifacts", "plan.md") {
		t.Fatalf("artifactPath = %q, want path inside run", got)
	}
	if _, err := artifactPath(run, runs.RunArtifact{Path: "../secret"}); err == nil {
		t.Fatal("artifactPath escape error = nil, want error")
	}
}

func TestArtifactBrowserNavigatesAndOpensSelection(t *testing.T) {
	m := readyWithRuns(t)
	m, _ = step(m, keyText("e"))
	if m.artifactBrowser == nil {
		t.Fatal("e should open the artifact browser")
	}
	if out := m.render(); !strings.Contains(out, "Run artifacts") || !strings.Contains(out, "Implementation plan") {
		t.Fatalf("artifact browser missing content:\n%s", out)
	}
	m, _ = step(m, keyText("j"))
	if m.artifactBrowser.cursor != 1 {
		t.Fatalf("artifact cursor = %d, want 1", m.artifactBrowser.cursor)
	}
	m, cmd := step(m, keyPress(tea.KeyEnter))
	if m.artifactBrowser != nil || cmd == nil {
		t.Fatal("enter should close browser and return an open command")
	}
}

func TestArtifactCommandPrefersCmuxMarkdown(t *testing.T) {
	lookup := func(name string) (string, error) {
		if name == "cmux" {
			return "/usr/local/bin/cmux", nil
		}
		return "", errors.New("missing")
	}
	cmd, interactive, err := artifactCommand("/tmp/report.md", config.Default(), lookup)
	if err != nil {
		t.Fatalf("artifactCommand: %v", err)
	}
	if interactive || filepath.Base(cmd.Args[0]) != "cmux" || !strings.Contains(strings.Join(cmd.Args, " "), "markdown open /tmp/report.md") {
		t.Fatalf("artifact command = %v, interactive=%v; want cmux markdown", cmd.Args, interactive)
	}
}
