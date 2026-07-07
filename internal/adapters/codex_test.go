package adapters

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/KirillSachkov/gvardia/internal/model"
)

type fakeLister struct {
	cwds map[string]int
	err  error
}

func (f fakeLister) LiveCwds(context.Context) (map[string]int, error) { return f.cwds, f.err }

func TestCodexSessionsAreProcessBacked(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	// Files for /a (newer wins) and /b — but only /a has a running process.
	writeCodexSession(t, root, "2026/06/30/a-old.jsonl", "/a", "a-old-1111", now.Add(-time.Hour))
	writeCodexSession(t, root, "2026/07/01/a-new.jsonl", "/a", "a-new-2222", now.Add(-5*time.Second))
	writeCodexSession(t, root, "2026/07/01/b.jsonl", "/b", "b-3333", now.Add(-time.Hour))

	c := Codex{
		Root:      root,
		Staleness: 15 * time.Second,
		now:       func() time.Time { return now },
		lister:    fakeLister{cwds: map[string]int{"/a": 111}}, // only /a is running
	}
	sessions, err := c.Sessions(context.Background())
	if err != nil {
		t.Fatalf("Sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1 (only process-backed /a): %+v", len(sessions), sessions)
	}
	s := sessions[0]
	if s.Cwd != "/a" || !s.Live || s.PID != 111 {
		t.Errorf("session = %+v, want /a live pid 111", s)
	}
	if s.SessionID != "a-new-2222" { // newest file for /a
		t.Errorf("SessionID = %q, want a-new-2222", s.SessionID)
	}
	if s.Status != model.StatusBusy { // fresh mtime
		t.Errorf("Status = %q, want busy", s.Status)
	}
}

func TestCodexNoLiveProcessNoSessions(t *testing.T) {
	c := Codex{lister: fakeLister{cwds: map[string]int{}}}
	sessions, err := c.Sessions(context.Background())
	if err != nil || sessions != nil {
		t.Fatalf("Sessions = (%+v, %v), want (nil, nil)", sessions, err)
	}
}

func TestCodexListerErrorPropagates(t *testing.T) {
	c := Codex{lister: fakeLister{err: errors.New("pgrep boom")}}
	if _, err := c.Sessions(context.Background()); err == nil {
		t.Fatal("Sessions error = nil, want the lister error to surface (adapter skipped)")
	}
}

func writeCodexSession(t *testing.T, root, rel, cwd, sessionID string, mtime time.Time) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	line := `{"timestamp":"2026-06-30T16:02:17.528Z","type":"session_meta","payload":{"session_id":"` +
		sessionID + `","cwd":"` + cwd + `","timestamp":"2026-06-30T16:01:50.520Z"}}` + "\n"
	writeFileT(t, path, line)
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
}

func writeFileT(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
