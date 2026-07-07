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
		{Name: "alpha", Path: "/r/alpha", Worktrees: []model.Worktree{
			{Path: "/r/alpha", Branch: "main", IsPrimary: true},
		}},
		{Name: "beta", Path: "/r/beta", Worktrees: []model.Worktree{
			{Path: "/r/beta", Branch: "dev", IsPrimary: true},
			{Path: "/r/beta/wt", Branch: "feat"},
		}},
	}
}

func ready(t *testing.T) Model {
	t.Helper()
	m := New(config.Default())
	if !m.loading {
		t.Fatal("new model should start loading")
	}
	m, _ = step(m, tea.WindowSizeMsg{Width: 120, Height: 40})
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

func TestFleetPopulatesPanes(t *testing.T) {
	m := ready(t)
	if p := m.selectedProject(); p == nil || p.Name != "alpha" {
		t.Fatalf("selectedProject = %v, want alpha", p)
	}
	// alpha has 1 worktree → 1 session-table row.
	if got := len(m.sessions.Rows()); got != 1 {
		t.Errorf("sessions rows = %d, want 1", got)
	}
}

func TestNavigateProjectsRebuildsSessions(t *testing.T) {
	m := ready(t)
	m, cmd := step(m, keyText("j")) // move down to beta
	if p := m.selectedProject(); p == nil || p.Name != "beta" {
		t.Fatalf("after 'j', selectedProject = %v, want beta", p)
	}
	if got := len(m.sessions.Rows()); got != 2 {
		t.Errorf("beta sessions rows = %d, want 2", got)
	}
	if cmd == nil {
		t.Error("moving selection should trigger a diff command")
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
	if m.filter.Value() != "bet" {
		t.Errorf("filter value = %q, want bet", m.filter.Value())
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
