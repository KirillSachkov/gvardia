package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/table"

	"github.com/KirillSachkov/gvardia/internal/model"
	"github.com/KirillSachkov/gvardia/internal/runs"
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

// sessionColumns returns the work-pane columns sized to the given width; branch
// absorbs the slack.
func sessionColumns(width int) []table.Column {
	const state, harness, agent, task, delta, last = 2, 7, 16, 6, 9, 5
	branch := width - (state + harness + agent + task + delta + last)
	if branch < 8 {
		branch = 8
	}
	return []table.Column{
		{Title: "", Width: state},
		{Title: "harness", Width: harness},
		{Title: "agent", Width: agent},
		{Title: "task", Width: task},
		{Title: "branch", Width: branch},
		{Title: "Δ", Width: delta},
		{Title: "last", Width: last},
	}
}

// sessionRow renders one work-session as a table row.
func sessionRow(s model.Session) table.Row {
	branch := s.Branch
	if branch == "" {
		branch = "(detached)"
	}
	task := s.Task
	if task == "" {
		task = "—"
	}
	delta := ""
	if s.ChangeStat.Files > 0 {
		delta = fmt.Sprintf("+%d/-%d", s.ChangeStat.Added, s.ChangeStat.Removed)
	}
	return table.Row{sessionGlyph(s), s.Harness, s.Name, task, branch, delta, relativeTime(s.LastActivity)}
}

func runColumns(width int) []table.Column {
	const state, runner, task, tmux, last = 8, 10, 24, 18, 5
	branch := width - (state + runner + task + tmux + last)
	if branch < 8 {
		branch = 8
	}
	return []table.Column{
		{Title: "status", Width: state},
		{Title: "runner", Width: runner},
		{Title: "task", Width: task},
		{Title: "branch", Width: branch},
		{Title: "tmux", Width: tmux},
		{Title: "last", Width: last},
	}
}

func runRow(r runs.Run) table.Row {
	task := r.TaskTitle
	if task == "" {
		task = "—"
	}
	branch := r.Branch
	if branch == "" {
		branch = "(none)"
	}
	tmux := r.TmuxTarget
	if tmux == "" {
		tmux = "—"
	}
	return table.Row{runStatusLabel(r.Status), r.Runner, task, branch, tmux, relativeTime(r.UpdatedAt)}
}

func runStatusLabel(status runs.Status) string {
	switch status {
	case runs.StatusRunning:
		return "● run"
	case runs.StatusReview:
		return "◆ review"
	case runs.StatusDone:
		return "✓ done"
	case runs.StatusFailed:
		return "✖ fail"
	case runs.StatusKilled:
		return "■ killed"
	default:
		return "○ pending"
	}
}

// worktreeColumns returns the worktree-pane columns sized to the given width;
// branch absorbs the slack.
func worktreeColumns(width int) []table.Column {
	const state, ab, delta, agent, commit = 2, 7, 9, 14, 6
	branch := width - (state + ab + delta + agent + commit)
	if branch < 8 {
		branch = 8
	}
	return []table.Column{
		{Title: "", Width: state},
		{Title: "branch", Width: branch},
		{Title: "±", Width: ab},
		{Title: "Δ", Width: delta},
		{Title: "agent", Width: agent},
		{Title: "commit", Width: commit},
	}
}

// worktreeRow2 renders one worktree as a table row, marking whether a live agent
// runs in it (from the joined sessions).
func worktreeRow2(w model.Worktree) table.Row {
	branch := w.Branch
	if branch == "" {
		branch = "(detached)"
	}
	if w.IsPrimary {
		branch += " *"
	}
	ab := ""
	if w.Ahead > 0 || w.Behind > 0 {
		ab = fmt.Sprintf("↑%d↓%d", w.Ahead, w.Behind)
	}
	delta := ""
	if w.ChangeStat.Files > 0 {
		delta = fmt.Sprintf("+%d/-%d", w.ChangeStat.Added, w.ChangeStat.Removed)
	}
	agent := "·"
	if len(w.Sessions) > 0 {
		agent = "● " + w.Sessions[0].Harness
		if len(w.Sessions) > 1 {
			agent = fmt.Sprintf("● %s +%d", w.Sessions[0].Harness, len(w.Sessions)-1)
		}
	}
	return table.Row{worktreeGlyph(w), branch, ab, delta, agent, relativeTime(w.LastCommit)}
}

// worktreeGlyph marks a worktree dirty (◆) or clean (◇).
func worktreeGlyph(w model.Worktree) string {
	if w.Dirty {
		return "◆"
	}
	return "◇"
}

// sessionGlyph is the status marker: ended sessions get ✓; live sessions show
// their run state.
func sessionGlyph(s model.Session) string {
	if !s.Live {
		return "✓"
	}
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

// relativeTime renders a compact "time since" (now/5m/3h/2d); empty for zero.
func relativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return relativeAge(time.Since(t))
}

func relativeAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
