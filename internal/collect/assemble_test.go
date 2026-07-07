package collect

import (
	"context"
	"testing"

	"github.com/KirillSachkov/gvardia/internal/model"
)

func TestAssembleLive(t *testing.T) {
	projects := []model.Project{{
		Name: "p", Path: "/r/p",
		Worktrees: []model.Worktree{
			{Path: "/r/p", Branch: "feat/675-s3", BaseBranch: "main"},
			{Path: "/r/p/idle", Branch: "dev", BaseBranch: "main"},
		},
	}}
	sessions := []model.Session{
		{Harness: "claude", Name: "a1", SessionID: "s1", Cwd: "/r/p"},
		{Harness: "claude", Name: "a2", SessionID: "s2", Cwd: "/r/p"},
	}

	got := AssembleLive(context.Background(), numstatRunner{out: "9\t1\tx.go\n"}, projects, sessions)
	work := got[0].WorkSessions
	if len(work) != 2 {
		t.Fatalf("got %d work sessions, want 2 (one per live session): %+v", len(work), work)
	}
	for _, s := range work {
		if !s.Live || s.Branch != "feat/675-s3" || s.WorktreePath != "/r/p" {
			t.Errorf("session not enriched: %+v", s)
		}
		if s.Task != "#675" {
			t.Errorf("Task = %q, want #675", s.Task)
		}
		if s.ChangeStat != (model.ChangeStat{Files: 1, Added: 9, Removed: 1}) {
			t.Errorf("ChangeStat = %+v", s.ChangeStat)
		}
	}
}

func TestMergeHistory(t *testing.T) {
	work := []model.Session{{SessionID: "s1", Live: true, Name: "live"}}
	hist := []model.Session{
		{SessionID: "s1", Name: "dup"},  // already live → dropped
		{SessionID: "s2", Name: "past"}, // kept
	}
	got := MergeHistory(work, hist)
	if len(got) != 2 {
		t.Fatalf("got %d, want 2 (dedup s1)", len(got))
	}
	if !got[0].Live || got[1].SessionID != "s2" {
		t.Errorf("merge order/dedup wrong: %+v", got)
	}
}
