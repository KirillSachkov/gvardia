package collect

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/KirillSachkov/gvardia/internal/config"
)

// TestCollectIntegration exercises the whole collector against a real git repo
// with a linked worktree in a temp dir. Skipped where git is unavailable.
func TestCollectIntegration(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	repo := filepath.Join(root, "myrepo")

	git(t, root, "init", "-b", "main", "myrepo")
	git(t, repo, "config", "user.email", "test@example.com")
	git(t, repo, "config", "user.name", "Test")
	writeFile(t, filepath.Join(repo, "README.md"), "hello\n")
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-m", "init")

	// A linked worktree on a new branch, plus an untracked file dirtying primary.
	git(t, repo, "worktree", "add", "-b", "feature", filepath.Join(root, "wt"))
	writeFile(t, filepath.Join(repo, "scratch.txt"), "wip\n")

	cfg := config.Default()
	cfg.Roots = []string{root}

	projects, err := Collect(context.Background(), Git{}, cfg)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("got %d projects, want 1: %+v", len(projects), projects)
	}
	p := projects[0]
	if p.Name != "myrepo" {
		t.Errorf("project Name = %q, want myrepo", p.Name)
	}
	if len(p.Worktrees) != 2 {
		t.Fatalf("got %d worktrees, want 2", len(p.Worktrees))
	}

	byBranch := map[string]int{} // branch -> index
	for i, w := range p.Worktrees {
		byBranch[w.Branch] = i
	}
	mainIdx, ok := byBranch["main"]
	if !ok {
		t.Fatalf("no main worktree in %+v", p.Worktrees)
	}
	featIdx, ok := byBranch["feature"]
	if !ok {
		t.Fatalf("no feature worktree in %+v", p.Worktrees)
	}

	if !p.Worktrees[mainIdx].IsPrimary {
		t.Error("main worktree should be primary")
	}
	if !p.Worktrees[mainIdx].Dirty {
		t.Error("main worktree should be dirty (untracked scratch.txt)")
	}
	if p.Worktrees[featIdx].Dirty {
		t.Error("feature worktree should be clean")
	}
	if p.Worktrees[mainIdx].BaseBranch != "main" {
		t.Errorf("BaseBranch = %q, want main (auto resolves to main)", p.Worktrees[mainIdx].BaseBranch)
	}
	if p.Worktrees[mainIdx].LastCommit.IsZero() {
		t.Error("main worktree LastCommit should be set")
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
