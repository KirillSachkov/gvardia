package adapters

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/KirillSachkov/gvardia/internal/model"
)

func TestParseClaudeSessions(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "claude_agents.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	sessions, err := parseClaudeSessions(data)
	if err != nil {
		t.Fatalf("parseClaudeSessions: %v", err)
	}
	if len(sessions) != 3 {
		t.Fatalf("got %d sessions, want 3", len(sessions))
	}

	// First entry has no name → falls back to short session id.
	if sessions[0].Name != "cc09aa4e" {
		t.Errorf("sessions[0].Name = %q, want short id cc09aa4e", sessions[0].Name)
	}
	if sessions[0].Status != model.StatusIdle {
		t.Errorf("sessions[0].Status = %q, want idle", sessions[0].Status)
	}
	if sessions[0].Cwd != "/Users/dev/code/education-platform" {
		t.Errorf("sessions[0].Cwd = %q", sessions[0].Cwd)
	}
	if sessions[0].Harness != "claude" {
		t.Errorf("Harness = %q, want claude", sessions[0].Harness)
	}
	if got := sessions[0].StartedAt; got != time.UnixMilli(1783148015569) {
		t.Errorf("StartedAt = %v, want from 1783148015569ms", got)
	}

	// Named + busy entry.
	if sessions[1].Name != "software-engineer-tutorial-91" || sessions[1].Status != model.StatusBusy {
		t.Errorf("sessions[1] = %+v, want named busy", sessions[1])
	}
	if sessions[1].PID != 73112 {
		t.Errorf("sessions[1].PID = %d, want 73112", sessions[1].PID)
	}
}

func TestClaudeSessionsUsesInjectedRunner(t *testing.T) {
	c := Claude{run: func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != "claude" {
			t.Errorf("exec %q, want claude", name)
		}
		return []byte(`[{"pid":1,"cwd":"/x","sessionId":"deadbeef-0000","status":"busy"}]`), nil
	}}
	sessions, err := c.Sessions(context.Background())
	if err != nil {
		t.Fatalf("Sessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].Cwd != "/x" {
		t.Fatalf("got %+v", sessions)
	}
}
