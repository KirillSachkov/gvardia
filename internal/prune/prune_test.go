package prune

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/KirillSachkov/gvardia/internal/collect"
	"github.com/KirillSachkov/gvardia/internal/config"
	"github.com/KirillSachkov/gvardia/internal/model"
)

func TestClassify(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	base := "main"
	merged := map[string]bool{"done": true, "main": true}
	recent := now.Add(-24 * time.Hour)
	old := now.Add(-40 * 24 * time.Hour)

	cases := []struct {
		name string
		wt   model.Worktree
		days int
		want Class
	}{
		{"primary", model.Worktree{IsPrimary: true, Branch: "main"}, 30, Primary},
		{"dirty beats merged", model.Worktree{Branch: "done", Dirty: true, LastCommit: recent}, 30, Dirty},
		{"merged", model.Worktree{Branch: "done", LastCommit: recent}, 30, Merged},
		{"stale", model.Worktree{Branch: "wip", LastCommit: old}, 30, Stale},
		{"active", model.Worktree{Branch: "wip", LastCommit: recent}, 30, Active},
		{"base branch not merged-flagged", model.Worktree{Branch: "main", LastCommit: recent}, 30, Active},
		{"stale disabled", model.Worktree{Branch: "wip", LastCommit: old}, 0, Active},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := classify("proj", "/primary", base, c.wt, merged, now, c.days)
			if got.Class != c.want {
				t.Errorf("Class = %q, want %q", got.Class, c.want)
			}
		})
	}
}

func TestRemovable(t *testing.T) {
	if !(Item{Class: Merged}).Removable(false) {
		t.Error("merged should be removable")
	}
	if (Item{Class: Stale}).Removable(false) {
		t.Error("stale should not be removable without includeStale")
	}
	if !(Item{Class: Stale}).Removable(true) {
		t.Error("stale should be removable with includeStale")
	}
	for _, c := range []Class{Primary, Dirty, Active} {
		if (Item{Class: c}).Removable(true) {
			t.Errorf("%s must never be removable", c)
		}
	}
}

func TestPlanAndRemoveIntegration(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	repo := filepath.Join(root, "repo")

	git(t, root, "init", "-b", "main", "repo")
	git(t, repo, "config", "user.email", "t@e.com")
	git(t, repo, "config", "user.name", "T")
	writeFile(t, filepath.Join(repo, "f.txt"), "1\n")
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-m", "init")

	// A merged worktree: branch, commit, merged back into main.
	git(t, repo, "worktree", "add", "-b", "done", filepath.Join(root, "done"))
	writeFile(t, filepath.Join(root, "done", "d.txt"), "d\n")
	git(t, filepath.Join(root, "done"), "add", ".")
	git(t, filepath.Join(root, "done"), "commit", "-m", "done work")
	git(t, repo, "merge", "done")

	// An active worktree: branch + commit, not merged.
	git(t, repo, "worktree", "add", "-b", "wip", filepath.Join(root, "wip"))
	writeFile(t, filepath.Join(root, "wip", "w.txt"), "w\n")
	git(t, filepath.Join(root, "wip"), "add", ".")
	git(t, filepath.Join(root, "wip"), "commit", "-m", "wip work")

	cfg := config.Default()
	cfg.Roots = []string{root}
	ctx := context.Background()

	plan, err := Plan(ctx, collect.Git{}, cfg, Options{StaleDays: 30})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	byBranch := map[string]Item{}
	for _, it := range plan {
		byBranch[it.Branch] = it
	}
	if byBranch["main"].Class != Primary {
		t.Errorf("main class = %q, want primary", byBranch["main"].Class)
	}
	if byBranch["done"].Class != Merged {
		t.Errorf("done class = %q, want merged", byBranch["done"].Class)
	}
	if byBranch["wip"].Class != Active {
		t.Errorf("wip class = %q, want active", byBranch["wip"].Class)
	}

	if err := Remove(ctx, collect.Git{}, byBranch["done"]); err != nil {
		t.Fatalf("Remove(done): %v", err)
	}
	plan2, _ := Plan(ctx, collect.Git{}, cfg, Options{StaleDays: 30})
	for _, it := range plan2 {
		if it.Branch == "done" {
			t.Error("done worktree should be gone after Remove")
		}
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
