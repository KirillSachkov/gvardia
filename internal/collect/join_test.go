package collect

import (
	"testing"

	"github.com/KirillSachkov/gvardia/internal/model"
)

func TestJoin(t *testing.T) {
	projects := []model.Project{{
		Name: "p",
		Path: "/root/p",
		Worktrees: []model.Worktree{
			{Path: "/root/p", Branch: "main", IsPrimary: true},
			{Path: "/root/p/.claude/worktrees/agent-x", Branch: "feat"},
		},
	}}

	sessions := []model.Session{
		{Harness: "claude", Cwd: "/root/p", Status: model.StatusBusy},
		// tmux for the same worktree — must be deduped (claude already there).
		{Harness: "tmux", Cwd: "/root/p", Status: model.StatusBusy},
		// A cwd inside the nested worktree — longest-prefix must pick agent-x, not /root/p.
		{Harness: "claude", Cwd: "/root/p/.claude/worktrees/agent-x/backend", Status: model.StatusIdle},
		// Orphan: matches no worktree.
		{Harness: "codex", Cwd: "/root/other", Status: model.StatusIdle},
	}

	got := Join(projects, sessions)

	wt1 := got[0].Worktrees[0]
	wt2 := got[0].Worktrees[1]
	if len(wt1.Sessions) != 1 || wt1.Sessions[0].Harness != "claude" {
		t.Errorf("wt1 sessions = %+v, want single claude (tmux deduped)", wt1.Sessions)
	}
	if len(wt2.Sessions) != 1 || wt2.Sessions[0].Cwd != "/root/p/.claude/worktrees/agent-x/backend" {
		t.Errorf("wt2 sessions = %+v, want the nested-cwd claude (longest prefix)", wt2.Sessions)
	}
	if got[0].LiveAgents != 2 {
		t.Errorf("LiveAgents = %d, want 2", got[0].LiveAgents)
	}
}

func TestJoinTmuxFillsEmptyWorktree(t *testing.T) {
	projects := []model.Project{{
		Name:      "p",
		Path:      "/root/p",
		Worktrees: []model.Worktree{{Path: "/root/p", Branch: "main"}},
	}}
	// Only a tmux session — it should land since nothing else claims the worktree.
	got := Join(projects, []model.Session{{Harness: "tmux", Cwd: "/root/p"}})
	if len(got[0].Worktrees[0].Sessions) != 1 {
		t.Errorf("want tmux session to fill empty worktree, got %+v", got[0].Worktrees[0].Sessions)
	}
}
