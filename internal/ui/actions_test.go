package ui

import (
	"os/exec"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/KirillSachkov/gvardia/internal/model"
)

func TestSessionExec(t *testing.T) {
	claude := sessionExec(model.Session{Harness: "claude", SessionID: "abc", WorktreePath: "/w"}, true)
	if claude == nil || !hasArgs(claude.Args, "--resume", "abc") {
		t.Errorf("claude exec args = %v", args(claude))
	}
	if claude != nil && claude.Dir != "/w" {
		t.Errorf("claude Dir = %q, want /w", claude.Dir)
	}

	codex := sessionExec(model.Session{Harness: "codex", SessionID: "xyz", WorktreePath: "/w"}, true)
	if codex == nil || !hasArgs(codex.Args, "resume", "xyz") {
		t.Errorf("codex exec args = %v", args(codex))
	}
	codexLast := sessionExec(model.Session{Harness: "codex", WorktreePath: "/w"}, true)
	if codexLast == nil || !hasArgs(codexLast.Args, "resume", "--last") {
		t.Errorf("codex without id should use --last, got %v", args(codexLast))
	}

	tm := sessionExec(model.Session{Harness: "tmux", SessionID: "work"}, true)
	if tm == nil || !hasArgs(tm.Args, "attach", "-t", "work") {
		t.Errorf("tmux attach args = %v", args(tm))
	}
	if sessionExec(model.Session{Harness: "tmux", SessionID: "work"}, false) != nil {
		t.Error("tmux resume (attach=false) should be nil")
	}
	if sessionExec(model.Session{Harness: "unknown"}, true) != nil {
		t.Error("unknown harness → nil command")
	}
}

func TestKillConfirmFlow(t *testing.T) {
	m := ready(t)
	// alpha's selected session has no PID → 'k' banners, no confirm.
	m, _ = step(m, keyText("k"))
	if m.confirm != nil {
		t.Fatal("k with a PID-less session should not open a confirm")
	}

	// Give the selected session a PID, then 'k' → confirm.
	m.projects[0].WorkSessions[0].PID = 4242
	m.rebuildSessions()
	m, _ = step(m, keyText("k"))
	if m.confirm == nil {
		t.Fatal("k with a killable session should open a confirm")
	}

	if m2, cmd := step(m, keyText("n")); m2.confirm != nil || cmd != nil {
		t.Error("n should cancel the confirm with no action")
	}
	m3, cmd := step(m, keyText("y"))
	if m3.confirm != nil || cmd == nil {
		t.Error("y should clear the confirm and return the kill action")
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

func TestLaunchPrompt(t *testing.T) {
	m := ready(t)
	m.tasks = []model.Task{
		{ID: "a", Title: "Alpha task", Project: "alpha", Source: "local"},
		{ID: "b", Title: "Other project", Project: "beta", Source: "local"},
	}
	m, _ = step(m, keyText("n"))
	if m.launch == nil {
		t.Fatal("n should open the launch prompt")
	}
	if len(m.launch.tasks) != 1 || m.launch.tasks[0].Title != "Alpha task" {
		t.Fatalf("launch tasks = %+v, want alpha-scoped task", m.launch.tasks)
	}
	firstRunner := m.profiles[m.launch.profileIdx].Name
	m, _ = step(m, keyPress(tea.KeyTab))
	if m.profiles[m.launch.profileIdx].Name == firstRunner {
		t.Errorf("tab should switch runner profile from %q", firstRunner)
	}
	m2, cmd := step(m, keyPress(tea.KeyEnter))
	if m2.launch != nil || cmd == nil {
		t.Error("enter should close the launch prompt and return a launch command")
	}
	m, _ = step(m, keyText("n"))
	if m3, _ := step(m, keyPress(tea.KeyEscape)); m3.launch != nil {
		t.Error("esc should cancel the launch prompt")
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
