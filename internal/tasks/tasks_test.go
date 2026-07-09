package tasks

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/KirillSachkov/gvardia/internal/model"
)

func writeTask(t *testing.T, brain, col, name, front string) {
	t.Helper()
	dir := filepath.Join(brain, "tasks", col)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\n" + front + "---\n\nsome body text\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadReadsKanban(t *testing.T) {
	brain := t.TempDir()
	writeTask(t, brain, "active", "a.md",
		"id: \"#675\"\ntitle: Fix payment bug\nstatus: active\nproject: education-platform\n")
	writeTask(t, brain, "inbox", "b.md", "title: Research thing\n")
	writeTask(t, brain, "done", "c.md", "id: done-1\ntitle: Shipped\n")

	got := Load(context.Background(), brain)
	if len(got) != 3 {
		t.Fatalf("Load returned %d tasks, want 3: %+v", len(got), got)
	}

	byTitle := map[string]model.Task{}
	for _, tk := range got {
		byTitle[tk.Title] = tk
	}

	a := byTitle["Fix payment bug"]
	if a.ID != "#675" || a.Status != "active" || a.Project != "education-platform" {
		t.Errorf("active task parsed wrong: %+v", a)
	}
	b := byTitle["Research thing"]
	if b.Status != "inbox" || b.ID != "b" { // ID falls back to the slug
		t.Errorf("inbox task parsed wrong: %+v", b)
	}
	if byTitle["Shipped"].Status != "done" {
		t.Errorf("done task status = %q, want done", byTitle["Shipped"].Status)
	}
}

func TestLoadMissingBrainIsEmpty(t *testing.T) {
	if got := Load(context.Background(), filepath.Join(t.TempDir(), "nope")); len(got) != 0 {
		t.Errorf("missing brain should yield no tasks, got %+v", got)
	}
}

func TestLinkTasksByBranchRef(t *testing.T) {
	projects := []model.Project{{
		Name:         "edu",
		WorkSessions: []model.Session{{Harness: "claude", Task: "#675", Branch: "feat/675-x"}},
	}}
	LinkTasks(projects, []model.Task{{ID: "#675", Title: "Fix payment bug", Status: "active"}})
	if got := projects[0].WorkSessions[0].Task; got != "Fix payment bug" {
		t.Errorf("linked task = %q, want the task title", got)
	}
}

func TestLinkTasksIDWithoutHash(t *testing.T) {
	projects := []model.Project{{
		WorkSessions: []model.Session{{Task: "#42"}},
	}}
	LinkTasks(projects, []model.Task{{ID: "42", Title: "No-hash task"}})
	if got := projects[0].WorkSessions[0].Task; got != "No-hash task" {
		t.Errorf("linked task = %q, want No-hash task", got)
	}
}

func TestLoadLocalReadsProjectTasksWithBody(t *testing.T) {
	project := t.TempDir()
	dir := filepath.Join(project, ".gvardia", "tasks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nid: local-1\ntitle: Build ops console\nstatus: active\n---\n\nImplement local runs.\n"
	if err := os.WriteFile(filepath.Join(dir, "local-1.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	got := LoadLocal(context.Background(), project)
	if len(got) != 1 {
		t.Fatalf("LoadLocal returned %d tasks, want 1: %+v", len(got), got)
	}
	task := got[0]
	if task.ID != "local-1" || task.Title != "Build ops console" || task.Status != "active" {
		t.Errorf("local task metadata = %+v, want parsed frontmatter", task)
	}
	if task.Project != filepath.Base(project) || task.Source != "local" {
		t.Errorf("local task project/source = %q/%q, want %q/local", task.Project, task.Source, filepath.Base(project))
	}
	if task.Body != "Implement local runs." {
		t.Errorf("local task body = %q, want body text", task.Body)
	}
}

func TestCreateLocalWritesReloadableTask(t *testing.T) {
	project := t.TempDir()

	task, err := CreateLocal(project, model.Task{
		ID:     "ops-console",
		Title:  "Build ops console",
		Status: "inbox",
		Body:   "Make gvardia launch local agent runs.",
	})
	if err != nil {
		t.Fatalf("CreateLocal: %v", err)
	}
	if task.Path == "" {
		t.Fatal("CreateLocal returned empty Path")
	}

	got := LoadLocal(context.Background(), project)
	if len(got) != 1 {
		t.Fatalf("LoadLocal returned %d tasks, want 1", len(got))
	}
	if got[0].ID != "ops-console" || got[0].Body != "Make gvardia launch local agent runs." {
		t.Errorf("reloaded task = %+v, want written task", got[0])
	}
}
