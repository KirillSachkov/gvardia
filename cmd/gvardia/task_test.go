package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/KirillSachkov/gvardia/internal/tasks"
)

func TestRunTaskCreatesAndUpdatesStandaloneTask(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	configPath := filepath.Join(t.TempDir(), "missing.toml")

	if err := runTask([]string{
		"create", "--id", "ops-1", "--title", "Reliable launch",
		"--project", "gvardia", "--body", "Fix launch health.",
	}, configPath); err != nil {
		t.Fatalf("runTask create: %v", err)
	}
	if err := runTask([]string{"update", "--id", "ops-1", "--status", "active"}, configPath); err != nil {
		t.Fatalf("runTask update: %v", err)
	}

	got := tasks.LoadGvardia(context.Background(), filepath.Join(dataHome, "gvardia"))
	if len(got) != 1 {
		t.Fatalf("tasks = %d, want 1", len(got))
	}
	if got[0].Title != "Reliable launch" || got[0].Status != "active" || got[0].Body != "Fix launch health." {
		t.Errorf("task = %+v, want preserved title/body and active status", got[0])
	}
}

func TestRunTaskUpdateRequiresExistingID(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	err := runTask([]string{"update", "--id", "missing", "--status", "done"}, filepath.Join(t.TempDir(), "missing.toml"))
	if err == nil {
		t.Fatal("runTask update missing error = nil, want error")
	}
}
