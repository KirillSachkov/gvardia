package adapters

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/KirillSachkov/gvardia/internal/model"
)

func TestCodexSessionsNewestPerCwd(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	// Two sessions for /a (keep the newer), one for /b. Nested dirs test recursion.
	writeCodexSession(t, root, "2026/06/30/a-old.jsonl", "/a", "a-old-1111", now.Add(-time.Hour))
	writeCodexSession(t, root, "2026/07/01/a-new.jsonl", "/a", "a-new-2222", now.Add(-5*time.Second))
	writeCodexSession(t, root, "2026/07/01/b.jsonl", "/b", "b-3333", now.Add(-time.Hour))
	// A non-session file that must be ignored.
	writeFileT(t, filepath.Join(root, "notes.txt"), "ignore me")

	c := Codex{
		Root:      root,
		Staleness: 15 * time.Second,
		now:       func() time.Time { return now },
	}
	sessions, err := c.Sessions(context.Background())
	if err != nil {
		t.Fatalf("Sessions: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("got %d sessions, want 2 (newest per cwd): %+v", len(sessions), sessions)
	}

	byCwd := map[string]model.Session{}
	for _, s := range sessions {
		byCwd[s.Cwd] = s
	}
	a, ok := byCwd["/a"]
	if !ok {
		t.Fatal("no session for /a")
	}
	if a.Name != "a-new-22" { // shortID keeps first 8 chars of "a-new-2222"
		t.Errorf("/a session name = %q, want newest a-new-22", a.Name)
	}
	if a.Status != model.StatusBusy {
		t.Errorf("/a status = %q, want busy (fresh mtime)", a.Status)
	}
	if byCwd["/b"].Status != model.StatusIdle {
		t.Errorf("/b status = %q, want idle (stale mtime)", byCwd["/b"].Status)
	}
	if a.Harness != "codex" {
		t.Errorf("Harness = %q, want codex", a.Harness)
	}
}

func TestCodexMissingRootIsNotAnError(t *testing.T) {
	c := Codex{Root: filepath.Join(t.TempDir(), "nope")}
	sessions, err := c.Sessions(context.Background())
	if err != nil {
		t.Fatalf("Sessions on missing root: %v, want nil", err)
	}
	if sessions != nil {
		t.Errorf("sessions = %+v, want nil", sessions)
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
