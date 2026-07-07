package collect

import (
	"context"
	"strconv"
	"strings"

	"github.com/KirillSachkov/gvardia/internal/model"
)

// ChangeStatFor computes a worktree's diff stat against its base branch. It is
// called only for worktrees that carry a session (a handful), not for every
// worktree in the fleet — a `git diff` per worktree would dominate collection.
func ChangeStatFor(ctx context.Context, runner Runner, path, base string) model.ChangeStat {
	rangeArg := "HEAD"
	if base != "" && base != "HEAD" {
		rangeArg = base + "...HEAD"
	}
	out, err := runner.Run(ctx, path, "diff", "--numstat", rangeArg)
	if err != nil {
		return model.ChangeStat{}
	}
	return parseNumstat(out)
}

// parseNumstat sums `git diff --numstat` output. Binary files (added/removed
// shown as "-") count toward Files but not the line totals.
func parseNumstat(data []byte) model.ChangeStat {
	var st model.ChangeStat
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}
		f := strings.SplitN(line, "\t", 3)
		if len(f) < 3 {
			continue
		}
		st.Files++
		if a, err := strconv.Atoi(f[0]); err == nil {
			st.Added += a
		}
		if r, err := strconv.Atoi(f[1]); err == nil {
			st.Removed += r
		}
	}
	return st
}
