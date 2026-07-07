package ui

import (
	"strings"
	"testing"

	"github.com/KirillSachkov/gvardia/internal/model"
)

func TestWorktreeViewToggles(t *testing.T) {
	m := ready(t)
	if m.worktreeView {
		t.Fatal("should start in the agents view")
	}
	m, _ = step(m, keyText("w"))
	if !m.worktreeView {
		t.Fatal("w should switch to the worktree view")
	}
	m, _ = step(m, keyText("w"))
	if m.worktreeView {
		t.Fatal("w again should switch back to the agents view")
	}
}

func TestWorktreeViewShowsAllWorktrees(t *testing.T) {
	m := ready(t)
	m, _ = step(m, keyText("j")) // move to beta (2 worktrees)
	m, _ = step(m, keyText("w"))
	if got := len(m.sessions.Rows()); got != 2 {
		t.Errorf("beta worktree rows = %d, want 2", got)
	}
	if got := len(m.worktreeList); got != 2 {
		t.Errorf("worktreeList = %d, want 2", got)
	}
	if w := m.selectedWorktree(); w == nil {
		t.Error("a worktree should be selected in the worktree view")
	}
}

// TestWorktreeRowAgentMarker checks the agent ↔ worktree link surfaces in a row.
func TestWorktreeRowAgentMarker(t *testing.T) {
	withAgent := model.Worktree{Path: "/r/beta/wt", Branch: "feat/675-x",
		Sessions: []model.Session{{Harness: "codex"}}}
	if row := worktreeRow2(withAgent); !strings.Contains(strings.Join(row, " "), "codex") {
		t.Errorf("worktree with an agent should show it; got %v", row)
	}
	bare := model.Worktree{Path: "/r/beta/wt", Branch: "feat/675-x"}
	if row := worktreeRow2(bare); strings.Contains(strings.Join(row, " "), "codex") {
		t.Errorf("worktree without an agent should not show one; got %v", row)
	}
}

// TestWorktreeViewDiffAndDetail proves the worktree selection drives detail/diff.
func TestWorktreeViewDiffAndDetail(t *testing.T) {
	m := ready(t)
	m, _ = step(m, keyText("j")) // beta
	m, _ = step(m, keyText("w")) // worktree view
	if _, cmd := step(m, keyText("d")); cmd == nil {
		t.Error("d in the worktree view should issue a diff command")
	}
	if h, w := m.currentDetail(); h == "" || w == nil {
		t.Errorf("worktree view should have a detail header (%q) and worktree (%v)", h, w)
	}
}
