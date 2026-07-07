package ui

import (
	"os/exec"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/KirillSachkov/gvardia/internal/model"
)

func TestSessionExec(t *testing.T) {
	wt := func(s model.Session) model.Worktree {
		return model.Worktree{Path: "/w", Sessions: []model.Session{s}}
	}

	claude := sessionExec(wt(model.Session{Harness: "claude", SessionID: "abc"}), true)
	if claude == nil || !hasArgs(claude.Args, "--resume", "abc") {
		t.Errorf("claude exec args = %v", args(claude))
	}
	if claude != nil && claude.Dir != "/w" {
		t.Errorf("claude Dir = %q, want /w", claude.Dir)
	}

	codex := sessionExec(wt(model.Session{Harness: "codex", SessionID: "xyz"}), true)
	if codex == nil || !hasArgs(codex.Args, "resume", "xyz") {
		t.Errorf("codex exec args = %v", args(codex))
	}
	codexLast := sessionExec(wt(model.Session{Harness: "codex"}), true)
	if codexLast == nil || !hasArgs(codexLast.Args, "resume", "--last") {
		t.Errorf("codex without id should use --last, got %v", args(codexLast))
	}

	tm := sessionExec(wt(model.Session{Harness: "tmux", SessionID: "work"}), true)
	if tm == nil || !hasArgs(tm.Args, "attach", "-t", "work") {
		t.Errorf("tmux attach args = %v", args(tm))
	}
	if sessionExec(wt(model.Session{Harness: "tmux", SessionID: "work"}), false) != nil {
		t.Error("tmux resume (attach=false) should be nil")
	}
	if sessionExec(model.Worktree{Path: "/w"}, true) != nil {
		t.Error("no session → nil command")
	}
}

func TestKillConfirmFlow(t *testing.T) {
	m := ready(t)
	// alpha's worktree has no session → 'k' should banner, not open a confirm.
	m, _ = step(m, keyText("k"))
	if m.confirm != nil {
		t.Fatal("k with no session should not open a confirm")
	}

	m.projects[0].Worktrees[0].Sessions = []model.Session{{Harness: "claude", Name: "a1", PID: 4242}}
	m.rebuildSessions()
	m, _ = step(m, keyText("k"))
	if m.confirm == nil {
		t.Fatal("k with a killable session should open a confirm")
	}

	if m2, cmd := step(m, keyText("n")); m2.confirm != nil || cmd != nil {
		t.Error("n should cancel the confirm with no action")
	}
	m3, cmd := step(m, keyText("y"))
	if m3.confirm != nil {
		t.Error("y should clear the confirm")
	}
	if cmd == nil {
		t.Error("y should return the kill action command")
	}
}

func TestGCConfirm(t *testing.T) {
	m := ready(t)
	m, _ = step(m, keyText("g"))
	if m.confirm == nil {
		t.Fatal("g should open a gc confirm")
	}
	if _, cmd := step(m, keyText("y")); cmd == nil {
		t.Error("confirming gc should return an action command")
	}
}

func TestNewAgentPrompt(t *testing.T) {
	m := ready(t)
	m, _ = step(m, keyText("n"))
	if m.prompt == nil || m.prompt.harness != "claude" {
		t.Fatal("n should open the new-agent prompt defaulting to claude")
	}
	for _, r := range "foo" {
		m, _ = step(m, keyText(string(r)))
	}
	if m.prompt.input.Value() != "foo" {
		t.Errorf("prompt input = %q, want foo", m.prompt.input.Value())
	}
	m, _ = step(m, keyPress(tea.KeyTab))
	if m.prompt.harness != "codex" {
		t.Errorf("tab should switch harness to codex, got %q", m.prompt.harness)
	}
	m2, cmd := step(m, keyPress(tea.KeyEnter))
	if m2.prompt != nil {
		t.Error("enter should close the prompt")
	}
	if cmd == nil {
		t.Error("enter should return a newAgent command")
	}
	if m3, _ := step(m, keyPress(tea.KeyEscape)); m3.prompt != nil {
		t.Error("esc should cancel the prompt")
	}
}

func hasArgs(args []string, want ...string) bool {
	return strings.Contains(strings.Join(args, "\x00"), strings.Join(want, "\x00"))
}

func args(c *exec.Cmd) string {
	if c == nil {
		return "<nil>"
	}
	return c.String()
}
