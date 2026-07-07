package collect

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Runner runs a git subcommand in dir and returns its stdout. It is declared
// here, where the collectors consume it, so tests can fake git without spawning
// real processes (accept interfaces, return structs).
type Runner interface {
	Run(ctx context.Context, dir string, args ...string) ([]byte, error)
}

// Git is the production [Runner]: it shells out to the real git binary.
type Git struct{}

// Run executes `git <args>` with the working directory set to dir.
func (Git) Run(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s in %s: %w: %s",
			strings.Join(args, " "), dir, err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}
