package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/KirillSachkov/gvardia/internal/model"
)

// Claude reports sessions from `claude agents --all --json`.
type Claude struct {
	// run overrides command execution in tests; nil uses the real claude CLI.
	run commandFunc
}

// Name identifies the adapter and harness.
func (Claude) Name() string { return "claude" }

// Sessions execs the claude CLI and maps its JSON to model sessions.
func (c Claude) Sessions(ctx context.Context) ([]model.Session, error) {
	run := c.run
	if run == nil {
		run = execCommand
	}
	out, err := run(ctx, "claude", "agents", "--all", "--json")
	if err != nil {
		return nil, err
	}
	return parseClaudeSessions(out)
}

type claudeAgent struct {
	PID       int    `json:"pid"`
	Cwd       string `json:"cwd"`
	Name      string `json:"name"`
	SessionID string `json:"sessionId"`
	StartedAt int64  `json:"startedAt"` // milliseconds since epoch
	Status    string `json:"status"`
}

func parseClaudeSessions(data []byte) ([]model.Session, error) {
	var agents []claudeAgent
	if err := json.Unmarshal(data, &agents); err != nil {
		return nil, fmt.Errorf("parse claude agents json: %w", err)
	}
	sessions := make([]model.Session, 0, len(agents))
	for _, a := range agents {
		name := a.Name
		if name == "" {
			name = shortID(a.SessionID)
		}
		sessions = append(sessions, model.Session{
			Harness:   "claude",
			Name:      name,
			SessionID: a.SessionID,
			PID:       a.PID,
			Cwd:       a.Cwd,
			Status:    claudeStatus(a.Status),
			StartedAt: time.UnixMilli(a.StartedAt),
		})
	}
	return sessions, nil
}

func claudeStatus(s string) model.Status {
	switch s {
	case "busy":
		return model.StatusBusy
	case "idle":
		return model.StatusIdle
	case "failed", "error":
		return model.StatusFailed
	default:
		return model.StatusUnknown
	}
}
