package collect

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/KirillSachkov/gvardia/internal/model"
)

// status is the parsed result of `git status --porcelain=v2 --branch`.
type status struct {
	Dirty      bool
	Ahead      int
	Behind     int
	HasCommits bool
}

// parseStatus reads porcelain-v2 status output. Dirty is true if any tracked
// change or untracked file is present; Ahead/Behind come from the branch.ab
// header (absent without an upstream); HasCommits is false for a fresh repo.
func parseStatus(data []byte) status {
	s := status{HasCommits: true}
	for _, line := range strings.Split(string(data), "\n") {
		switch {
		case line == "":
			continue
		case strings.HasPrefix(line, "# branch.oid "):
			if strings.TrimSpace(strings.TrimPrefix(line, "# branch.oid ")) == "(initial)" {
				s.HasCommits = false
			}
		case strings.HasPrefix(line, "# branch.ab "):
			// "# branch.ab +2 -1" — Atoi accepts the leading +/-.
			if f := strings.Fields(line); len(f) == 4 {
				if a, err := strconv.Atoi(f[2]); err == nil {
					s.Ahead = a
				}
				if b, err := strconv.Atoi(f[3]); err == nil {
					s.Behind = -b
				}
			}
		case strings.HasPrefix(line, "#"):
			// other header line — ignore
		default:
			// a change entry (1/2/u/?) — the tree is dirty
			s.Dirty = true
		}
	}
	return s
}

// Enrich fills a worktree's status, ahead/behind, and last-commit time via git.
// A per-worktree git failure leaves the corresponding fields at their zero value
// rather than aborting; baseBranch is recorded as given.
func Enrich(ctx context.Context, runner Runner, wt model.Worktree, baseBranch string) model.Worktree {
	wt.BaseBranch = baseBranch

	out, err := runner.Run(ctx, wt.Path, "status", "--porcelain=v2", "--branch")
	if err != nil {
		return wt
	}
	s := parseStatus(out)
	wt.Dirty = s.Dirty
	wt.Ahead = s.Ahead
	wt.Behind = s.Behind

	if s.HasCommits {
		if ct, err := runner.Run(ctx, wt.Path, "log", "-1", "--format=%ct"); err == nil {
			if sec, err := strconv.ParseInt(strings.TrimSpace(string(ct)), 10, 64); err == nil {
				wt.LastCommit = time.Unix(sec, 0)
			}
		}
	}
	return wt
}

// resolveBaseBranch turns a configured base ("auto" or "") into a concrete
// branch: the first of dev/main/master that exists, else "main".
func resolveBaseBranch(ctx context.Context, runner Runner, repo, configured string) string {
	if configured != "auto" && configured != "" {
		return configured
	}
	for _, b := range []string{"dev", "main", "master"} {
		if _, err := runner.Run(ctx, repo, "show-ref", "--verify", "--quiet", "refs/heads/"+b); err == nil {
			return b
		}
	}
	return "main"
}
