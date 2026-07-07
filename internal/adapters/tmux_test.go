package adapters

import (
	"context"
	"errors"
	"testing"

	"github.com/KirillSachkov/gvardia/internal/model"
)

func TestParseTmuxPanes(t *testing.T) {
	// path \t pid \t session_name \t command
	data := []byte(
		"/Users/dev/code/proj-a\t111\twork\tclaude\n" +
			"/Users/dev/code/proj-b\t222\tmisc\tzsh\n" +
			"/Users/dev/code/proj-c\t333\tagents\tcodex\n" +
			"/Users/dev/code/proj-d\t444\tedit\tvim\n")

	sessions := parseTmuxPanes(data)
	if len(sessions) != 2 {
		t.Fatalf("got %d sessions, want 2 (claude+codex only): %+v", len(sessions), sessions)
	}
	if sessions[0].Cwd != "/Users/dev/code/proj-a" || sessions[0].Name != "claude" {
		t.Errorf("sessions[0] = %+v", sessions[0])
	}
	if sessions[0].SessionID != "work" {
		t.Errorf("sessions[0].SessionID = %q, want tmux session name 'work'", sessions[0].SessionID)
	}
	if sessions[0].PID != 111 || sessions[0].Status != model.StatusBusy {
		t.Errorf("sessions[0] pid/status = %d/%q", sessions[0].PID, sessions[0].Status)
	}
	if sessions[1].Name != "codex" || sessions[1].Harness != "tmux" {
		t.Errorf("sessions[1] = %+v", sessions[1])
	}
}

func TestTmuxNoServerIsSkipped(t *testing.T) {
	// A missing tmux server exits non-zero; the adapter must propagate the error
	// so the caller skips it (never crashes).
	tm := Tmux{run: func(context.Context, string, ...string) ([]byte, error) {
		return nil, errors.New("no server running")
	}}
	if _, err := tm.Sessions(context.Background()); err == nil {
		t.Fatal("Sessions error = nil, want error to trigger skip")
	}
}
