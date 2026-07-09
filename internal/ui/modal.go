package ui

import (
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/KirillSachkov/gvardia/internal/model"
	"github.com/KirillSachkov/gvardia/internal/runners"
)

// confirmPrompt is a pending yes/no confirmation guarding a destructive action.
type confirmPrompt struct {
	message string
	action  tea.Cmd
}

// newAgentPrompt is the "new agent" form: a harness toggle plus a name input.
type newAgentPrompt struct {
	harness string
	input   textinput.Model
}

type launchPrompt struct {
	tasks      []model.Task
	taskIdx    int
	profileIdx int
}

type actionKind int

const (
	actionOpenDetails actionKind = iota
	actionAttach
	actionOpenDiff
	actionOpenReport
	actionLaunch
	actionKill
	actionGC
	actionCopyResume
	actionToggleTaskScope
	actionRefresh
)

type actionItem struct {
	label string
	hint  string
	kind  actionKind
}

type actionMenu struct {
	title  string
	items  []actionItem
	cursor int
}

// pathMode distinguishes the two project-curation prompts.
type pathMode int

const (
	pathAdd    pathMode = iota // track an existing git repo
	pathCreate                 // git init a new repo, then track it
)

// pathPrompt is the add/create-project form: a single path input.
type pathPrompt struct {
	mode  pathMode
	input textinput.Model
}

// handleConfirmKey resolves a pending confirmation.
func (m Model) handleConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	action := m.confirm.action
	switch normalizeKey(msg.String()) {
	case "y", "Y", "enter":
		m.confirm = nil
		return m, action
	case "n", "N", "esc", "ctrl+c":
		m.confirm = nil
		return m, nil
	}
	return m, nil
}

// handlePromptKey drives the new-agent form.
func (m Model) handlePromptKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.prompt = nil
		return m, nil
	case "tab":
		if m.prompt.harness == "claude" {
			m.prompt.harness = "codex"
		} else {
			m.prompt.harness = "claude"
		}
		return m, nil
	case "enter":
		p := m.selectedProject()
		if p == nil {
			m.prompt = nil
			return m, nil
		}
		harness, name := m.prompt.harness, m.prompt.input.Value()
		m.prompt = nil
		return m, newAgent(*p, harness, name)
	}

	var cmd tea.Cmd
	m.prompt.input, cmd = m.prompt.input.Update(msg)
	return m, cmd
}

func (m Model) handleLaunchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch normalizeKey(msg.String()) {
	case "esc", "ctrl+c":
		m.launch = nil
		return m, nil
	case "j", "down":
		if len(m.launch.tasks) > 0 {
			m.launch.taskIdx = (m.launch.taskIdx + 1) % len(m.launch.tasks)
		}
		return m, nil
	case "k", "up":
		if len(m.launch.tasks) > 0 {
			m.launch.taskIdx--
			if m.launch.taskIdx < 0 {
				m.launch.taskIdx = len(m.launch.tasks) - 1
			}
		}
		return m, nil
	case "tab":
		if len(m.profiles) > 0 {
			m.launch.profileIdx = (m.launch.profileIdx + 1) % len(m.profiles)
		}
		return m, nil
	case "enter":
		p := m.selectedProject()
		if p == nil {
			m.launch = nil
			m.banner = "select a project first"
			return m, nil
		}
		if len(m.launch.tasks) == 0 {
			m.launch = nil
			m.banner = "no task selected"
			return m, nil
		}
		if len(m.profiles) == 0 {
			m.launch = nil
			m.banner = "no runner profiles configured"
			return m, nil
		}
		task := m.launch.tasks[m.launch.taskIdx]
		profile := m.profiles[m.launch.profileIdx]
		m.launch = nil
		return m, launchRun(*p, task, profile, m.cfg)
	}
	return m, nil
}

// handlePathKey drives the add/create-project path form. The text input receives
// the original key so Cyrillic paths type through unchanged.
func (m Model) handlePathKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.pathPrompt = nil
		return m, nil
	case "enter":
		path := strings.TrimSpace(m.pathPrompt.input.Value())
		mode := m.pathPrompt.mode
		m.pathPrompt = nil
		if path == "" {
			m.banner = "path required"
			return m, nil
		}
		if mode == pathAdd {
			return m, trackProject(path)
		}
		return m, createProject(path)
	}
	var cmd tea.Cmd
	m.pathPrompt.input, cmd = m.pathPrompt.input.Update(msg)
	return m, cmd
}

