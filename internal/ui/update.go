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
		m.tasks = msg.tasks
		m.runsByProject = msg.runs
		if msg.tools != nil {
			m.tools = msg.tools
		}
		if msg.profiles != nil {
			m.profiles = msg.profiles
		}
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

	case runLaunchedMsg:
		m.toast = "launched " + msg.run.ID + " in tmux"
		m.activeTab = tabAgents
		m.runsView = true
		m.worktreeView = false
		return m, tea.Batch(collectFleet(m.cfg), m.diffForSelection())

	case spawnMsg:
		return m, spawnHarness(msg.harness, msg.dir)

	case tickMsg:
		// Re-collect and re-arm the ticker (Bubble Tea ticks once per call).
		return m, tea.Batch(collectFleet(m.cfg), tick(m.cfg.RefreshInterval.Duration))

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tea.MouseWheelMsg:
		return m.handleMouseWheel(msg)

	case tea.MouseClickMsg:
		return m.handleMouseClick(msg)
	}
	return m, nil
}

func (m Model) handleMouseWheel(msg tea.MouseWheelMsg) (tea.Model, tea.Cmd) {
	if m.level == levelDetail {
		var cmd tea.Cmd
		m.diff, cmd = m.diff.Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	if m.level == levelProjects {
		m.projList, cmd = m.projList.Update(msg)
	} else {
		m.sessions.Focus()
		m.sessions, cmd = m.sessions.Update(msg)
	}
	return m, cmd
}

func (m Model) handleMouseClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if msg.Button != tea.MouseLeft {
		return m, nil
	}
	g := m.geometry()
	leftOuterW := g.leftOuterW
	rightX := msg.X - leftOuterW
	if rightX >= 0 && rightX < g.rightOuterW && msg.Y <= 2 {
		idx := rightX * len(workTabs) / max1(g.rightOuterW)
		if idx < 0 {
			idx = 0
		}
		if idx >= len(workTabs) {
			idx = len(workTabs) - 1
		}
		return m.switchTab(workTabs[idx].tab)
	}
	if m.showProjects && msg.X < leftOuterW {
		m.level = levelProjects
		return m, nil
	}
	if msg.Y < g.sessOuterH {
		m.level = levelWork
		return m, nil
	}
	m.level = levelDetail
	return m, nil
}

// handleKey routes a key press. Modals (confirm, new-agent) take priority, then
// the filter input, then global keys and navigation.
func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	m.toast = "" // any key dismisses the last transient toast
	switch {
	case m.confirm != nil:
		return m.handleConfirmKey(msg)
	case m.launch != nil:
		return m.handleLaunchKey(msg)
	case m.prompt != nil:
		return m.handlePromptKey(msg)
	case m.pathPrompt != nil:
		return m.handlePathKey(msg)
	case m.filtering:
		return m.handleFilterKey(msg)
	case m.showActions:
		return m.handleActionsKey(msg)
	}

	switch normalizeKey(msg.String()) {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "tab":
		return m.nextPane()
	case "shift+tab":
		return m.prevPane()
	case "right":
		return m.movePaneRight()
	case "left":
		return m.movePaneLeft()
	case "/":
		m.filtering = true
		return m, m.filter.Focus()
	case "?":
		return m.openActionMenu()
	case "1":
		return m.switchTab(tabAgents)
	case "2":
		return m.switchTab(tabTasks)
	case "3":
		return m.switchTab(tabWorktrees)
	case "4":
		return m.switchTab(tabTools)
	case "5":
		return m.switchTab(tabHistory)
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
	case "o":
		if r := m.selectedRun(); r != nil {
			return m, enterReport(r.ReportPath)
		}
		m.banner = "no run report selected"
		return m, nil
	case "u":
		return m.switchTab(tabAgents)
	case "w":
		if m.activeTab == tabWorktrees {
			return m.switchTab(tabAgents)
		}
		return m.switchTab(tabWorktrees)
	case "a":
		if r := m.selectedRun(); r != nil {
			return m, attachRun(*r)
		}
		if s := m.selectedSession(); s != nil {
			return m, attachSession(*s)
		}
		return m, nil
	case "r":
		s := m.selectedSession()
		if s == nil {
			return m, nil
		}
		cmd := handoffCommand(*s)
		if cmd == "" {
			m.banner = "no resumable command for this session"
			return m, nil
		}
		m.banner = ""
		m.toast = "copied resume command — paste in a terminal"
		return m, tea.SetClipboard(cmd)
	case "k":
		return m.confirmKill()
	case "g":
		return m.confirmGC()
	case "n":
		return m, m.openLaunchPrompt()
	case "p":
		return m.toggleProjectsDrawer()
	case "s":
		if m.activeTab == tabTasks {
			m.taskScope = !m.taskScope
			m.sessions.SetCursor(0)
			m.rebuildSessions()
			return m, m.diffForSelection()
		}
		return m, nil
	case "t":
		if m.activeTab == tabTasks {
			return m.switchTab(tabAgents)
		}
		return m.switchTab(tabTasks)
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

func (m Model) handleActionsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch normalizeKey(msg.String()) {
	case "?", "esc", "backspace", "q":
		m.showActions = false
		m.actionMenu = nil
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		if m.actionMenu != nil && len(m.actionMenu.items) > 0 {
			m.actionMenu.cursor = (m.actionMenu.cursor + 1) % len(m.actionMenu.items)
		}
		return m, nil
	case "k", "up":
		if m.actionMenu != nil && len(m.actionMenu.items) > 0 {
			m.actionMenu.cursor--
			if m.actionMenu.cursor < 0 {
				m.actionMenu.cursor = len(m.actionMenu.items) - 1
			}
		}
		return m, nil
	case "enter":
		return m.runSelectedAction()
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		if m.actionMenu == nil {
			return m, nil
		}
		idx := int([]rune(normalizeKey(msg.String()))[0] - '1')
		if idx >= 0 && idx < len(m.actionMenu.items) {
			m.actionMenu.cursor = idx
			return m.runSelectedAction()
		}
	}
	return m, nil
}

