package ui

import (
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
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

// confirmKill opens a confirmation to SIGTERM the selected live session's
// process. Ended (history) sessions have no PID and cannot be killed.
func (m Model) confirmKill() (tea.Model, tea.Cmd) {
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
