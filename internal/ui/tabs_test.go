package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

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
	if m.showProjects {
		t.Fatal("enter from projects should hide the projects drawer")
	}
	m, _ = step(m, keyText("2"))
	if m.level != levelWork {
		t.Fatalf("switching tabs should keep work focus, got %v", m.level)
	}
	m, _ = step(m, keyPress(tea.KeyEnter))
	if !m.showActions || m.actionMenu == nil {
		t.Fatalf("enter from tasks tab should open contextual actions, showActions=%v menu=%v", m.showActions, m.actionMenu)
	}
	m, _ = step(m, keyPress(tea.KeyEscape))
	if m.showActions || m.level != levelWork || m.activeTab != tabTasks {
		t.Fatalf("esc should return to tasks list, actions=%v level=%v tab=%v", m.showActions, m.level, m.activeTab)
	}
}

func TestProjectDrawerAndPaneNavigation(t *testing.T) {
	m := readyWithOpsData(t)
	if !m.showProjects || m.level != levelProjects {
		t.Fatalf("initial drawer=%v level=%v, want visible projects", m.showProjects, m.level)
	}

	m, _ = step(m, keyPress(tea.KeyEnter))
	if m.showProjects || m.level != levelWork {
		t.Fatalf("enter should hide projects and focus work, drawer=%v level=%v", m.showProjects, m.level)
	}

	m, _ = step(m, keyText("p"))
	if !m.showProjects || m.level != levelProjects {
		t.Fatalf("p should show projects and focus them, drawer=%v level=%v", m.showProjects, m.level)
	}

	m, _ = step(m, keyPress(tea.KeyRight))
	if m.level != levelWork {
		t.Fatalf("right from projects level=%v, want work", m.level)
	}
	m, _ = step(m, keyPress(tea.KeyRight))
	if m.level != levelDetail {
		t.Fatalf("right from work level=%v, want detail", m.level)
	}
	m, _ = step(m, keyPress(tea.KeyLeft))
	if m.level != levelWork {
		t.Fatalf("left from detail level=%v, want work", m.level)
	}
}

func TestShiftTabMovesFocusBackward(t *testing.T) {
	m := readyWithOpsData(t)
	m, _ = step(m, keyPress(tea.KeyEnter)) // projects -> work, drawer hidden
	m, _ = step(m, keyPress(tea.KeyRight)) // work -> detail
	m, _ = step(m, tea.KeyPressMsg(tea.Key{Text: "shift+tab"}))
	if m.level != levelWork {
		t.Fatalf("shift+tab from detail level=%v, want work", m.level)
	}
	m, _ = step(m, keyText("p")) // show projects
	m, _ = step(m, keyPress(tea.KeyRight))
	m, _ = step(m, tea.KeyPressMsg(tea.Key{Text: "shift+tab"}))
	if m.level != levelProjects {
		t.Fatalf("shift+tab from work with drawer level=%v, want projects", m.level)
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
	for _, want := range []string{"1 agents", "2 tasks", "3 worktrees", "enter actions"} {
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
	if !m.showActions || m.actionMenu == nil {
		t.Fatal("? should open contextual actions")
	}
	out := strings.ToLower(m.render())
	for _, want := range []string{"Actions", "details", "attach", "launch"} {
		if !strings.Contains(out, strings.ToLower(want)) {
			t.Fatalf("actions help missing %q:\n%s", want, out)
		}
	}
	m, _ = step(m, keyPress(tea.KeyEscape))
	if m.showActions {
		t.Fatal("esc should close contextual actions")
	}
}

func TestActionsRenderAsModalOverlayNotDetailReplacement(t *testing.T) {
	m := readyWithOpsData(t)
	m.diff.SetContent("DETAIL_UNDERLAY_MARKER")
	m.showActions = true
	m.actionMenu = &actionMenu{
		title: "Agent: a1",
		items: []actionItem{{label: "Open details", hint: "Show details", kind: actionOpenDetails}},
	}

	if detail := m.detailPaneView(); !strings.Contains(detail, "DETAIL_UNDERLAY_MARKER") {
		t.Fatalf("detail pane should keep its content under the modal, got:\n%s", detail)
	}
	if detail := m.detailPaneView(); strings.Contains(detail, "Actions") {
		t.Fatalf("detail pane should not be replaced by actions, got:\n%s", detail)
	}

	out := m.render()
	if !strings.Contains(out, "Actions - Agent: a1") {
		t.Fatalf("render should show the actions modal:\n%s", out)
	}
}

func TestContextActionMenuCanOpenDetail(t *testing.T) {
	m := readyWithOpsData(t)
	m, _ = step(m, keyPress(tea.KeyEnter)) // project -> work
	m, _ = step(m, keyPress(tea.KeyEnter)) // work -> actions
	if m.actionMenu == nil || len(m.actionMenu.items) == 0 {
		t.Fatal("enter on a selected work item should open an action menu")
	}
	if got := m.actionMenu.items[0].label; got != "Open details" {
		t.Fatalf("first action = %q, want Open details", got)
	}
	m, _ = step(m, keyPress(tea.KeyEnter))
	if m.showActions || m.level != levelDetail {
		t.Fatalf("enter on Open details should close menu and focus detail, actions=%v level=%v", m.showActions, m.level)
	}
}

func TestRenderedPanesFillTerminalWidth(t *testing.T) {
	m := readyWithOpsData(t)
	m.width = 120
	m.height = 32
	m.layout()

	lines := strings.Split(m.render(), "\n")
	bodyLines := lines[:len(lines)-footerHeight]
	for i, line := range bodyLines {
		if got := lipgloss.Width(line); got != m.width {
			t.Fatalf("body line %d width = %d, want %d: %q", i, got, m.width, line)
		}
	}

	m, _ = step(m, keyText("p"))
	lines = strings.Split(m.render(), "\n")
	bodyLines = lines[:len(lines)-footerHeight]
	for i, line := range bodyLines {
		if got := lipgloss.Width(line); got != m.width {
			t.Fatalf("hidden drawer body line %d width = %d, want %d: %q", i, got, m.width, line)
		}
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

func TestTaskAndToolDetailsDoNotShowWorktreeGoneNoise(t *testing.T) {
	m := readyWithOpsData(t)
	m, _ = step(m, keyText("2"))
	if out := m.render(); strings.Contains(out, "worktree gone") {
		t.Fatalf("task detail should not show worktree-gone noise:\n%s", out)
	}
	m, _ = step(m, keyText("4"))
	if out := m.render(); strings.Contains(out, "worktree gone") {
		t.Fatalf("tool detail should not show worktree-gone noise:\n%s", out)
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
