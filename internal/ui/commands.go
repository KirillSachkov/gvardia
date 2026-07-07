package ui

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/KirillSachkov/gvardia/internal/adapters"
	"github.com/KirillSachkov/gvardia/internal/collect"
	"github.com/KirillSachkov/gvardia/internal/config"
	"github.com/KirillSachkov/gvardia/internal/history"
	"github.com/KirillSachkov/gvardia/internal/model"
)

// fleetMsg carries a completed collect+adapters+join pass. curated is true when
// the projects came from the tracked list rather than a roots scan.
type fleetMsg struct {
	projects []model.Project
	failures []adapters.Failure
	curated  bool
}

// projectsChangedMsg signals that the tracked project list was edited and the
// fleet should be re-collected.
type projectsChangedMsg struct{}

// errMsg carries a fatal-to-this-pass error (rendered as a banner, not a crash).
type errMsg struct{ err error }

// tickMsg is the periodic refresh trigger.
type tickMsg time.Time

// historyMsg carries lazily-loaded past sessions for a project.
type historyMsg struct {
	projectPath string
	sessions    []model.Session
}

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

		tracked, _ := config.LoadTracked()
		curated := len(tracked) > 0

		var projects []model.Project
		var err error
		if curated {
			projects, err = collect.CollectTracked(ctx, collect.Git{}, cfg, tracked)
		} else {
			projects, err = collect.Collect(ctx, collect.Git{}, cfg)
		}
		if err != nil {
			return errMsg{err}
		}
		sessions, failures := adapters.CollectSessions(ctx, adapters.Enabled(cfg))
		projects = collect.AssembleLive(ctx, collect.Git{}, projects, sessions)
		attachSummaries(ctx, history.New(), projects)
		return fleetMsg{projects: projects, failures: failures, curated: curated}
	}
}

// addTracked appends path to the curated list (deduplicated) and persists it.
func addTracked(path string) error {
	tracked, err := config.LoadTracked()
	if err != nil {
		return err
	}
	for _, p := range tracked {
		if p == path {
			return nil // already tracked
		}
	}
	return config.SaveTracked(append(tracked, path))
}

// trackProject validates that path is a git repo, then adds it to the curated
// list. It runs as a tea.Cmd (pure I/O).
func trackProject(path string) tea.Cmd {
	return func() tea.Msg {
		path = absExpand(path)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := exec.CommandContext(ctx, "git", "-C", path, "rev-parse", "--is-inside-work-tree").Run(); err != nil {
			return errMsg{fmt.Errorf("not a git repo: %s", path)}
		}
		if err := addTracked(path); err != nil {
			return errMsg{err}
		}
		return projectsChangedMsg{}
	}
}

// createProject git-inits a new repo at path, then tracks it.
func createProject(path string) tea.Cmd {
	return func() tea.Msg {
		path = absExpand(path)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if out, err := exec.CommandContext(ctx, "git", "init", path).CombinedOutput(); err != nil {
			return errMsg{fmt.Errorf("git init: %w: %s", err, strings.TrimSpace(string(out)))}
		}
		if err := addTracked(path); err != nil {
			return errMsg{err}
		}
		return projectsChangedMsg{}
	}
}

// untrackProject removes path from the curated list (never touches the repo).
func untrackProject(path string) tea.Cmd {
	return func() tea.Msg {
		tracked, err := config.LoadTracked()
		if err != nil {
			return errMsg{err}
		}
		kept := make([]string, 0, len(tracked))
		for _, p := range tracked {
			if p != path {
				kept = append(kept, p)
			}
		}
		if err := config.SaveTracked(kept); err != nil {
			return errMsg{err}
		}
		return projectsChangedMsg{}
	}
}

// absExpand expands a leading "~" and makes the path absolute where possible.
func absExpand(path string) string {
	path = config.ExpandPath(path)
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}

// attachSummaries fills each live work-session's Summary from its transcript.
func attachSummaries(ctx context.Context, hist history.Reader, projects []model.Project) {
	for pi := range projects {
		for si := range projects[pi].WorkSessions {
			s := &projects[pi].WorkSessions[si]
			if s.Summary == "" {
				s.Summary = hist.SummaryFor(ctx, s.Harness, s.SessionID, s.Cwd)
			}
		}
	}
}

// loadHistory fetches recent past sessions for a project's primary cwd.
func loadHistory(projectPath string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		sessions := history.New().Recent(ctx, projectPath, history.Options{Limit: 8, Since: 14 * 24 * time.Hour})
		return historyMsg{projectPath: projectPath, sessions: sessions}
	}
}

