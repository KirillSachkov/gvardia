package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/KirillSachkov/gvardia/internal/model"
	"github.com/KirillSachkov/gvardia/internal/runners"
)

func readyWithOpsData(t *testing.T) Model {
	t.Helper()
	m := readyWithTasks(t)
	m.tools = []runners.Tool{
		{Name: "claude", Command: "claude", Path: "/bin/claude", Installed: true, BuiltIn: true},
		{Name: "gemini", Command: "gemini", Installed: false, BuiltIn: true},
	}
	return m
}

func TestNumberKeysSwitchWorkTabs(t *testing.T) {
	m := readyWithOpsData(t)
	if m.activeTab != tabAgents {
		t.Fatalf("start tab = %v, want agents", m.activeTab)
	}

	cases := []struct {
		key  string
		tab  workTab
		want string
	}{
		{"2", tabTasks, "Fix payment bug"},
		{"3", tabWorktrees, "main *"},
		{"4", tabTools, "claude"},
		{"5", tabHistory, "harness"},
		{"1", tabAgents, "a1"},
	}
	for _, tc := range cases {
		m, _ = step(m, keyText(tc.key))
		if m.activeTab != tc.tab {
			t.Fatalf("%s tab = %v, want %v", tc.key, m.activeTab, tc.tab)
		}
		if out := m.render(); !strings.Contains(out, tc.want) {
			t.Fatalf("%s render should contain %q:\n%s", tc.key, tc.want, out)
		}
	}
}

func TestEnterAndEscMoveFocusWithoutModeJumps(t *testing.T) {
	m := readyWithOpsData(t)
	m, _ = step(m, keyPress(tea.KeyEnter))
	if m.level != levelWork {
		t.Fatalf("enter from projects level = %v, want work", m.level)
	}
	m, _ = step(m, keyText("2"))
	if m.level != levelWork {
		t.Fatalf("switching tabs should keep work focus, got %v", m.level)
	}
	m, _ = step(m, keyPress(tea.KeyEnter))
	if m.level != levelDetail {
		t.Fatalf("enter from tasks tab level = %v, want detail", m.level)
	}
	m, _ = step(m, keyPress(tea.KeyEscape))
	if m.level != levelWork || m.activeTab != tabTasks {
		t.Fatalf("esc should return to tasks list, got level=%v tab=%v", m.level, m.activeTab)
	}
}

func TestAgentsTabArrowKeysMoveBetweenAgents(t *testing.T) {
	m := readyWithOpsData(t)
	m, _ = step(m, keyText("j"))           // beta has two live agent rows
	m, _ = step(m, keyPress(tea.KeyEnter)) // focus the agents tab
	m, _ = step(m, keyPress(tea.KeyDown))  // move from b1 to b2
	if got := m.sessions.Cursor(); got != 1 {
		t.Fatalf("down arrow in agents tab cursor = %d, want 1", got)
	}
	if s := m.selectedSession(); s == nil || s.Name != "b2" {
		t.Fatalf("selected agent after down = %+v, want b2", s)
	}
}

func TestFooterIsContextualAndNotAHotkeyDump(t *testing.T) {
	m := readyWithOpsData(t)
	out := m.footer()
	for _, want := range []string{"1 agents", "2 tasks", "3 worktrees", "? actions"} {
		if !strings.Contains(strings.ToLower(out), want) {
			t.Fatalf("footer missing %q: %s", want, out)
		}
	}
	for _, stale := range []string{"u runs", "w worktrees", "t tasks", "A add", "X untrack", "C create"} {
		if strings.Contains(out, stale) {
			t.Fatalf("footer should not expose old global hotkey dump %q: %s", stale, out)
		}
	}
}

func TestQuestionMarkShowsContextActions(t *testing.T) {
	m := readyWithOpsData(t)
	m, _ = step(m, keyText("?"))
	if !m.showActions {
		t.Fatal("? should open contextual actions")
	}
	out := m.render()
	for _, want := range []string{"Actions", "attach", "report", "launch"} {
		if !strings.Contains(out, want) {
			t.Fatalf("actions help missing %q:\n%s", want, out)
		}
	}
	m, _ = step(m, keyPress(tea.KeyEscape))
	if m.showActions {
		t.Fatal("esc should close contextual actions")
	}
}

func TestTasksTabDetailUsesSelection(t *testing.T) {
	m := readyWithOpsData(t)
	m, _ = step(m, keyText("2"))
	body, wt := m.currentDetail()
	if wt != nil {
		t.Fatalf("task detail should not select a worktree, got %v", wt)
	}
	if !strings.Contains(body, "Fix payment bug") || !strings.Contains(body, "active") {
		t.Fatalf("task detail should describe selected task, got:\n%s", body)
	}
}

func TestToolsTabShowsInstallStatus(t *testing.T) {
	m := readyWithOpsData(t)
	m, _ = step(m, keyText("4"))
	out := m.render()
	if !strings.Contains(out, "installed") || !strings.Contains(out, "missing") {
		t.Fatalf("tools tab should show installed and missing states:\n%s", out)
	}
}

func TestMouseClickCanSelectTab(t *testing.T) {
	m := readyWithOpsData(t)
	g := m.geometry()
	x := g.leftInnerW + 2 + g.rightInnerW/2
	m, _ = step(m, tea.MouseClickMsg{X: x, Y: 1, Button: tea.MouseLeft})
	if m.activeTab != tabWorktrees {
		t.Fatalf("middle top click tab = %v, want worktrees", m.activeTab)
	}
}

func TestTaskRowKeepsProjectVisible(t *testing.T) {
	row := taskRow(model.Task{ID: "t1", Title: "Fix payment bug", Status: "active", Project: "alpha"})
	if !strings.Contains(strings.Join(row, " "), "alpha") {
		t.Fatalf("task row should keep project context, got %v", row)
	}
}
