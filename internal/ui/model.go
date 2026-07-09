// Package ui is gvardia's Bubble Tea cockpit: a three-pane read-only view over
// the fleet (projects, sessions, diff) plus a footer. All I/O happens in tea.Cmds
// (see commands.go); Update only reacts to messages and View only formats.
package ui

import (
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"

	"github.com/KirillSachkov/gvardia/internal/collect"
	"github.com/KirillSachkov/gvardia/internal/config"
	"github.com/KirillSachkov/gvardia/internal/model"
	"github.com/KirillSachkov/gvardia/internal/runners"
	"github.com/KirillSachkov/gvardia/internal/runs"
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

type workTab int

const (
	tabAgents workTab = iota
	tabTasks
	tabWorktrees
	tabTools
	tabHistory
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

	tasks     []model.Task   // task snapshot from local files and configured sources
	showTasks bool           // legacy compatibility flag; tasks now live in tabTasks
	taskScope bool           // in the tasks tab, limit to the selected project
	tasksVP   viewport.Model // legacy full-screen task viewport

	level     navLevel
	filtering bool   // true while the filter textinput is capturing input
	loading   bool   // true until the first fleet result arrives
	banner    string // last adapter/collector error, shown in the footer area
	toast     string // transient success note (e.g. clipboard copy), cleared on next key

	showHistory      bool                       // include ended sessions in the work pane
	historyByProject map[string][]model.Session // lazily loaded, keyed by project path
	runsByProject    map[string][]runs.Run      // local gvardia runs, keyed by project path
	runList          []runs.Run                 // rows currently shown when the runs view is active
	sessionList      []model.Session            // rows currently in the work table (cursor maps here)
	worktreeView     bool                       // legacy compatibility flag for tabWorktrees
	runsView         bool                       // legacy compatibility flag for tabAgents
	worktreeList     []model.Worktree           // rows in the worktree view (cursor maps here)
	taskList         []model.Task               // rows in the tasks tab (cursor maps here)
	toolList         []runners.Tool             // rows in the tools tab (cursor maps here)
	curated          bool                       // true when showing a curated tracked list (not a roots scan)
	tools            []runners.Tool             // installed/missing agent tools
	profiles         []runners.RunnerProfile    // runner profiles
	activeTab        workTab                    // selected right-pane tab
	showActions      bool                       // true while contextual actions help is open

	confirm    *confirmPrompt  // non-nil while a y/n confirmation is pending
	prompt     *newAgentPrompt // non-nil while the new-agent form is open
	launch     *launchPrompt   // non-nil while the run-launch picker is open
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
		tasksVP:          viewport.New(),
		filter:           filter,
		level:            levelProjects,
		loading:          true,
		runsView:         true,
		activeTab:        tabAgents,
		historyByProject: make(map[string][]model.Session),
		runsByProject:    make(map[string][]runs.Run),
		profiles:         runners.Profiles(cfg),
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

// selectedSession returns the work-session for the current selection. In the
// agents view it maps the cursor into sessionList; in the worktree view it
// returns the live session running in the selected worktree (if any), so
// attach/resume/kill act on the right agent.
func (m *Model) selectedSession() *model.Session {
	if m.showingRuns() || m.activeTab == tabTasks || m.activeTab == tabTools {
		return nil
	}
	if m.activeTab == tabWorktrees {
		w := m.selectedWorktree()
		if w == nil {
			return nil
		}
		p := m.selectedProject()
		if p == nil {
			return nil
		}
		for i := range p.WorkSessions {
			if p.WorkSessions[i].WorktreePath == w.Path {
				return &p.WorkSessions[i]
			}
		}
		return nil
	}
	i := m.sessions.Cursor()
	if i < 0 || i >= len(m.sessionList) {
		return nil
	}
	return &m.sessionList[i]
}

func (m *Model) selectedRun() *runs.Run {
	if !m.showingRuns() {
		return nil
	}
	i := m.sessions.Cursor()
	if i < 0 || i >= len(m.runList) {
		return nil
	}
	return &m.runList[i]
}

// selectedWorktree returns the worktree under the cursor in the worktree view.
func (m *Model) selectedWorktree() *model.Worktree {
	if m.activeTab != tabWorktrees {
		return nil
	}
	i := m.sessions.Cursor()
	if i < 0 || i >= len(m.worktreeList) {
		return nil
	}
	return &m.worktreeList[i]
}

func (m *Model) selectedTask() *model.Task {
	if m.activeTab != tabTasks {
		return nil
	}
	i := m.sessions.Cursor()
	if i < 0 || i >= len(m.taskList) {
		return nil
	}
	return &m.taskList[i]
}

func (m *Model) selectedTool() *runners.Tool {
	if m.activeTab != tabTools {
		return nil
	}
	i := m.sessions.Cursor()
	if i < 0 || i >= len(m.toolList) {
		return nil
	}
	return &m.toolList[i]
}

// currentDetail returns the detail body (summary · task · report · artifacts)
// and diff-target worktree for the current selection in either view. The body is
// "" when nothing is selected.
func (m *Model) currentDetail() (string, *model.Worktree) {
	if t := m.selectedTask(); t != nil {
		return taskDetail(*t), nil
	}
	if tool := m.selectedTool(); tool != nil {
		return toolDetail(*tool), nil
	}
	if r := m.selectedRun(); r != nil {
		return runDetail(*r), m.worktreeForRun(r)
	}
	if m.activeTab == tabWorktrees {
		w := m.selectedWorktree()
		if w == nil {
			return "", nil
		}
		body := worktreeHeader(*w)
		if s := m.selectedSession(); s != nil { // enriched WorkSession for this worktree
			body += sessionExtra(*s)
		}
		return body, w
	}
	s := m.selectedSession()
	if s == nil {
		return "", nil
	}
	return sessionDetail(*s), m.worktreeFor(s)
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

func (m *Model) worktreeForRun(r *runs.Run) *model.Worktree {
	p := m.selectedProject()
	if p == nil || r == nil {
		return nil
	}
	for i := range p.Worktrees {
		if p.Worktrees[i].Path == r.WorktreePath {
			return &p.Worktrees[i]
		}
	}
	if r.WorktreePath == "" {
		return nil
	}
	base := "main"
	if len(p.Worktrees) > 0 {
		base = p.Worktrees[0].BaseBranch
	}
	return &model.Worktree{Path: r.WorktreePath, Branch: r.Branch, BaseBranch: base}
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

// applyColumns sets the work-table columns for the active view. Callers must
// ensure the table holds no stale rows of a different column count first, since
// SetColumns re-renders the current rows.
func (m *Model) applyColumns() {
	w := m.geometry().rightInnerW
	switch m.activeTab {
	case tabTasks:
		m.sessions.SetColumns(taskColumns(w))
	case tabWorktrees:
		m.sessions.SetColumns(worktreeColumns(w))
	case tabTools:
		m.sessions.SetColumns(toolColumns(w))
	case tabHistory:
		m.sessions.SetColumns(sessionColumns(w))
	default:
		if m.showingRuns() {
			m.sessions.SetColumns(runColumns(w))
		} else {
			m.sessions.SetColumns(sessionColumns(w))
		}
	}
}

// rebuildSessions repopulates the work table for the selected project: either
// the agent sessions (live, plus ended history when enabled) or, in the worktree
// view, every worktree. Rows are cleared before the columns are (re)applied so a
// view switch never renders old rows against the new column set.
func (m *Model) rebuildSessions() {
	m.sessions.SetRows(nil)
	m.applyColumns()
	m.runList = nil
	m.sessionList = nil
	m.worktreeList = nil
	m.taskList = nil
	m.toolList = nil

	p := m.selectedProject()
	if p == nil && m.activeTab != tabTools {
		return
	}

	var rows []table.Row
	switch m.activeTab {
	case tabTasks:
		list := m.filteredTasks()
		m.taskList = list
		rows = make([]table.Row, len(list))
		for i, t := range list {
			rows[i] = taskRow(t)
		}
	case tabWorktrees:
		m.worktreeList = p.Worktrees
		rows = make([]table.Row, len(p.Worktrees))
		for i, w := range p.Worktrees {
			rows[i] = worktreeRow2(w)
		}
	case tabTools:
		list := m.filteredTools()
		m.toolList = list
		rows = make([]table.Row, len(list))
		for i, tool := range list {
			rows[i] = toolRow(tool)
		}
	case tabHistory:
		m.showHistory = true
		list := collect.MergeHistory(p.WorkSessions, m.historyByProject[p.Path])
		m.sessionList = list
		rows = make([]table.Row, len(list))
		for i, s := range list {
			rows[i] = sessionRow(s)
		}
	default:
		if !m.showingRuns() {
			list := p.WorkSessions
			m.sessionList = list
			rows = make([]table.Row, len(list))
			for i, s := range list {
				rows[i] = sessionRow(s)
			}
			break
		}
		list := m.runsByProject[p.Path]
		m.runList = list
		rows = make([]table.Row, len(list))
		for i, r := range list {
			rows[i] = runRow(r)
		}
	}
	m.sessions.SetRows(rows)
	if c := m.sessions.Cursor(); (c < 0 || c >= len(rows)) && len(rows) > 0 {
		m.sessions.SetCursor(0)
	}
}

func (m *Model) showingRuns() bool {
	if m.activeTab != tabAgents || !m.runsView {
		return false
	}
	p := m.selectedProject()
	return p != nil && len(m.runsByProject[p.Path]) > 0
}

func (m *Model) filteredTasks() []model.Task {
	q := strings.ToLower(strings.TrimSpace(m.filter.Value()))
	scope := ""
	if m.taskScope {
		if p := m.selectedProject(); p != nil {
			scope = p.Name
		}
	}
	out := make([]model.Task, 0, len(m.tasks))
	for _, t := range m.tasks {
		if scope != "" && !projectMatches(t.Project, scope) {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(t.Title+" "+t.Project+" "+t.ID+" "+t.Status), q) {
			continue
		}
		out = append(out, t)
	}
	return out
}

func (m *Model) filteredTools() []runners.Tool {
	q := strings.ToLower(strings.TrimSpace(m.filter.Value()))
	out := make([]runners.Tool, 0, len(m.tools))
	for _, tool := range m.tools {
		haystack := strings.ToLower(tool.Name + " " + tool.Command + " " + tool.Path)
		if q == "" || strings.Contains(haystack, q) {
			out = append(out, tool)
		}
	}
	return out
}
