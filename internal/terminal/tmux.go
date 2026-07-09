// Package terminal manages external terminal backends. v1 uses tmux only.
package terminal

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
)

// Runner runs an external command. Tests fake it.
type Runner interface {
	Run(ctx context.Context, dir, name string, args ...string) ([]byte, error)
}

// ExecRunner is the production Runner.
type ExecRunner struct{}

// Run executes name in dir and returns combined output.
func (ExecRunner) Run(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

// TmuxService launches and controls tmux sessions.
type TmuxService struct {
	Runner Runner
}

// LaunchSpec describes a tmux-backed run launch.
type LaunchSpec struct {
	RunID       string
	Worktree    string
	Command     string
	Target      string
	WindowTitle string
}

// Launch starts command in a detached tmux session and returns its target.
func (s TmuxService) Launch(ctx context.Context, spec LaunchSpec) (string, error) {
	if spec.Command == "" {
		return "", errors.New("launch command is required")
	}
	target := spec.Target
	if target == "" {
		if spec.RunID == "" {
			return "", errors.New("run id is required")
		}
		target = "gvardia-" + spec.RunID
	}
	title := spec.WindowTitle
	if title == "" {
		title = target
	}
	_, err := s.runner().Run(ctx, "", "tmux",
		"new-session", "-d",
		"-s", target,
		"-c", spec.Worktree,
		"-n", title,
		"sh", "-lc", spec.Command,
	)
	if err != nil {
		return "", fmt.Errorf("tmux launch: %w", err)
	}
	return target, nil
}

// Attach attaches the current terminal to a tmux target.
func (s TmuxService) Attach(ctx context.Context, target string) error {
	if target == "" {
		return errors.New("tmux target is required")
	}
	_, err := s.runner().Run(ctx, "", "tmux", "attach", "-t", target)
	if err != nil {
		return fmt.Errorf("tmux attach: %w", err)
	}
	return nil
}

// Kill terminates a tmux session.
func (s TmuxService) Kill(ctx context.Context, target string) error {
	if target == "" {
		return errors.New("tmux target is required")
	}
	_, err := s.runner().Run(ctx, "", "tmux", "kill-session", "-t", target)
	if err != nil {
		return fmt.Errorf("tmux kill: %w", err)
	}
	return nil
}

func (s TmuxService) runner() Runner {
	if s.Runner != nil {
		return s.Runner
	}
	return ExecRunner{}
}
