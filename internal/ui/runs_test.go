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
				Report:    "REPORT_READY",
				Telemetry: runs.TelemetryStatus{Phase: "verify", Summary: "Tests finished"},
				Events:    []runs.Event{{Type: "status", Message: "Started verification"}},
				RunArtifacts: []runs.RunArtifact{
					{Type: "plan", Title: "Implementation plan", Path: "artifacts/plan.md"},
					{Type: "report", Title: "Final report", Path: "report.md"},
				},
				UpdatedAt:  time.Now(),
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
	for _, want := range []string{"Objective", "Evidence", "Activity", "Tests finished", "REPORT_READY", "d open diff"} {
		if !strings.Contains(out, want) {
			t.Errorf("runs view missing %q; render:\n%s", want, out)
		}
	}
	if strings.Contains(out, "internal/ui/view.go") {
		t.Fatalf("changed-file list should not appear in artifacts pane:\n%s", out)
	}
	if !strings.Contains(out, "Build ops console") {
		t.Errorf("runs view should show task and report; render:\n%s", out)
	}
}

func TestAgentsTabDefaultsToGlobalAttentionQueue(t *testing.T) {
	m := readyWithRuns(t)
	m.runsByProject["/r/beta"] = []runs.Run{
		{ID: "run-running", Project: "beta", ProjectPath: "/r/beta", TaskTitle: "Running beta", Runner: "codex", Status: runs.StatusRunning, UpdatedAt: time.Now()},
		{ID: "run-failed", Project: "beta", ProjectPath: "/r/beta", TaskTitle: "Failed beta", Runner: "codex", Status: runs.StatusFailed, UpdatedAt: time.Now().Add(-time.Minute)},
	}
	m.rebuildSessions()

	if m.agentScopeProject {
		t.Fatal("agents scope should default to all projects")
	}
	if len(m.runList) != 3 {
		t.Fatalf("global queue runs = %d, want 3", len(m.runList))
	}
	if m.runList[0].Status != runs.StatusReview || m.runList[1].Status != runs.StatusFailed {
		t.Fatalf("queue order = %v, %v; want review then failed", m.runList[0].Status, m.runList[1].Status)
	}
	out := m.render()
	if !strings.Contains(out, "gvardia") || !strings.Contains(out, "beta") {
		t.Fatalf("global queue should show both project names:\n%s", out)
	}

	m, _ = step(m, keyText("s"))
	if !m.agentScopeProject || len(m.runList) != 1 || m.runList[0].Project != "alpha" {
		t.Fatalf("project scope = %v, runs = %+v; want selected alpha only", m.agentScopeProject, m.runList)
	}
}

func TestRunsViewSplitsSummaryReportAndArtifactsIntoSeparatePanes(t *testing.T) {
	m := readyWithRuns(t)
	out := m.render()
	for _, want := range []string{"Summary", "Report", "Artifacts", "Build ops console", "REPORT_READY", "Implementation plan"} {
		if !strings.Contains(out, want) {
			t.Fatalf("split detail layout missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Objective\nBuild ops console") {
		t.Fatalf("run detail should not be rendered as one raw block anymore:\n%s", out)
	}
}

func TestRunDetailSummarizesReportInsteadOfDumpingRawMarkdown(t *testing.T) {
	detail := runDetail(runs.Run{
		ID: "run-2", TaskTitle: "Readable report", Runner: "claude", Tool: "claude",
		Status: runs.StatusReview,
		Report: strings.Join([]string{
			"# Отчёт",
			"",
			"## TL;DR",
			"Short useful summary.",
			"",
			"## Raw transcript",
			"line 1",
			"line 2",
			"line 3",
			"line 4",
			"line 5",
			"line 6",
		}, "\n"),
	})

	if !strings.Contains(detail, "Report summary") {
		t.Fatalf("run detail should have a report summary section:\n%s", detail)
	}
	if !strings.Contains(detail, "Short useful summary.") {
		t.Fatalf("run detail should include the TL;DR content:\n%s", detail)
	}
	if strings.Contains(detail, "Raw transcript") || strings.Contains(detail, "line 6") {
		t.Fatalf("run detail should not dump the whole raw report:\n%s", detail)
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

func TestRunEnvironmentIncludesTelemetryPaths(t *testing.T) {
	run := runs.Run{
		ID:            "run-123",
		ProjectPath:   "/repo",
		MetaPath:      "/repo/.gvardia/runs/run-123/meta.json",
		PromptPath:    "/repo/.gvardia/runs/run-123/prompt.md",
		ReportPath:    "/repo/.gvardia/runs/run-123/report.md",
		StatusPath:    "/repo/.gvardia/runs/run-123/status.json",
		EventsPath:    "/repo/.gvardia/runs/run-123/events.jsonl",
		ArtifactsPath: "/repo/.gvardia/runs/run-123/artifacts.json",
		ArtifactsDir:  "/repo/.gvardia/runs/run-123/artifacts",
	}

	got := runEnvironment(run)
	for _, key := range []string{
		"GVARDIA_RUN_ID",
		"GVARDIA_RUN_DIR",
		"GVARDIA_PROMPT_PATH",
		"GVARDIA_REPORT_PATH",
		"GVARDIA_STATUS_PATH",
		"GVARDIA_EVENTS_PATH",
		"GVARDIA_ARTIFACTS_PATH",
		"GVARDIA_ARTIFACTS_DIR",
	} {
		if got[key] == "" {
			t.Fatalf("runEnvironment missing %s: %+v", key, got)
		}
	}
}

func testConfig() config.Config {
	return config.Default()
}
