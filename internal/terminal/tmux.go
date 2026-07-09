// Package terminal manages external terminal backends. v1 uses tmux only.
package terminal

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
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
	Env         map[string]string
	Target      string
	WindowTitle string
}

// PaneState is the observable state of the first pane in a tmux target.
type PaneState struct {
	Alive    bool
	ExitCode int
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
	command := withShellEnv(spec.Command, spec.Env)
	_, err := s.runner().Run(ctx, "", "tmux",
		"new-session", "-d",
		"-s", target,
		"-c", spec.Worktree,
		"-n", title,
		"sh", "-lc", command,
	)
	if err != nil {
		return "", fmt.Errorf("tmux launch: %w", err)
	}
	if _, err := s.runner().Run(ctx, "", "tmux", "set-option", "-t", target, "remain-on-exit", "on"); err != nil {
		_, _ = s.runner().Run(ctx, "", "tmux", "kill-session", "-t", target)
		return "", fmt.Errorf("tmux keep exited pane: %w", err)
	}
	return target, nil
}

// Inspect reports whether the target pane is alive and its exit code when dead.
func (s TmuxService) Inspect(ctx context.Context, target string) (PaneState, error) {
	if target == "" {
		return PaneState{}, errors.New("tmux target is required")
	}
	out, err := s.runner().Run(ctx, "", "tmux", "display-message", "-p", "-t", target, "#{pane_dead}|#{pane_dead_status}")
	if err != nil {
		return PaneState{}, fmt.Errorf("tmux inspect: %w", err)
	}
	parts := strings.SplitN(strings.TrimSpace(string(out)), "|", 2)
	if len(parts) != 2 || (parts[0] != "0" && parts[0] != "1") {
		return PaneState{}, fmt.Errorf("tmux inspect: unexpected state %q", strings.TrimSpace(string(out)))
	}
	state := PaneState{Alive: parts[0] == "0"}
	if parts[1] != "" {
		code, err := strconv.Atoi(parts[1])
		if err != nil {
			return PaneState{}, fmt.Errorf("tmux inspect: invalid exit code %q", parts[1])
		}
		state.ExitCode = code
	}
	return state, nil
}

func withShellEnv(command string, env map[string]string) string {
	if len(env) == 0 {
		return command
	}
	keys := make([]string, 0, len(env))
	for key, value := range env {
		if key == "" || value == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return command
	}
	parts := make([]string, 0, len(keys)+1)
	for _, key := range keys {
		parts = append(parts, "export "+key+"="+shellQuote(env[key]))
	}
	parts = append(parts, command)
	return strings.Join(parts, "; ")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
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
