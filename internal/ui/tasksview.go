package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/KirillSachkov/gvardia/internal/model"
)

// kanbanColumns is the display order of the tasks browser sections.
var kanbanColumns = []string{"inbox", "active", "done"}

// rebuildTasks renders the kanban into the tasks viewport, grouped by column and
// narrowed by the active filter and (optionally) the selected project scope.
func (m *Model) rebuildTasks() {
	q := strings.ToLower(m.filter.Value())

	scope := ""
	if m.taskScope {
		if p := m.selectedProject(); p != nil {
			scope = p.Name
		}
	}

	var b strings.Builder
	title := fmt.Sprintf("Tasks — %d in the kanban", len(m.tasks))
	if scope != "" {
		title += " · project:" + scope
	}
	if q != "" {
		title += " · filter:" + m.filter.Value()
	}
	b.WriteString(title + "\n\n")

	for _, col := range kanbanColumns {
		matched := make([]model.Task, 0)
		for _, t := range m.tasks {
			if t.Status != col {
				continue
			}
			if scope != "" && !projectMatches(t.Project, scope) {
				continue
			}
			if q != "" && !strings.Contains(strings.ToLower(t.Title+" "+t.Project+" "+t.ID), q) {
				continue
			}
			matched = append(matched, t)
		}
		b.WriteString(dim.Render(fmt.Sprintf("%s (%d)", col, len(matched))) + "\n")
		for _, t := range matched {
			line := "  • " + t.Title
			if t.Project != "" {
				line += "  [" + t.Project + "]"
			}
			b.WriteString(truncate(line, max1(m.width-2)) + "\n")
		}
		b.WriteString("\n")
	}

	m.tasksVP.SetContent(b.String())
	m.tasksVP.GotoTop()
}

// projectMatches reports whether a task's project field refers to the given
// project name (case-insensitive, either-way substring). Empty never matches.
func projectMatches(taskProject, name string) bool {
	tp := strings.ToLower(strings.TrimSpace(taskProject))
	n := strings.ToLower(strings.TrimSpace(name))
	if tp == "" || n == "" {
		return false
	}
	return strings.Contains(tp, n) || strings.Contains(n, tp)
}

// handleTasksKey drives the full-screen tasks browser: scroll, filter, scope,
// and close. It is read-only.
func (m Model) handleTasksKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch normalizeKey(msg.String()) {
	case "t", "esc", "q":
		m.showTasks = false
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	case "/":
		m.filtering = true
		return m, m.filter.Focus()
	case "p":
		m.taskScope = !m.taskScope
		m.rebuildTasks()
		return m, nil
	case "R":
		return m, collectFleet(m.cfg)
	}
	var cmd tea.Cmd
	m.tasksVP, cmd = m.tasksVP.Update(msg)
	return m, cmd
}
