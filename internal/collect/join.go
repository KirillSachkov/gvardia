package collect

import (
	"os"
	"strings"

	"github.com/KirillSachkov/gvardia/internal/model"
)

// Join attaches sessions to worktrees by longest-prefix cwd match, mutating and
// returning projects with their Sessions populated and LiveAgents counted.
// First-class adapters (claude, codex) win over the tmux fallback: a tmux
// session is dropped if its worktree already has a session. Sessions that match
// no worktree are ignored.
func Join(projects []model.Project, sessions []model.Session) []model.Project {
	type wref struct {
		path string
		pi   int
		wi   int
	}
	var refs []wref
	for pi := range projects {
		for wi := range projects[pi].Worktrees {
			refs = append(refs, wref{projects[pi].Worktrees[wi].Path, pi, wi})
		}
	}

	attach := func(s model.Session, onlyEmpty bool) {
		best := -1
		for i, r := range refs {
			if !pathWithin(s.Cwd, r.path) {
				continue
			}
			if best == -1 || len(r.path) > len(refs[best].path) {
				best = i
			}
		}
		if best == -1 {
			return
		}
		wt := &projects[refs[best].pi].Worktrees[refs[best].wi]
		if onlyEmpty && len(wt.Sessions) > 0 {
			return
		}
		wt.Sessions = append(wt.Sessions, s)
	}

	// First-class adapters first, then tmux only where nothing landed.
	for _, s := range sessions {
		if s.Harness != "tmux" {
			attach(s, false)
		}
	}
	for _, s := range sessions {
		if s.Harness == "tmux" {
			attach(s, true)
		}
	}

	for pi := range projects {
		n := 0
		for _, wt := range projects[pi].Worktrees {
			n += len(wt.Sessions)
		}
		projects[pi].LiveAgents = n
	}
	return projects
}

// pathWithin reports whether cwd is base or lives beneath it.
func pathWithin(cwd, base string) bool {
	return cwd == base || strings.HasPrefix(cwd, base+string(os.PathSeparator))
}
