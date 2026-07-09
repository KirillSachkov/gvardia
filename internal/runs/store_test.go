package runs

import (
	"os"
	"path/filepath"
	"strings"
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

func TestStoreRootKeepsRunsOutsideProjectAndFiltersByProject(t *testing.T) {
	root := t.TempDir()
	projectA := filepath.Join(t.TempDir(), "alpha")
	projectB := filepath.Join(t.TempDir(), "beta")
	for _, project := range []string{projectA, projectB} {
		if err := os.MkdirAll(project, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	ids := []string{"run-a", "run-b"}
	store := Store{Root: root, NewID: func() string {
		id := ids[0]
		ids = ids[1:]
		return id
	}}
	runA, err := store.Create(projectA, CreateInput{Project: "alpha", TaskTitle: "A", Runner: "codex", Tool: "codex", Prompt: "A"})
	if err != nil {
		t.Fatalf("Create A: %v", err)
	}
	if _, err := store.Create(projectB, CreateInput{Project: "beta", TaskTitle: "B", Runner: "codex", Tool: "codex", Prompt: "B"}); err != nil {
		t.Fatalf("Create B: %v", err)
	}

	if want := filepath.Join(root, "runs", "run-a"); runA.Dir() != want {
		t.Fatalf("run dir = %q, want %q", runA.Dir(), want)
	}
	if strings.HasPrefix(runA.Dir(), projectA) {
		t.Fatalf("run dir %q must not be inside project %q", runA.Dir(), projectA)
	}

	got, err := store.LoadProject(projectA)
	if err != nil {
		t.Fatalf("LoadProject A: %v", err)
	}
	if len(got) != 1 || got[0].ID != "run-a" {
		t.Fatalf("LoadProject A = %+v, want only run-a", got)
	}
}
