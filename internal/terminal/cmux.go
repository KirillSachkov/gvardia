package terminal

import (
	"context"
	"errors"
	"fmt"
	"strconv"
)

// CmuxService opens a presentation workspace for an existing persistent
// session. tmux remains the session owner.
type CmuxService struct {
	Runner Runner
}

// OpenSpec describes a new cmux workspace.
type OpenSpec struct {
	Name    string
	Cwd     string
	Command string
	Focus   bool
}

// Open creates a cmux workspace that runs Command.
func (s CmuxService) Open(ctx context.Context, spec OpenSpec) error {
	if spec.Command == "" {
		return errors.New("workspace command is required")
	}
	name := spec.Name
	if name == "" {
		name = "Gvardia agent"
	}
	args := []string{"new-workspace", "--name", name}
	if spec.Cwd != "" {
		args = append(args, "--cwd", spec.Cwd)
	}
	args = append(args, "--command", spec.Command, "--focus", strconv.FormatBool(spec.Focus))
	if _, err := s.runner().Run(ctx, "", "cmux", args...); err != nil {
		return fmt.Errorf("cmux workspace: %w", err)
	}
	return nil
}

// AttachCommand returns a pasteable command for an existing tmux target.
func AttachCommand(target string) string {
	if target == "" {
		return ""
	}
	return "tmux attach -t " + shellQuote(target)
}

func (s CmuxService) runner() Runner {
	if s.Runner != nil {
		return s.Runner
	}
	return ExecRunner{}
}
