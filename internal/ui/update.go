package ui

import (
	"fmt"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"

	"github.com/KirillSachkov/gvardia/internal/model"
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
		m.curated = msg.curated
		m.banner = failureBanner(msg.failures)
		m.setProjects(msg.projects)
		return m, tea.Batch(m.diffForSelection(), m.ensureHistory())

	case projectsChangedMsg:
		return m, collectFleet(m.cfg)

	case errMsg:
		m.loading = false
		m.banner = msg.err.Error()
		return m, nil

	case diffMsg:
		header, _ := m.currentDetail()
		if header != "" {
			header += "\n\n"
		}
		m.diff.SetContent(header + msg.content)
		m.diff.GotoTop()
		return m, nil

	case historyMsg:
		m.historyByProject[msg.projectPath] = msg.sessions
		m.rebuildSessions()
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
	case m.pathPrompt != nil:
		return m.handlePathKey(msg)
	case m.filtering:
		return m.handleFilterKey(msg)
	}

	switch normalizeKey(msg.String()) {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "tab":
		if m.level == levelProjects {
			m.level = levelWork
		} else {
			m.level = levelProjects
		}
		return m, m.diffForSelection()
	case "/":
		m.filtering = true
		return m, m.filter.Focus()
	case "R":
		return m, collectFleet(m.cfg)
	case "h":
		return m.toggleHistory()
	case "enter":
		return m.drillDown()
	case "esc", "backspace":
		return m.drillUp()
	case "d":
		if w := m.selectionWorktree(); w != nil {
			return m, enterDiff(*w, m.cfg)
		}
		return m, nil
	case "w":
		m.worktreeView = !m.worktreeView
		m.sessions.SetCursor(0)
		m.rebuildSessions()
		return m, m.diffForSelection()
	case "a":
		if s := m.selectedSession(); s != nil {
			return m, attachSession(*s)
		}
		return m, nil
	case "r":
		if s := m.selectedSession(); s != nil {
			return m, resumeSession(*s)
		}
		return m, nil
	case "k":
		return m.confirmKill()
	case "g":
		return m.confirmGC()
	case "n":
		return m, m.openNewAgentPrompt()
	case "A":
		return m, m.openPathPrompt(pathAdd)
	case "C":
		return m, m.openPathPrompt(pathCreate)
	case "X":
		return m.confirmUntrack()
	}

	switch m.level {
	case levelProjects:
		return m.navigateProjects(msg)
	case levelWork:
		return m.navigateSessions(msg)
	default: // levelDetail
		return m.navigateDiff(msg)
	}
}

// drillDown moves one navigation level deeper (projects → work → detail),
// refreshing the detail pane. It is a no-op at the deepest level or when the
// work level has no session to open.
func (m Model) drillDown() (tea.Model, tea.Cmd) {
	switch m.level {
	case levelProjects:
		m.level = levelWork
		return m, m.diffForSelection()
	case levelWork:
		if h, _ := m.currentDetail(); h == "" {
			return m, nil
		}
		m.level = levelDetail
		return m, m.diffForSelection()
	}
	return m, nil
}

// drillUp climbs one navigation level back toward the projects list.
func (m Model) drillUp() (tea.Model, tea.Cmd) {
	switch m.level {
	case levelDetail:
		m.level = levelWork
	case levelWork:
		m.level = levelProjects
	}
	return m, nil
}

// navigateDiff scrolls the detail/diff viewport at the deepest level.
func (m Model) navigateDiff(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.diff, cmd = m.diff.Update(msg)
	return m, cmd
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
		return m, tea.Batch(cmd, m.diffForSelection(), m.ensureHistory())
	}
	return m, cmd
}

// toggleHistory flips the history flag, lazily loading it for the selected
// project the first time it is shown.
func (m Model) toggleHistory() (tea.Model, tea.Cmd) {
	m.showHistory = !m.showHistory
	cmd := m.ensureHistory()
	m.rebuildSessions()
	return m, cmd
}

// ensureHistory returns a command to load history for the selected project when
// history is shown and not yet cached; nil otherwise.
func (m *Model) ensureHistory() tea.Cmd {
	if !m.showHistory {
		return nil
	}
	p := m.selectedProject()
	if p == nil {
		return nil
	}
	if _, ok := m.historyByProject[p.Path]; ok {
		return nil
	}
	return loadHistory(p.Path)
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

// diffForSelection shows the current selection's detail header immediately and
// returns a command to load its worktree diff. The detail pane is never left
// blank: with no selection it explains why and how to populate the view, and an
// orphaned session shows its header without a diff.
func (m *Model) diffForSelection() tea.Cmd {
	header, w := m.currentDetail()
	if header == "" {
		m.diff.SetContent(emptyDetail(m.selectedProject()))
		m.diff.GotoTop()
		return nil
	}
	if w == nil {
		m.diff.SetContent(header + "\n\n" + dim.Render("worktree gone — history only"))
		m.diff.GotoTop()
		return nil
	}
	m.diff.SetContent(header + "\n\nloading diff…")
	m.diff.GotoTop()
	return diffStat(w.Path, w.BaseBranch)
}

// selectionWorktree returns the worktree behind the current selection (either
// view), for the diff key.
func (m *Model) selectionWorktree() *model.Worktree {
	_, w := m.currentDetail()
	return w
}

// emptyDetail is the placeholder shown when no session is selected, so the
// detail pane never renders blank.
func emptyDetail(p *model.Project) string {
	if p == nil {
		return dim.Render("no project selected")
	}
	return dim.Render(fmt.Sprintf("%s — no active sessions\n\npress h for history · n for new agent", p.Name))
}

// layout sizes the panes to the current terminal dimensions, using the shared
// geometry so View and Update never drift.
func (m *Model) layout() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	g := m.geometry()
	m.projList.SetSize(g.leftInnerW, g.leftInnerH)
	m.applyColumns()
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
