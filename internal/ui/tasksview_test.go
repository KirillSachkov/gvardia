package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/KirillSachkov/gvardia/internal/config"
	"github.com/KirillSachkov/gvardia/internal/model"
)

func readyWithTasks(t *testing.T) Model {
	t.Helper()
	m := New(config.Default())
	m, _ = step(m, tea.WindowSizeMsg{Width: 140, Height: 40})
	m, _ = step(m, fleetMsg{
		projects: testProjects(),
		tasks: []model.Task{
			{ID: "t1", Title: "Fix payment bug", Status: "active", Project: "alpha"},
			{ID: "t2", Title: "Research thing", Status: "inbox"},
			{ID: "t3", Title: "Shipped feature", Status: "done", Project: "beta"},
		},
	})
	return m
}

func TestTasksViewToggles(t *testing.T) {
	m := readyWithTasks(t)
	if m.activeTab != tabAgents {
		t.Fatal("tasks tab should not be active initially")
	}
	m, _ = step(m, keyText("t"))
	if m.activeTab != tabTasks {
		t.Fatal("t should switch to the tasks tab")
	}
	if out := m.render(); !strings.Contains(out, "Fix payment bug") {
		t.Errorf("tasks tab should list tasks; render:\n%s", out)
	}
	m, _ = step(m, keyPress(tea.KeyEscape))
	if m.level != levelProjects {
		t.Error("esc should return focus toward projects")
	}
}

func TestTasksViewProjectScope(t *testing.T) {
	m := readyWithTasks(t)
	m, _ = step(m, keyText("t")) // tasks tab (alpha selected)
	// scope on → only alpha's task remains visible
	m, _ = step(m, keyText("s"))
	if !m.taskScope {
		t.Fatal("s should enable project scope")
	}
	out := m.render()
	if !strings.Contains(out, "Fix payment bug") {
		t.Error("alpha-scoped task should stay visible")
	}
	if strings.Contains(out, "Shipped feature") {
		t.Error("beta task should be hidden under alpha scope")
	}
}

func TestProjectMatches(t *testing.T) {
	if !projectMatches("education-platform", "education-platform") {
		t.Error("exact project should match")
	}
	if !projectMatches("sachkov-os / ai-workflows", "sachkov-os") {
		t.Error("substring project should match")
	}
	if projectMatches("", "alpha") {
		t.Error("empty task project must not match")
	}
}
