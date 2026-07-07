package collect

import (
	"context"

	"github.com/KirillSachkov/gvardia/internal/model"
)

// AssembleLive attaches live sessions to worktrees (via Join) and builds each
// project's flat WorkSessions list — one entry per live session, carrying the
// matched worktree's branch, path, inferred task, and change stat. Change stats
// are computed once per worktree (only worktrees that actually hold a session).
func AssembleLive(ctx context.Context, runner Runner, projects []model.Project, sessions []model.Session) []model.Project {
	projects = Join(projects, sessions)

	statCache := make(map[string]model.ChangeStat)
	for pi := range projects {
		p := &projects[pi]
		var work []model.Session
		for wi := range p.Worktrees {
			wt := &p.Worktrees[wi]
			if len(wt.Sessions) == 0 {
				continue
			}
			stat, ok := statCache[wt.Path]
			if !ok {
				stat = ChangeStatFor(ctx, runner, wt.Path, wt.BaseBranch)
				statCache[wt.Path] = stat
			}
			wt.ChangeStat = stat
			for _, s := range wt.Sessions {
				s.Live = true
				s.Branch = wt.Branch
				s.WorktreePath = wt.Path
				s.Task = TaskFromBranch(wt.Branch)
				s.ChangeStat = stat
				work = append(work, s)
			}
		}
		p.WorkSessions = work
	}
	return projects
}

// MergeHistory appends history sessions after the live ones, dropping any whose
// SessionID already appears live. Live order is preserved; history stays in the
// order given (newest-first from history.Recent).
func MergeHistory(work, hist []model.Session) []model.Session {
	seen := make(map[string]bool, len(work))
	out := make([]model.Session, 0, len(work)+len(hist))
	for _, s := range work {
		out = append(out, s)
		if s.SessionID != "" {
			seen[s.SessionID] = true
		}
	}
	for _, h := range hist {
		if h.SessionID != "" && seen[h.SessionID] {
			continue
		}
		out = append(out, h)
	}
	return out
}
