// Package ui is gvardia's Bubble Tea cockpit: a three-pane read-only view over
// the fleet (projects, sessions, diff) plus a footer. All I/O happens in tea.Cmds
// (see commands.go); Update only reacts to messages and View only formats.
package ui

import (
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"

	"github.com/KirillSachkov/gvardia/internal/collect"
	"github.com/KirillSachkov/gvardia/internal/config"
	"github.com/KirillSachkov/gvardia/internal/model"
)

// navLevel is how deep the cockpit is drilled in: projects (L0), the selected
// project's work sessions (L1), or a single session's detail (L2). enter drills
// down a level, esc/backspace climbs back up. The active level also decides which
// pane receives navigation keys and shows the focus border.
type navLevel int

const (
	levelProjects navLevel = iota
	levelWork
	levelDetail
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

	level     navLevel
	filtering bool   // true while the filter textinput is capturing input
	loading   bool   // true until the first fleet result arrives
	banner    string // last adapter/collector error, shown in the footer area

	showHistory      bool                       // include ended sessions in the work pane
	historyByProject map[string][]model.Session // lazily loaded, keyed by project path
	sessionList      []model.Session            // rows currently in the work table (cursor maps here)
	curated          bool                       // true when showing a curated tracked list (not a roots scan)

	confirm    *confirmPrompt  // non-nil while a y/n confirmation is pending
	prompt     *newAgentPrompt // non-nil while the new-agent form is open
	pathPrompt *pathPrompt     // non-nil while the add/create-project path form is open
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
		cfg:              cfg,
		projList:         projList,
		sessions:         sessions,
		diff:             viewport.New(),
		filter:           filter,
		level:            levelProjects,
		loading:          true,
		historyByProject: make(map[string][]model.Session),
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

// selectedSession returns the work-session under the work-pane cursor, or nil.
func (m *Model) selectedSession() *model.Session {
	i := m.sessions.Cursor()
	if i < 0 || i >= len(m.sessionList) {
		return nil
	}
	return &m.sessionList[i]
}

// worktreeFor returns the worktree a session runs in (by path), or nil.
func (m *Model) worktreeFor(s *model.Session) *model.Worktree {
	p := m.selectedProject()
	if p == nil || s == nil {
		return nil
	}
	for i := range p.Worktrees {
		if p.Worktrees[i].Path == s.WorktreePath {
			return &p.Worktrees[i]
		}
	}
	return nil
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

// rebuildSessions repopulates the work table from the selected project's
// sessions (live, plus ended history when enabled).
func (m *Model) rebuildSessions() {
	p := m.selectedProject()
	if p == nil {
		m.sessionList = nil
		m.sessions.SetRows(nil)
		return
	}
	list := p.WorkSessions
	if m.showHistory {
		list = collect.MergeHistory(p.WorkSessions, m.historyByProject[p.Path])
	}
	m.sessionList = list

	rows := make([]table.Row, len(list))
	for i, s := range list {
		rows[i] = sessionRow(s)
	}
	m.sessions.SetRows(rows)
	if m.sessions.Cursor() >= len(rows) {
		m.sessions.SetCursor(0)
	}
}
