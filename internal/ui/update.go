package ui

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

// Init kicks off the first fleet collection and starts the refresh ticker.
func (m Model) Init() tea.Cmd {
	return tea.Batch(collectFleet(m.cfg), tick(m.cfg.RefreshInterval.Duration))
}

// Update is the Elm loop: it reacts to messages, never doing I/O itself.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		return m, nil

	case fleetMsg:
		m.loading = false
		m.banner = failureBanner(msg.failures)
		m.setProjects(msg.projects)
		return m, m.diffForSelection()

	case errMsg:
		m.loading = false
		m.banner = msg.err.Error()
		return m, nil

	case diffMsg:
		m.diff.SetContent(msg.content)
		m.diff.GotoTop()
		return m, nil

	case execDoneMsg:
		// Returned from lazygit/git-diff/action: refresh in case things changed.
		return m, tea.Batch(collectFleet(m.cfg), m.diffForSelection())

	case spawnMsg:
		return m, spawnHarness(msg.harness, msg.dir)

	case tickMsg:
		// Re-collect and re-arm the ticker (Bubble Tea ticks once per call).
		return m, tea.Batch(collectFleet(m.cfg), tick(m.cfg.RefreshInterval.Duration))

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// handleKey routes a key press. Modals (confirm, new-agent) take priority, then
// the filter input, then global keys and navigation.
func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.confirm != nil:
		return m.handleConfirmKey(msg)
	case m.prompt != nil:
		return m.handlePromptKey(msg)
	case m.filtering:
		return m.handleFilterKey(msg)
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "tab":
		if m.focus == focusProjects {
			m.focus = focusSessions
		} else {
			m.focus = focusProjects
		}
		return m, nil
	case "/":
		m.filtering = true
		return m, m.filter.Focus()
	case "R":
		return m, collectFleet(m.cfg)
	case "enter":
		if w := m.selectedWorktree(); w != nil {
			return m, enterDiff(*w, m.cfg)
		}
		return m, nil
	case "a":
		if w := m.selectedWorktree(); w != nil {
			return m, attachSession(*w)
		}
		return m, nil
	case "r":
		if w := m.selectedWorktree(); w != nil {
			return m, resumeSession(*w)
		}
		return m, nil
	case "k":
		return m.confirmKill()
	case "g":
		return m.confirmGC()
	case "n":
		return m, m.openNewAgentPrompt()
	}

	if m.focus == focusProjects {
		return m.navigateProjects(msg)
	}
	return m.navigateSessions(msg)
}

// handleFilterKey feeds keys to the filter input, applying it live.
func (m Model) handleFilterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filtering = false
		m.filter.Blur()
		m.filter.Reset()
		m.refilter()
		return m, m.diffForSelection()
	case "enter":
		m.filtering = false
		m.filter.Blur()
		return m, m.diffForSelection()
	}

	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	m.refilter()
	return m, tea.Batch(cmd, m.diffForSelection())
}

// navigateProjects forwards a key to the projects list and refreshes downstream
// panes if the selection moved.
func (m Model) navigateProjects(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	prev := m.projList.Index()
	var cmd tea.Cmd
	m.projList, cmd = m.projList.Update(msg)
	if m.projList.Index() != prev {
		m.sessions.SetCursor(0)
		m.rebuildSessions()
		return m, tea.Batch(cmd, m.diffForSelection())
	}
	return m, cmd
}

// navigateSessions forwards a key to the sessions table and refreshes the diff if
// the selection moved.
func (m Model) navigateSessions(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	prev := m.sessions.Cursor()
	var cmd tea.Cmd
	m.sessions, cmd = m.sessions.Update(msg)
	if m.sessions.Cursor() != prev {
		return m, tea.Batch(cmd, m.diffForSelection())
	}
	return m, cmd
}

// refilter rebuilds the list items from the current projects under the filter.
func (m *Model) refilter() {
	items := make([]list.Item, 0, len(m.projects))
	for i, p := range m.projects {
		items = append(items, projectItem{idx: i, project: p})
	}
	m.applyFilter(items)
	m.rebuildSessions()
}

// diffForSelection returns a command to load the diff for the selected worktree,
// or clears the diff if nothing is selected.
func (m *Model) diffForSelection() tea.Cmd {
	w := m.selectedWorktree()
	if w == nil {
		m.diff.SetContent("")
		return nil
	}
	return diffStat(w.Path, w.BaseBranch)
}

// layout sizes the panes to the current terminal dimensions, using the shared
// geometry so View and Update never drift.
func (m *Model) layout() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	g := m.geometry()
	m.projList.SetSize(g.leftInnerW, g.leftInnerH)
	m.sessions.SetColumns(sessionColumns(g.rightInnerW))
	m.sessions.SetWidth(g.rightInnerW)
	m.sessions.SetHeight(g.sessInnerH)
	m.diff.SetWidth(g.rightInnerW)
	m.diff.SetHeight(g.diffInnerH)
}

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}