func (m Model) openActionMenu() (tea.Model, tea.Cmd) {
	items := m.contextActions()
	if len(items) == 0 {
		m.banner = "nothing actionable here"
		return m, nil
	}
	m.actionMenu = &actionMenu{title: m.actionTitle(), items: items}
	m.showActions = true
	return m, nil
}

func (m Model) actionTitle() string {
	switch m.activeTab {
	case tabTasks:
		if t := m.selectedTask(); t != nil {
			return "Task: " + valueOr(t.Title, "(untitled)")
		}
		return "Tasks"
	case tabWorktrees:
		if w := m.selectedWorktree(); w != nil {
			return "Worktree: " + valueOr(w.Branch, w.Path)
		}
		return "Worktrees"
	case tabTools:
		if tool := m.selectedTool(); tool != nil {
			return "Tool: " + tool.Name
		}
		return "Tools"
	case tabHistory:
		if s := m.selectedSession(); s != nil {
			return "History: " + valueOr(s.Name, s.Harness)
		}
		return "History"
	default:
		if r := m.selectedRun(); r != nil {
			return "Run: " + valueOr(r.TaskTitle, r.ID)
		}
		if s := m.selectedSession(); s != nil {
			return "Agent: " + valueOr(s.Name, s.Harness)
		}
		return "Agents"
	}
}

