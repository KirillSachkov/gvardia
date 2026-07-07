package adapters

import (
	"context"
	"strconv"
	"strings"

	"github.com/KirillSachkov/gvardia/internal/model"
)

// agentCommands are foreground process names that mark a tmux pane as an agent.
// Kept conservative: generic runtimes like "node"/"python" are excluded to avoid
// false positives.
var agentCommands = map[string]bool{
	"claude": true, "codex": true, "aider": true,
	"goose": true, "opencode": true, "gemini": true,
}

// Tmux is a fallback signal: it reports panes whose foreground command looks like
// an agent, so agents without a first-class adapter still surface. See
// docs/ADAPTERS.md.
type Tmux struct {
	run commandFunc // nil uses the real tmux CLI
}

// Name identifies the adapter and harness.
func (Tmux) Name() string { return "tmux" }

// Sessions lists tmux panes and keeps the ones running an agent command. A
// missing tmux server makes tmux exit non-zero, which the caller treats as
// "adapter skipped".
func (t Tmux) Sessions(ctx context.Context) ([]model.Session, error) {
	run := t.run
	if run == nil {
		run = execCommand
	}
	out, err := run(ctx, "tmux", "list-panes", "-a", "-F",
		"#{pane_current_path}\t#{pane_pid}\t#{session_name}\t#{pane_current_command}")
	if err != nil {
		return nil, err
	}
	return parseTmuxPanes(out), nil
}

func parseTmuxPanes(data []byte) []model.Session {
	var sessions []model.Session
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, "\t", 4)
		if len(fields) != 4 {
			continue
		}
		path, pidStr, session, command := fields[0], fields[1], fields[2], fields[3]
		if !agentCommands[command] {
			continue
		}
		pid, _ := strconv.Atoi(pidStr)
		sessions = append(sessions, model.Session{
			Harness:   "tmux",
			Name:      command,
			SessionID: session, // tmux session name = `tmux attach -t` target
			PID:       pid,
			Cwd:       path,
			Status:    model.StatusBusy, // a live pane running an agent command
		})
	}
	return sessions
}
