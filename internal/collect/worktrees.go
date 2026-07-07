package collect

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/KirillSachkov/gvardia/internal/model"
)

// maxDepth bounds how deep DiscoverRepos descends below each root looking for a
// ".git" entry. Repos are typically 1–2 levels down; nested linked worktrees are
// found via `git worktree list`, not by walking, so a shallow scan suffices.
const maxDepth = 3

// DiscoverRepos returns directories under the given roots that contain a ".git"
// entry (a repo checkout or a linked worktree). It does not descend into a repo
// once found, and tolerates unreadable directories. Roots must already be
// absolute (config expands "~"). Results may map to the same primary worktree;
// the caller collapses them via `git worktree list`.
func DiscoverRepos(roots []string) []string {
	var repos []string
	seen := make(map[string]bool)
	for _, root := range roots {
		root = filepath.Clean(root)
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || !d.IsDir() {
				return nil //nolint:nilerr // skip unreadable dirs, keep scanning
			}
			if depthBelow(root, path) > maxDepth {
				return fs.SkipDir
			}
			if hasGit(path) {
				if !seen[path] {
					seen[path] = true
					repos = append(repos, path)
				}
				return fs.SkipDir
			}
			return nil
		})
	}
	return repos
}

func depthBelow(root, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	return strings.Count(rel, string(os.PathSeparator)) + 1
}

func hasGit(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

// parseWorktrees turns `git worktree list --porcelain` output into worktrees.
// The first entry is the primary; a "detached" worktree gets an empty Branch.
func parseWorktrees(data []byte) []model.Worktree {
	var wts []model.Worktree
	var cur model.Worktree
	inBlock := false

	flush := func() {
		if inBlock {
			wts = append(wts, cur)
			cur = model.Worktree{}
			inBlock = false
		}
	}

	for _, line := range strings.Split(string(data), "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			flush()
			cur.Path = strings.TrimPrefix(line, "worktree ")
			inBlock = true
		case strings.HasPrefix(line, "branch "):
			cur.Branch = strings.TrimPrefix(strings.TrimPrefix(line, "branch "), "refs/heads/")
		case line == "":
			flush()
		}
	}
	flush()

	if len(wts) > 0 {
		wts[0].IsPrimary = true
	}
	return wts
}