func (m Model) contextActions() []actionItem {
	items := []actionItem{{label: "Open details", hint: "Show the selected item's detail pane", kind: actionOpenDetails}}
	if w := m.selectionWorktree(); w != nil {
		items = append(items, actionItem{label: "Open diff", hint: "Review changes in lazygit/git diff", kind: actionOpenDiff})
	}
	switch m.activeTab {
	case tabTasks:
		items = append(items,
			actionItem{label: "Launch run", hint: "Start selected task with a runner profile", kind: actionLaunch},
			actionItem{label: "Toggle task scope", hint: "Switch project tasks / all tasks", kind: actionToggleTaskScope},
		)
	case tabWorktrees:
		items = append(items, actionItem{label: "GC worktrees", hint: "Confirm cleanup for merged/stale worktrees", kind: actionGC})
	case tabTools:
		items = append(items, actionItem{label: "Refresh tools", hint: "Re-detect installed agent CLIs", kind: actionRefresh})
	case tabHistory:
		if m.selectedSession() != nil {
			items = append(items, actionItem{label: "Attach", hint: "Open the terminal/session if still available", kind: actionAttach})
		}
	default:
		if r := m.selectedRun(); r != nil {
			if r.TmuxTarget != "" {
				items = append(items, actionItem{label: "Attach", hint: "Attach to the run tmux target", kind: actionAttach})
			}
			if r.ReportPath != "" {
				items = append(items, actionItem{label: "Open report", hint: "Open report.md", kind: actionOpenReport})
			}
			items = append(items, actionItem{label: "Kill", hint: "Confirm stopping this run", kind: actionKill})
		} else if s := m.selectedSession(); s != nil {
			items = append(items,
				actionItem{label: "Attach", hint: "Attach/resume this agent", kind: actionAttach},
				actionItem{label: "Copy resume command", hint: "Copy a shell handoff command", kind: actionCopyResume},
				actionItem{label: "Kill", hint: "Confirm stopping the live process", kind: actionKill},
			)
		}
		items = append(items, actionItem{label: "Launch run", hint: "Start a new run in this project", kind: actionLaunch})
	}
	return items
}

func (m Model) runSelectedAction() (tea.Model, tea.Cmd) {
	if m.actionMenu == nil || len(m.actionMenu.items) == 0 {
		m.showActions = false
		m.actionMenu = nil
		return m, nil
	}
	item := m.actionMenu.items[m.actionMenu.cursor]
	m.showActions = false
	m.actionMenu = nil
	switch item.kind {
	case actionOpenDetails:
		if h, _ := m.currentDetail(); h == "" {
			return m, nil
		}
		m.level = levelDetail
		return m, m.diffForSelection()
	case actionAttach:
		if r := m.selectedRun(); r != nil {
			return m, attachRun(*r)
		}
		if s := m.selectedSession(); s != nil {
			return m, attachSession(*s)
		}
	case actionOpenDiff:
		if w := m.selectionWorktree(); w != nil {
			return m, enterDiff(*w, m.cfg)
		}
	case actionOpenReport:
		if r := m.selectedRun(); r != nil {
			return m, enterReport(r.ReportPath)
		}
		m.banner = "no run report selected"
	case actionLaunch:
		return m, m.openLaunchPrompt()
	case actionKill:
		return m.confirmKill()
	case actionGC:
		return m.confirmGC()
	case actionCopyResume:
		s := m.selectedSession()
		if s == nil {
			return m, nil
		}
		cmd := handoffCommand(*s)
		if cmd == "" {
			m.banner = "no resumable command for this session"
			return m, nil
		}
		m.toast = "copied resume command - paste in a terminal"
		return m, tea.SetClipboard(cmd)
	case actionToggleTaskScope:
		if m.activeTab == tabTasks {
			m.taskScope = !m.taskScope
			m.sessions.SetCursor(0)
			m.rebuildSessions()
			return m, m.diffForSelection()
		}
	case actionRefresh:
		return m, collectFleet(m.cfg)
	}
	return m, nil
}

func (m Model) toggleProjectsDrawer() (tea.Model, tea.Cmd) {
	m.showProjects = !m.showProjects
	if m.showProjects {
		m.level = levelProjects
	} else if m.level == levelProjects {
		m.level = levelWork
	}
	m.showActions = false
	m.actionMenu = nil
	m.layout()
	return m, m.diffForSelection()
}

func (m Model) nextPane() (tea.Model, tea.Cmd) {
	switch m.level {
	case levelProjects:
		m.level = levelWork
	case levelWork:
		if header, _ := m.currentDetail(); header != "" {
			m.level = levelDetail
		} else if m.showProjects {
			m.level = levelProjects
		}
	case levelDetail:
		if m.showProjects {
			m.level = levelProjects
		} else {
			m.level = levelWork
		}
	}
	return m, m.diffForSelection()
}

func (m Model) prevPane() (tea.Model, tea.Cmd) {
	switch m.level {
	case levelProjects:
		if header, _ := m.currentDetail(); header != "" {
			m.level = levelDetail
		} else {
			m.level = levelWork
		}
	case levelWork:
		if m.showProjects {
			m.level = levelProjects
		} else if header, _ := m.currentDetail(); header != "" {
			m.level = levelDetail
		}
	case levelDetail:
		m.level = levelWork
	}
	return m, m.diffForSelection()
}

