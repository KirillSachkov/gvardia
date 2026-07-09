package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/KirillSachkov/gvardia/internal/runs"
)

func TestRunTelemetryCommandsWriteRunFiles(t *testing.T) {
	project := t.TempDir()
	store := runs.Store{NewID: func() string { return "run-123" }}
	created, err := store.Create(project, runs.CreateInput{Project: "alpha", TaskTitle: "Fix", Runner: "claude", Tool: "claude", Prompt: "Prompt"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	runDir := created.Dir()

	if err := run([]string{"run", "status", "--run-dir", runDir, "--state", "running", "--phase", "inspect", "--summary", "Inspecting"}); err != nil {
		t.Fatalf("run status: %v", err)
	}
	if err := run([]string{"run", "event", "--run-dir", runDir, "--type", "status", "--message", "Started"}); err != nil {
		t.Fatalf("run event: %v", err)
	}
	artifact := filepath.Join(t.TempDir(), "plan.md")
	if err := os.WriteFile(artifact, []byte("Plan"), 0o644); err != nil {
		t.Fatalf("write artifact source: %v", err)
	}
	if err := run([]string{"run", "artifact", "--run-dir", runDir, "--type", "plan", "--title", "Plan", "--file", artifact}); err != nil {
		t.Fatalf("run artifact: %v", err)
	}
	report := filepath.Join(t.TempDir(), "report.md")
	if err := os.WriteFile(report, []byte("Report"), 0o644); err != nil {
		t.Fatalf("write report source: %v", err)
	}
	if err := run([]string{"run", "report", "--run-dir", runDir, "--file", report}); err != nil {
		t.Fatalf("run report: %v", err)
	}

	got, err := store.LoadProject(project)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("LoadProject returned %d runs, want 1", len(got))
	}
	loaded := got[0]
	if loaded.Telemetry.Phase != "inspect" || loaded.Telemetry.Summary != "Inspecting" {
		t.Fatalf("telemetry = %+v", loaded.Telemetry)
	}
	if len(loaded.Events) != 1 || loaded.Events[0].Message != "Started" {
		t.Fatalf("events = %+v", loaded.Events)
	}
	if len(loaded.RunArtifacts) != 2 {
		t.Fatalf("artifacts = %+v, want plan + report", loaded.RunArtifacts)
	}
	if loaded.Report != "Report" {
		t.Fatalf("report = %q", loaded.Report)
	}
}

func TestRunTelemetryCommandsRequireRunDir(t *testing.T) {
	t.Setenv("GVARDIA_RUN_DIR", "")
	if err := run([]string{"run", "event", "--message", "Started"}); err == nil {
		t.Fatal("run event without --run-dir or GVARDIA_RUN_DIR should fail")
	}
}
