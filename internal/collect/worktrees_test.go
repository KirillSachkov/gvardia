package collect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseWorktreesMulti(t *testing.T) {
	wts := parseWorktrees(readFixture(t, "worktree_list_multi.txt"))
	if len(wts) != 3 {
		t.Fatalf("got %d worktrees, want 3", len(wts))
	}
	if !wts[0].IsPrimary {
		t.Error("first worktree should be primary")
	}
	if wts[1].IsPrimary || wts[2].IsPrimary {
		t.Error("only the first worktree should be primary")
	}
	wantBranches := []string{
		"codex/599-global-full-access",
		"fix/telegram-double-plan-welcome",
		"feat/675-s2",
	}
	for i, want := range wantBranches {
		if wts[i].Branch != want {
			t.Errorf("worktree[%d].Branch = %q, want %q", i, wts[i].Branch, want)
		}
	}
	if wts[0].Path != "/Users/dev/code/education-platform" {
		t.Errorf("primary path = %q", wts[0].Path)
	}
}

func TestParseWorktreesDetached(t *testing.T) {
	wts := parseWorktrees(readFixture(t, "worktree_list_detached.txt"))
	if len(wts) != 2 {
		t.Fatalf("got %d worktrees, want 2", len(wts))
	}
	if wts[1].Branch != "" {
		t.Errorf("detached worktree Branch = %q, want empty", wts[1].Branch)
	}
}

func TestDiscoverRepos(t *testing.T) {
	root := t.TempDir()
	mkRepo(t, filepath.Join(root, "a"))                   // depth 1
	mkRepo(t, filepath.Join(root, "group", "b"))          // depth 2
	mkRepo(t, filepath.Join(root, "a", "nested"))         // inside a repo — must be skipped
	mkRepo(t, filepath.Join(root, "1", "2", "3", "deep")) // depth 4 — beyond maxDepth
	mustMkdirAll(t, filepath.Join(root, "plain", "dir"))  // no .git

	got := DiscoverRepos([]string{root})

	want := map[string]bool{
		filepath.Join(root, "a"):          true,
		filepath.Join(root, "group", "b"): true,
	}
	if len(got) != len(want) {
		t.Fatalf("DiscoverRepos = %v, want keys %v", got, want)
	}
	for _, r := range got {
		if !want[r] {
			t.Errorf("unexpected repo %q", r)
		}
	}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func mkRepo(t *testing.T, dir string) {
	t.Helper()
	mustMkdirAll(t, filepath.Join(dir, ".git"))
}

func mustMkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
}