// tick schedules the next periodic refresh.
func tick(interval time.Duration) tea.Cmd {
	return tea.Every(interval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// execDoneMsg signals that an external program (lazygit/git diff) has exited and
// the TUI has the terminal back.
type execDoneMsg struct{}

// spawnMsg asks the UI to launch a harness in dir once its worktree is ready.
type spawnMsg struct {
	harness string
	dir     string
}

// sessionExec builds the attach/resume command for a session. attach enables
// tmux attach for tmux sessions; without it tmux has no resume command. Returns
// nil when there is nothing to run.
func sessionExec(s model.Session, attach bool) *exec.Cmd {
	dir := s.WorktreePath
	if dir == "" {
		dir = s.Cwd
	}
	switch s.Harness {
	case "claude":
		cmd := exec.Command("claude", "--resume", s.SessionID)
		cmd.Dir = dir
		return cmd
	case "codex":
		args := []string{"resume", "--last"}
		if s.SessionID != "" {
			args = []string{"resume", s.SessionID}
		}
		cmd := exec.Command("codex", args...)
		cmd.Dir = dir
		return cmd
	case "tmux":
		if attach && s.SessionID != "" {
			return exec.Command("tmux", "attach", "-t", s.SessionID)
		}
		return nil
	default:
		return nil
	}
}

// attachSession hands the terminal to the selected session (tmux-attach aware).
func attachSession(s model.Session) tea.Cmd { return execOrBanner(sessionExec(s, true)) }

// resumeSession resumes the selected session's harness (claude/codex).
func resumeSession(s model.Session) tea.Cmd { return execOrBanner(sessionExec(s, false)) }

// execOrBanner runs cmd interactively, or reports a banner if there is no command.
func execOrBanner(cmd *exec.Cmd) tea.Cmd {
	if cmd == nil {
		return func() tea.Msg { return errMsg{errors.New("no attachable session here")} }
	}
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return errMsg{fmt.Errorf("attach: %w", err)}
		}
		return execDoneMsg{}
	})
}

// killSession sends SIGTERM to a session PID, then refreshes so it drops off.
func killSession(pid int) tea.Cmd {
	return func() tea.Msg {
		if pid <= 0 {
			return errMsg{errors.New("session has no PID to kill")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := exec.CommandContext(ctx, "kill", "-TERM", strconv.Itoa(pid)).Run(); err != nil {
			return errMsg{fmt.Errorf("kill %d: %w", pid, err)}
		}
		return execDoneMsg{}
	}
}

// gcRoot runs wt-prune (which preserves primary/dirty worktrees) for a root, then
// refreshes. wt-prune is expected on PATH (installed in Phase 6).
func gcRoot(root string) tea.Cmd {
	cmd := exec.Command("wt-prune", "--yes", root)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return errMsg{fmt.Errorf("wt-prune: %w", err)}
		}
		return execDoneMsg{}
	})
}

// newAgent starts a new agent. claude creates its own worktree+tmux session;
// other harnesses get a fresh `git worktree` first, then the CLI is spawned there.
func newAgent(project model.Project, harness, name string) tea.Cmd {
	if name == "" {
		return func() tea.Msg { return errMsg{errors.New("agent name required")} }
	}
	if harness == "claude" {
		cmd := exec.Command("claude", "-w", name, "--tmux")
		cmd.Dir = project.Path
		return tea.ExecProcess(cmd, func(err error) tea.Msg {
			if err != nil {
				return errMsg{fmt.Errorf("new claude agent: %w", err)}
			}
			return execDoneMsg{}
		})
	}

	// Generic path: create the worktree, then ask the UI to spawn the harness.
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		wtPath := filepath.Join(filepath.Dir(project.Path), project.Name+"-"+name)
		add := exec.CommandContext(ctx, "git", "-C", project.Path, "worktree", "add", wtPath, "-b", name)
		if out, err := add.CombinedOutput(); err != nil {
			return errMsg{fmt.Errorf("git worktree add: %w: %s", err, strings.TrimSpace(string(out)))}
		}
		return spawnMsg{harness: harness, dir: wtPath}
	}
}

// spawnHarness runs a bare harness CLI in dir, returning to the TUI on exit.
func spawnHarness(harness, dir string) tea.Cmd {
	cmd := exec.Command(harness)
	cmd.Dir = dir
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return errMsg{fmt.Errorf("spawn %s: %w", harness, err)}
		}
		return execDoneMsg{}
	})
}

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
