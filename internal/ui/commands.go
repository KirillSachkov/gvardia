package ui

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/KirillSachkov/gvardia/internal/adapters"
	"github.com/KirillSachkov/gvardia/internal/collect"
	"github.com/KirillSachkov/gvardia/internal/config"
	"github.com/KirillSachkov/gvardia/internal/model"
)

// fleetMsg carries a completed collect+adapters+join pass.
type fleetMsg struct {
	projects []model.Project
	failures []adapters.Failure
}

// errMsg carries a fatal-to-this-pass error (rendered as a banner, not a crash).
type errMsg struct{ err error }

// tickMsg is the periodic refresh trigger.
type tickMsg time.Time

// diffMsg carries the diff stat for a worktree.
type diffMsg struct {
	path    string
	content string
}

// collectFleet runs the collectors and adapters and joins them. It is pure I/O,
// safe to run inside a tea.Cmd.
func collectFleet(cfg config.Config) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		projects, err := collect.Collect(ctx, collect.Git{}, cfg)
		if err != nil {
			return errMsg{err}
		}
		sessions, failures := adapters.CollectSessions(ctx, adapters.Enabled(cfg))
		projects = collect.Join(projects, sessions)
		return fleetMsg{projects: projects, failures: failures}
	}
}

// tick schedules the next periodic refresh.
func tick(interval time.Duration) tea.Cmd {
	return tea.Every(interval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// execDoneMsg signals that an external program (lazygit/git diff) has exited and
// the TUI has the terminal back.
type execDoneMsg struct{}

// enterDiff hands the terminal to an interactive diff viewer for the worktree and
// returns to the TUI when it exits.
func enterDiff(wt model.Worktree, cfg config.Config) tea.Cmd {
	cmd := diffCommand(wt, cfg)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return errMsg{fmt.Errorf("diff viewer: %w", err)}
		}
		return execDoneMsg{}
	})
}

// diffCommand builds the interactive diff command: lazygit rooted at the worktree
// (via cwd, which handles linked worktrees whose .git is a file), or a git-diff
// fallback through delta when lazygit is absent.
func diffCommand(wt model.Worktree, cfg config.Config) *exec.Cmd {
	lazygit := cfg.Commands.Lazygit
	if lazygit == "" {
		lazygit = "lazygit"
	}
	if _, err := exec.LookPath(lazygit); err == nil {
		cmd := exec.Command(lazygit)
		cmd.Dir = wt.Path
		return cmd
	}

	rangeArg := "HEAD"
	if wt.BaseBranch != "" {
		rangeArg = wt.BaseBranch + "...HEAD"
	}
	args := []string{"-C", wt.Path}
	if _, err := exec.LookPath("delta"); err == nil {
		args = append(args, "-c", "core.pager=delta")
	}
	args = append(args, "diff", rangeArg)
	return exec.Command("git", args...)
}

// diffStat computes `git -C <path> diff --stat <base>...HEAD` for a worktree.
func diffStat(path, base string) tea.Cmd {
	return func() tea.Msg {
		if path == "" {
			return diffMsg{}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		rangeArg := "HEAD"
		if base != "" {
			rangeArg = fmt.Sprintf("%s...HEAD", base)
		}
		cmd := exec.CommandContext(ctx, "git", "-C", path, "diff", "--stat", rangeArg)
		out, err := cmd.Output()
		content := strings.TrimRight(string(out), "\n")
		if err != nil {
			content = fmt.Sprintf("no diff available (%v)", err)
		} else if content == "" {
			content = "no changes vs " + rangeArg
		}
		return diffMsg{path: path, content: content}
	}
}
