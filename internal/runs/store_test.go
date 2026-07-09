package runs

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreCreateWritesPromptAndMeta(t *testing.T) {
	project := t.TempDir()
	store := Store{
		Now:   func() time.Time { return time.Unix(100, 0).UTC() },
		NewID: func() string { return "run-123" },
	}

	run, err := store.Create(project, CreateInput{
		Project:      "alpha",
		TaskID:       "task-1",
		TaskTitle:    "Build console",
		Runner:       "claude",
		Tool:         "claude",
		WorktreePath: filepath.Join(project, ".gvardia", "worktrees", "run-123"),
		Branch:       "gvardia/run-123",
		Prompt:       "Task prompt",
		TmuxTarget:   "gvardia-run-123",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if run.ID != "run-123" || run.Status != StatusPending {
		t.Fatalf("run = %+v, want pending run-123", run)
	}
	if run.PromptPath == "" || run.MetaPath == "" || run.ReportPath == "" {
		t.Fatalf("run paths missing: %+v", run)
	}
	if data, err := os.ReadFile(run.PromptPath); err != nil || string(data) != "Task prompt" {
		t.Fatalf("prompt file = %q, %v; want Task prompt", data, err)
	}
	if _, err := os.Stat(run.MetaPath); err != nil {
		t.Fatalf("meta file missing: %v", err)
	}
}

func TestStoreUpdateAndLoadProject(t *testing.T) {
	project := t.TempDir()
	store := Store{Now: func() time.Time { return time.Unix(100, 0).UTC() }, NewID: func() string { return "run-123" }}
	run, err := store.Create(project, CreateInput{Project: "alpha", TaskID: "task-1", TaskTitle: "Build", Runner: "codex", Tool: "codex", Prompt: "Prompt"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	store.Now = func() time.Time { return time.Unix(200, 0).UTC() }
	run.Status = StatusRunning
	run.TmuxTarget = "gvardia-run-123"
	if err := store.Save(run); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := os.WriteFile(run.ReportPath, []byte("Finished work"), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}

	got, err := store.LoadProject(project)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("LoadProject returned %d runs, want 1", len(got))
	}
	if got[0].Status != StatusReview || got[0].TmuxTarget != "gvardia-run-123" || got[0].Report != "Finished work" {
		t.Errorf("loaded run = %+v, want saved status/target/report", got[0])
	}
}

func TestLoadProjectMissingStoreIsEmpty(t *testing.T) {
	got, err := (Store{}).LoadProject(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatalf("LoadProject missing: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("LoadProject missing returned %+v, want empty", got)
	}
}
