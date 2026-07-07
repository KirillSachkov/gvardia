// Package ui is gvardia's Bubble Tea cockpit: a three-pane read-only view over
// the fleet (projects, sessions, diff) plus a footer. All I/O happens in tea.Cmds
// (see commands.go); Update only reacts to messages and View only formats.
package ui

import (
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"

	"github.com/KirillSachkov/gvardia/internal/config"
	"github.com/KirillSachkov/gvardia/internal/model"
)

// focusArea is the pane that currently receives navigation keys.
type focusArea int

const (
	focusProjects focusArea = iota
	focusSessions
)

// Model holds the cockpit state. It stores data and selection only; everything
// visual is derived in View.
type Model struct {
	cfg           config.Config
	width, height int

	projects []model.Project
	projList list.Model
	sessions table.Model
	diff     viewport.Model
	filter   textinput.Model

	focus     focusArea
	filtering bool   // true while the filter textinput is capturing input
	loading   bool   // true until the first fleet result arrives
	banner    string // last adapter/collector error, shown in the footer area
}

// New builds the initial cockpit model. Component sizes are placeholders until
// the first WindowSizeMsg arrives.
func New(cfg config.Config) Model {
	projList := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	projList.Title = "Projects"
	projList.SetShowStatusBar(false)
	projList.SetShowHelp(false)
	projList.SetFilteringEnabled(false) // gvardia drives filtering itself

	sessions := table.New(
		table.WithColumns(sessionColumns(0)),
		table.WithFocused(false),
	)

	filter := textinput.New()
	filter.Placeholder = "filter projects…"

	return Model{
		cfg:      cfg,
		projList: projList,
		sessions: sessions,
		diff:     viewport.New(),
		filter:   filter,
		focus:    focusProjects,
		loading:  true,
	}
}

// selectedProject returns the project under the projects cursor, or nil.
func (m *Model) selectedProject() *model.Project {
	if len(m.projects) == 0 {
		return nil
	}
	item, ok := m.projList.SelectedItem().(projectItem)
	if !ok {
		return nil
	}
	if item.idx < 0 || item.idx >= len(m.projects) {
		return nil
	}
	return &m.projects[item.idx]
}

// selectedWorktree returns the worktree under the sessions cursor, or nil.
func (m *Model) selectedWorktree() *model.Worktree {
	p := m.selectedProject()
	if p == nil {
		return nil
	}
	i := m.sessions.Cursor()
	if i < 0 || i >= len(p.Worktrees) {
		return nil
	}
	return &p.Worktrees[i]
}

// setProjects stores a fresh fleet and rebuilds the projects list, preserving the
// cursor position where possible.
func (m *Model) setProjects(projects []model.Project) {
	prevIdx := m.projList.Index()
	m.projects = projects

	items := make([]list.Item, 0, len(projects))
	for i, p := range projects {
		items = append(items, projectItem{idx: i, project: p})
	}
	m.applyFilter(items)

	if prevIdx >= 0 && prevIdx < len(items) {
		m.projList.Select(prevIdx)
	}
	m.rebuildSessions()
}

// applyFilter sets the list items, narrowing to those matching the filter text.
func (m *Model) applyFilter(items []list.Item) {
	q := m.filter.Value()
	if q == "" {
		m.projList.SetItems(items)
		return
	}
	filtered := make([]list.Item, 0, len(items))
	for _, it := range items {
		if it.(projectItem).matches(q) {
			filtered = append(filtered, it)
		}
	}
	m.projList.SetItems(filtered)
}

// rebuildSessions repopulates the sessions table from the selected project.
func (m *Model) rebuildSessions() {
	p := m.selectedProject()
	if p == nil {
		m.sessions.SetRows(nil)
		return
	}
	rows := make([]table.Row, 0, len(p.Worktrees))
	for _, w := range p.Worktrees {
		rows = append(rows, worktreeRow(w))
	}
	m.sessions.SetRows(rows)
	if m.sessions.Cursor() >= len(rows) {
		m.sessions.SetCursor(0)
	}
}
