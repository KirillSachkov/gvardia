package runs

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCreateInitializesTelemetryFiles(t *testing.T) {
	project := t.TempDir()
	store := Store{
		Now:   func() time.Time { return time.Unix(100, 0).UTC() },
		NewID: func() string { return "run-123" },
	}

	run, err := store.Create(project, CreateInput{Project: "alpha", TaskTitle: "Fix bug", Runner: "claude", Tool: "claude", Prompt: "Prompt"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	for _, path := range []string{run.StatusPath, run.EventsPath, run.ArtifactsPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected telemetry file %s: %v", path, err)
		}
	}
	if _, err := os.Stat(run.ArtifactsDir); err != nil {
		t.Fatalf("expected artifacts dir %s: %v", run.ArtifactsDir, err)
	}
	if run.Telemetry.State != StatusPending || run.Telemetry.Summary == "" {
		t.Fatalf("initial telemetry = %+v, want pending summary", run.Telemetry)
	}
}

func TestTelemetryStatusEventsAndArtifactsReload(t *testing.T) {
	project := t.TempDir()
	store := Store{
		Now:   func() time.Time { return time.Unix(100, 0).UTC() },
		NewID: func() string { return "run-123" },
	}
	run, err := store.Create(project, CreateInput{Project: "alpha", TaskTitle: "Fix bug", Runner: "codex", Tool: "codex", Prompt: "Prompt"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	store.Now = func() time.Time { return time.Unix(200, 0).UTC() }
	if err := store.WriteStatus(run.Dir(), TelemetryStatus{
		State:       StatusRunning,
		Phase:       "tests",
		Summary:     "Running verification",
		NeedsReview: false,
	}); err != nil {
		t.Fatalf("WriteStatus: %v", err)
	}
	if err := store.AppendEvent(run.Dir(), Event{Type: "status", Message: "Started tests"}); err != nil {
		t.Fatalf("AppendEvent: %v", err)
	}
	artifactPath := filepath.Join(t.TempDir(), "plan.md")
	if err := os.WriteFile(artifactPath, []byte("Plan"), 0o644); err != nil {
		t.Fatalf("write source artifact: %v", err)
	}
	if _, err := store.SaveArtifact(run.Dir(), ArtifactInput{Type: "plan", Title: "Plan", File: artifactPath}); err != nil {
		t.Fatalf("SaveArtifact: %v", err)
	}
	if err := store.WriteReport(run.Dir(), []byte("Final report")); err != nil {
		t.Fatalf("WriteReport: %v", err)
	}

	got, err := store.LoadProject(project)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("LoadProject returned %d runs, want 1", len(got))
	}
	loaded := got[0]
	if loaded.Telemetry.Phase != "tests" || loaded.Telemetry.Summary != "Running verification" {
		t.Fatalf("loaded telemetry = %+v", loaded.Telemetry)
	}
	if loaded.Status != StatusReview {
		t.Fatalf("loaded status = %s, want review after report", loaded.Status)
	}
	if len(loaded.Events) != 1 || loaded.Events[0].Message != "Started tests" {
		t.Fatalf("loaded events = %+v", loaded.Events)
	}
	if len(loaded.RunArtifacts) != 2 {
		t.Fatalf("loaded artifacts = %+v, want plan + report", loaded.RunArtifacts)
	}
	if loaded.Report != "Final report" {
		t.Fatalf("loaded report = %q", loaded.Report)
	}
}
