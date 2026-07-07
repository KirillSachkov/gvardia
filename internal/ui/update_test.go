package ui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/KirillSachkov/gvardia/internal/config"
	"github.com/KirillSachkov/gvardia/internal/model"
)

func testProjects() []model.Project {
	return []model.Project{
		{
			Name: "alpha", Path: "/r/alpha",
			Worktrees: []model.Worktree{{Path: "/r/alpha", Branch: "main", IsPrimary: true}},
			WorkSessions: []model.Session{
				{Harness: "claude", Name: "a1", SessionID: "s1", Live: true, Status: model.StatusBusy,
					Branch: "main", WorktreePath: "/r/alpha"},
			},
		},
		{
			Name: "beta", Path: "/r/beta",
			Worktrees: []model.Worktree{
				{Path: "/r/beta", Branch: "dev", IsPrimary: true},
				{Path: "/r/beta/wt", Branch: "feat/675-x"},
			},
			WorkSessions: []model.Session{
				{Harness: "claude", Name: "b1", SessionID: "sb1", Live: true, Status: model.StatusIdle,
					Branch: "dev", WorktreePath: "/r/beta"},
				{Harness: "codex", Name: "b2", SessionID: "sb2", Live: true, Status: model.StatusBusy,
					Branch: "feat/675-x", WorktreePath: "/r/beta/wt"},
			},
		},
	}
}

func ready(t *testing.T) Model {
	t.Helper()
	m := New(config.Default())
	if !m.loading {
		t.Fatal("new model should start loading")
	}
	m, _ = step(m, tea.WindowSizeMsg{Width: 140, Height: 40})
	m, cmd := step(m, fleetMsg{projects: testProjects()})
	if m.loading {
		t.Fatal("model should stop loading after fleetMsg")
	}
	if len(m.projects) != 2 {
		t.Fatalf("got %d projects, want 2", len(m.projects))
	}
	if cmd == nil {
		t.Error("fleetMsg should trigger a diff command for the selection")
	}
	return m
}

func TestFleetPopulatesSessions(t *testing.T) {
	m := ready(t)
	if p := m.selectedProject(); p == nil || p.Name != "alpha" {
		t.Fatalf("selectedProject = %v, want alpha", p)
	}
	if got := len(m.sessions.Rows()); got != 1 { // alpha has 1 work-session
		t.Errorf("session rows = %d, want 1", got)
	}
	if s := m.selectedSession(); s == nil || s.Name != "a1" {
		t.Fatalf("selectedSession = %v, want a1", s)
	}
}

func TestNavigateProjectsRebuildsSessions(t *testing.T) {
	m := ready(t)
	m, cmd := step(m, keyText("j")) // move to beta
	if p := m.selectedProject(); p == nil || p.Name != "beta" {
		t.Fatalf("after 'j', selectedProject = %v, want beta", p)
	}
	if got := len(m.sessions.Rows()); got != 2 { // beta has 2 work-sessions
		t.Errorf("beta session rows = %d, want 2", got)
	}
	if cmd == nil {
		t.Error("moving selection should trigger a diff command")
	}
}

func TestHistoryToggleMergesEndedSessions(t *testing.T) {
	m := ready(t)
	m, cmd := step(m, keyText("h"))
	if !m.showHistory {
		t.Fatal("'h' should turn history on")
	}
	if cmd == nil {
		t.Error("'h' should issue a history load for the selected project")
	}
	// Simulate the history result arriving.
	m, _ = step(m, historyMsg{projectPath: "/r/alpha", sessions: []model.Session{
		{Harness: "claude", Name: "old1", SessionID: "h1", Live: false},
	}})
	if got := len(m.sessions.Rows()); got != 2 { // 1 live + 1 ended
		t.Errorf("rows with history = %d, want 2", got)
	}
	// Toggle off → back to live only.
	m, _ = step(m, keyText("h"))
	if got := len(m.sessions.Rows()); got != 1 {
		t.Errorf("rows after hiding history = %d, want 1", got)
	}
}

func TestFilterNarrowsProjects(t *testing.T) {
	m := ready(t)
	m, _ = step(m, keyText("/"))
	if !m.filtering {
		t.Fatal("'/' should enter filter mode")
	}
	for _, r := range "bet" {
		m, _ = step(m, keyText(string(r)))
	}
	if p, ok := m.projList.SelectedItem().(projectItem); !ok || p.project.Name != "beta" {
		t.Errorf("filtered selection = %v, want beta", m.projList.SelectedItem())
	}
	m, _ = step(m, keyPress(tea.KeyEnter))
	if m.filtering {
		t.Error("enter should leave filter mode")
	}
}

func TestRefreshAndTickIssueCommands(t *testing.T) {
	m := ready(t)
	if _, cmd := step(m, keyText("R")); cmd == nil {
		t.Error("R should issue a collect command")
	}
	if _, cmd := step(m, tickMsg(time.Unix(0, 0))); cmd == nil {
		t.Error("tick should re-issue collect + tick")
	}
}

func TestQuitKey(t *testing.T) {
	m := ready(t)
	_, cmd := step(m, keyText("q"))
	if cmd == nil {
		t.Fatal("q should return the quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("q command should produce tea.QuitMsg")
	}
}

func step(m Model, msg tea.Msg) (Model, tea.Cmd) {
	next, cmd := m.Update(msg)
	return next.(Model), cmd
}

func keyText(s string) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: s, Code: []rune(s)[0]})
}

func keyPress(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code})
}
