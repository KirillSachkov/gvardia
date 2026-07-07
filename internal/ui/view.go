package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/KirillSachkov/gvardia/internal/adapters"
	"github.com/KirillSachkov/gvardia/internal/model"
)

// footerHeight is the number of lines reserved below the body: a status line and
// the keybind footer.
const footerHeight = 2

var (
	borderActive   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))
	borderInactive = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
	dim            = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	warn           = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
)

// geo holds the derived pane geometry for the current terminal size.
type geo struct {
	leftInnerW, leftInnerH int
	rightInnerW            int
	sessInnerH, diffInnerH int
}

// geometry derives pane sizes from the terminal dimensions. It is the single
// source of layout truth, used by both layout() and View().
func (m Model) geometry() geo {
	bodyH := m.height - footerHeight
	if bodyH < 3 {
		bodyH = 3
	}
	leftW := m.width * 34 / 100
	if leftW < 28 {
		leftW = 28
	}
	if leftW > m.width-20 {
		leftW = m.width - 20
	}
	rightW := m.width - leftW
	sessH := bodyH / 2

	return geo{
		leftInnerW: max1(leftW - 2), leftInnerH: max1(bodyH - 2),
		rightInnerW: max1(rightW - 2),
		sessInnerH:  max1(sessH - 2), diffInnerH: max1(bodyH - sessH - 2),
	}
}

// View renders the cockpit into a full-screen (alt-screen) view.
func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

// render composes the three bordered panes plus a status line and footer.
func (m Model) render() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	if m.loading && len(m.projects) == 0 {
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center, "collecting fleet…")
	}

	if m.showTasks {
		return lipgloss.JoinVertical(lipgloss.Left,
			pane(true, m.tasksVP.View()), m.statusLine(), m.footer())
	}

	left := pane(m.level == levelProjects, m.projList.View())
	sess := pane(m.level == levelWork, m.sessions.View())
	diff := pane(m.level == levelDetail, m.diff.View())
	right := lipgloss.JoinVertical(lipgloss.Left, sess, diff)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	return lipgloss.JoinVertical(lipgloss.Left, body, m.statusLine(), m.footer())
}

// pane wraps content in a rounded border, brighter when focused.
func pane(focused bool, content string) string {
	if focused {
		return borderActive.Render(content)
	}
	return borderInactive.Render(content)
}

// statusLine shows a modal prompt, the filter, an adapter banner, or a summary.
func (m Model) statusLine() string {
	switch {
	case m.confirm != nil:
		return warn.Render(truncate(m.confirm.message+" (y/n)", m.width))
	case m.prompt != nil:
		return dim.Render("new "+m.prompt.harness+" agent: ") + m.prompt.input.View()
	case m.pathPrompt != nil:
		label := "add project: "
		if m.pathPrompt.mode == pathCreate {
			label = "create project: "
		}
		return dim.Render(label) + m.pathPrompt.input.View()
	case m.filtering:
		return dim.Render("filter: ") + m.filter.Value() + dim.Render("▏")
	case m.toast != "":
		return dim.Render(truncate("✓ "+m.toast, m.width))
	case m.banner != "":
		return warn.Render(truncate("⚠ "+m.banner, m.width))
	case m.showTasks:
		scope := "all projects"
		if m.taskScope {
			scope = "selected project"
		}
		return dim.Render(truncate(fmt.Sprintf("kanban · %s · p to toggle scope", scope), m.width))
	default:
		agents := 0
		for _, p := range m.projects {
			agents += p.LiveAgents
		}
		line := fmt.Sprintf("%d projects · %d live agents", len(m.projects), agents)
		if !m.curated {
			line += " · A to curate"
		}
		return dim.Render(truncate(line, m.width))
	}
}

// footer renders the keybind hints for the current mode.
func (m Model) footer() string {
	keys := "↑↓ nav · enter drill · esc back · d diff · w worktrees · t tasks · h history · a attach · r handoff · n new · A add · X untrack · C create · k kill · g gc · / filter · R · q"
	switch {
	case m.confirm != nil:
		keys = "y confirm · n cancel"
	case m.prompt != nil:
		keys = "tab harness · enter create · esc cancel"
	case m.pathPrompt != nil:
		keys = "enter confirm · esc cancel"
	case m.filtering:
		keys = "type to filter · enter apply · esc cancel"
	case m.showTasks:
		keys = "↑↓ scroll · p project scope · / filter · esc close · R refresh"
	}
	return dim.Render(truncate(keys, m.width))
}

// detailHeader renders the selected session's summary and a metadata line for
// the top of the detail pane.
func detailHeader(s model.Session) string {
	summary := s.Summary
	if summary == "" {
		summary = "(no summary)"
	}
	branch := s.Branch
	if branch == "" {
		branch = "(detached)"
	}
	task := ""
	if s.Task != "" {
		task = " · " + s.Task
	}
	state := "ended"
	if s.Live {
		state = string(s.Status)
	}
	meta := fmt.Sprintf("%s %s%s · %s · %d files +%d -%d · %s",
		state, s.Harness, task, branch,
		s.ChangeStat.Files, s.ChangeStat.Added, s.ChangeStat.Removed, relativeTime(s.LastActivity))
	return summary + "\n" + dim.Render(meta)
}

// worktreeHeader renders the selected worktree's path and a metadata line for
// the top of the detail pane in the worktree view.
func worktreeHeader(w model.Worktree) string {
	branch := w.Branch
	if branch == "" {
		branch = "(detached)"
	}
	kind := "linked"
	if w.IsPrimary {
		kind = "primary"
	}
	state := "clean"
	if w.Dirty {
		state = "dirty"
	}
	agent := "no agent"
	if len(w.Sessions) > 0 {
		agent = w.Sessions[0].Harness
		if len(w.Sessions) > 1 {
			agent = fmt.Sprintf("%s +%d", w.Sessions[0].Harness, len(w.Sessions)-1)
		}
	}
	meta := fmt.Sprintf("%s %s · %s · ↑%d↓%d · %d files +%d -%d · %s · %s",
		kind, branch, state, w.Ahead, w.Behind,
		w.ChangeStat.Files, w.ChangeStat.Added, w.ChangeStat.Removed, agent, relativeTime(w.LastCommit))
	return w.Path + "\n" + dim.Render(meta)
}

// failureBanner summarizes skipped adapters for the status line.
func failureBanner(failures []adapters.Failure) string {
	if len(failures) == 0 {
		return ""
	}
	names := make([]string, 0, len(failures))
	for _, f := range failures {
		names = append(names, f.Adapter)
	}
	return "adapters skipped: " + strings.Join(names, ", ")
}

// truncate clips s to at most width display cells, appending an ellipsis.
func truncate(s string, width int) string {
	if width <= 0 || lipgloss.Width(s) <= width {
		return s
	}
	if width <= 1 {
		return "…"
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}