// openPathPrompt initializes and focuses the add/create-project form.
func (m *Model) openPathPrompt(mode pathMode) tea.Cmd {
	in := textinput.New()
	if mode == pathAdd {
		in.Placeholder = "path to an existing git repo…"
	} else {
		in.Placeholder = "path for a new project (git init)…"
	}
	m.pathPrompt = &pathPrompt{mode: mode, input: in}
	return m.pathPrompt.input.Focus()
}

// confirmUntrack opens a confirmation to drop the selected project from the
// curated list. It never deletes the repo on disk.
func (m Model) confirmUntrack() (tea.Model, tea.Cmd) {
	p := m.selectedProject()
	if p == nil {
		m.banner = "no project selected to untrack"
		return m, nil
	}
	if !m.curated {
		m.banner = "press A to start curating before untracking"
		return m, nil
	}
	m.confirm = &confirmPrompt{
		message: fmt.Sprintf("untrack %s? (repo stays on disk)", p.Name),
		action:  untrackProject(p.Path),
	}
	return m, nil
}

// confirmKill opens a confirmation to SIGTERM the selected live session's
// process. Ended (history) sessions have no PID and cannot be killed.
func (m Model) confirmKill() (tea.Model, tea.Cmd) {
	if r := m.selectedRun(); r != nil {
		if r.TmuxTarget == "" {
			m.banner = "run has no tmux target"
			return m, nil
		}
		m.confirm = &confirmPrompt{
			message: fmt.Sprintf("kill run %s (%s)?", r.ID, r.TmuxTarget),
			action:  killRun(*r),
		}
		return m, nil
	}
	s := m.selectedSession()
	if s == nil {
		m.banner = "no session selected to kill"
		return m, nil
	}
	if s.PID <= 0 {
		m.banner = "only a live session with a PID can be killed"
		return m, nil
	}
	m.confirm = &confirmPrompt{
		message: fmt.Sprintf("kill %s %s (pid %d)?", s.Harness, s.Name, s.PID),
		action:  killSession(s.PID),
	}
	return m, nil
}

// confirmGC opens a confirmation to gc merged/stale worktrees in the current root.
func (m Model) confirmGC() (tea.Model, tea.Cmd) {
	root := m.currentRoot()
	if root == "" {
		m.banner = "no root to gc"
		return m, nil
	}
	m.confirm = &confirmPrompt{
		message: fmt.Sprintf("gc merged/stale worktrees in %s?", root),
		action:  gcRoot(root),
	}
	return m, nil
}

// openNewAgentPrompt initializes and focuses the new-agent form.
func (m *Model) openNewAgentPrompt() tea.Cmd {
	in := textinput.New()
	in.Placeholder = "agent name…"
	m.prompt = &newAgentPrompt{harness: "claude", input: in}
	return m.prompt.input.Focus()
}

func (m *Model) openLaunchPrompt() tea.Cmd {
	p := m.selectedProject()
	if p == nil {
		m.banner = "select a project first"
		return nil
	}
	var scoped []model.Task
	for _, t := range m.tasks {
		if t.Source == "local" && t.Project == p.Name {
			scoped = append(scoped, t)
			continue
		}
		if projectMatches(t.Project, p.Name) {
			scoped = append(scoped, t)
		}
	}
	if len(scoped) == 0 {
		scoped = append(scoped, model.Task{ID: "ad-hoc", Title: "Ad-hoc run", Status: "inbox", Project: p.Name, Body: "Inspect the project and write a plan before editing.", Source: "local"})
	}
	_, profileIdx := runners.DefaultProfile(m.profiles, m.cfg.DefaultRunner)
	m.launch = &launchPrompt{tasks: scoped, profileIdx: profileIdx}
	return nil
}

// currentRoot returns the configured root that contains the selected project,
// falling back to the first root.
func (m *Model) currentRoot() string {
	if p := m.selectedProject(); p != nil {
		for _, root := range m.cfg.Roots {
			if p.Path == root || strings.HasPrefix(p.Path, root+string(os.PathSeparator)) {
				return root
			}
		}
	}
	if len(m.cfg.Roots) > 0 {
		return m.cfg.Roots[0]
	}
	return ""
}