func (m Model) movePaneRight() (tea.Model, tea.Cmd) {
	switch m.level {
	case levelProjects:
		m.level = levelWork
	case levelWork:
		if header, _ := m.currentDetail(); header != "" {
			m.level = levelDetail
		}
	}
	return m, m.diffForSelection()
}

func (m Model) movePaneLeft() (tea.Model, tea.Cmd) {
	switch m.level {
	case levelDetail:
		m.level = levelWork
	case levelWork:
		if m.showProjects {
			m.level = levelProjects
		}
	}
	return m, nil
}

// drillDown moves one navigation level deeper (projects → work → detail),
// refreshing the detail pane. It is a no-op at the deepest level or when the
// work level has no session to open.
func (m Model) drillDown() (tea.Model, tea.Cmd) {
	switch m.level {
	case levelProjects:
		m.showProjects = false
		m.level = levelWork
		m.layout()
		return m, m.diffForSelection()
	case levelWork:
		if h, _ := m.currentDetail(); h == "" {
			return m, nil
		}
		return m.openActionMenu()
	}
	return m, nil
}

// drillUp climbs one navigation level back toward the projects list.
func (m Model) drillUp() (tea.Model, tea.Cmd) {
	switch m.level {
	case levelDetail:
		m.level = levelWork
	case levelWork:
		if !m.showProjects {
			m.showProjects = true
			m.layout()
		}
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

// handleFilterKey feeds keys to the filter input, applying it live. The filter
// narrows the tasks browser when it is open, else the projects list.
func (m Model) handleFilterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filtering = false
		m.filter.Blur()
		m.filter.Reset()
		m.applyActiveFilter()
		return m, m.diffForSelection()
	case "enter":
		m.filtering = false
		m.filter.Blur()
		m.applyActiveFilter()
		return m, m.diffForSelection()
	}

	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	m.applyActiveFilter()
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
	if m.activeTab == tabHistory {
		m.showHistory = false
		return m.switchTab(tabAgents)
	}
	return m.switchTab(tabHistory)
}

// ensureHistory returns a command to load history for the selected project when
// history is shown and not yet cached; nil otherwise.
func (m *Model) ensureHistory() tea.Cmd {
	if m.activeTab != tabHistory && !m.showHistory {
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

func (m Model) switchTab(tab workTab) (tea.Model, tea.Cmd) {
	m.activeTab = tab
	m.showActions = false
	m.actionMenu = nil
	m.showTasks = false
	m.worktreeView = tab == tabWorktrees
	m.runsView = tab == tabAgents
	m.showHistory = tab == tabHistory
	if m.level == levelProjects || m.level == levelDetail {
		m.level = levelWork
	}
	m.sessions.SetCursor(0)
	m.rebuildSessions()
	return m, tea.Batch(m.ensureHistory(), m.diffForSelection())
}

// navigateSessions forwards a key to the sessions table and refreshes the diff if
// the selection moved.
func (m Model) navigateSessions(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	prev := m.sessions.Cursor()
	var cmd tea.Cmd
	m.sessions.Focus()
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

func (m *Model) applyActiveFilter() {
	if m.activeTab == tabTasks || m.activeTab == tabTools {
		m.sessions.SetCursor(0)
		m.rebuildSessions()
		return
	}
	m.refilter()
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
		content := header
		if m.activeTab == tabAgents || m.activeTab == tabHistory {
			content += "\n\n" + dim.Render("No worktree is attached, so diff is unavailable.")
		}
		m.diff.SetContent(content)
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
	return dim.Render(fmt.Sprintf("%s — nothing selected in this tab\n\npress 2 for tasks · n to launch a run · ? for actions", p.Name))
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
	m.sessions.SetHeight(max1(g.sessInnerH - workPaneHeaderLines))
	m.diff.SetWidth(g.rightInnerW)
	m.diff.SetHeight(g.diffInnerH)

	// The tasks browser is a single full-width pane above the footer.
	m.tasksVP.SetWidth(max1(m.width - 2))
	m.tasksVP.SetHeight(max1(m.height - footerHeight - 2))
}

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}
