package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/table"

	"github.com/KirillSachkov/gvardia/internal/model"
)

// projectItem adapts a model.Project to the list. idx indexes back into
// Model.projects so selection survives filtering.
type projectItem struct {
	idx     int
	project model.Project
}

// Title is the project name (list DefaultItem).
func (p projectItem) Title() string { return p.project.Name }

// Description summarizes worktree and live-agent counts (list DefaultItem).
func (p projectItem) Description() string {
	return fmt.Sprintf("%d wt · %d live", len(p.project.Worktrees), p.project.LiveAgents)
}

// FilterValue is what the filter matches against (list.Item).
func (p projectItem) FilterValue() string { return p.project.Name }

// matches reports whether the project matches a filter query (name or any branch).
func (p projectItem) matches(q string) bool {
	q = strings.ToLower(q)
	if strings.Contains(strings.ToLower(p.project.Name), q) {
		return true
	}
	for _, w := range p.project.Worktrees {
		if strings.Contains(strings.ToLower(w.Branch), q) {
			return true
		}
	}
	return false
}

// sessionColumns returns the sessions-table columns sized to the given width.
func sessionColumns(width int) []table.Column {
	// Fixed glyph/harness columns; branch/name take the remainder.
	rest := width - 2 - 8 - 12
	if rest < 10 {
		rest = 10
	}
	return []table.Column{
		{Title: "", Width: 2},
		{Title: "harness", Width: 8},
		{Title: "agent", Width: 12},
		{Title: "branch", Width: rest},
	}
}

// worktreeRow renders one worktree (with its lead session, if any) as a table row.
func worktreeRow(w model.Worktree) table.Row {
	branch := w.Branch
	if branch == "" {
		branch = "(detached)"
	}
	branch += worktreeSuffix(w)

	harness, agent := "", ""
	if len(w.Sessions) > 0 {
		s := w.Sessions[0]
		harness = s.Harness
		agent = s.Name
		if len(w.Sessions) > 1 {
			agent += fmt.Sprintf(" +%d", len(w.Sessions)-1)
		}
	}
	return table.Row{glyph(w), harness, agent, branch}
}

// worktreeSuffix appends dirty / ahead-behind flags to a branch label.
func worktreeSuffix(w model.Worktree) string {
	var parts []string
	if w.Dirty {
		parts = append(parts, "✱")
	}
	if w.Ahead > 0 || w.Behind > 0 {
		parts = append(parts, fmt.Sprintf("↑%d↓%d", w.Ahead, w.Behind))
	}
	if len(parts) == 0 {
		return ""
	}
	return " " + strings.Join(parts, " ")
}

// glyph is the status marker for a worktree's lead session (space if no agent).
func glyph(w model.Worktree) string {
	if len(w.Sessions) == 0 {
		return " "
	}
	s := w.Sessions[0]
	switch {
	case s.Status == model.StatusFailed:
		return "✖"
	case s.Harness == "codex":
		return "◍"
	case s.Status == model.StatusBusy:
		return "●"
	case s.Status == model.StatusIdle:
		return "○"
	default:
		return "·"
	}
}
