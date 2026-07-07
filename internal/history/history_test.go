package history

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecent(t *testing.T) {
	claudeRoot := t.TempDir()
	codexRoot := t.TempDir()
	base := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	// claude session for /proj/a with an AI title (newest).
	writeClaude(t, claudeRoot, "/proj/a", "s1", base.Add(-1*time.Minute), []string{
		`{"type":"user","isMeta":true,"message":{"content":"<local-command-caveat>ignore me"}}`,
		`{"type":"user","isMeta":false,"message":{"content":"real first prompt"}}`,
		`{"type":"ai-title","aiTitle":"Fix the login flow","sessionId":"s1"}`,
	})
	// claude session for a DIFFERENT cwd — must be excluded.
	writeClaude(t, claudeRoot, "/proj/b", "s2", base, []string{
		`{"type":"user","isMeta":false,"message":{"content":"other project work"}}`,
	})
	// codex session for /proj/a (older), first real user prompt after injected junk.
	writeCodex(t, codexRoot, "2026/07/01/cx.jsonl", "/proj/a", "cx1", base.Add(-10*time.Minute), []string{
		`{"type":"response_item","payload":{"type":"message","role":"developer","content":[{"type":"input_text","text":"<INSTRUCTIONS>"}]}}`,
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"# AGENTS.md dump"}]}}`,
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"actually do X please"}]}}`,
	})

	r := Reader{ClaudeRoot: claudeRoot, CodexRoot: codexRoot}
	got := r.Recent(context.Background(), "/proj/a", Options{})

	if len(got) != 2 {
		t.Fatalf("got %d sessions, want 2 (claude+codex for /proj/a): %+v", len(got), got)
	}
	// Newest first: claude (−1m) before codex (−10m).
	if got[0].Harness != "claude" || got[1].Harness != "codex" {
		t.Errorf("order = %s,%s, want claude,codex", got[0].Harness, got[1].Harness)
	}
	if got[0].Summary != "Fix the login flow" {
		t.Errorf("claude summary = %q, want the AI title", got[0].Summary)
	}
	if got[0].Live {
		t.Error("history sessions must be Live=false")
	}
	if got[1].Summary != "actually do X please" {
		t.Errorf("codex summary = %q, want first clean user prompt", got[1].Summary)
	}
}

func TestRecentSince(t *testing.T) {
	claudeRoot := t.TempDir()
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	writeClaude(t, claudeRoot, "/p", "old", now.Add(-48*time.Hour), []string{
		`{"type":"user","isMeta":false,"message":{"content":"old work"}}`,
	})
	writeClaude(t, claudeRoot, "/p", "fresh", now.Add(-1*time.Hour), []string{
		`{"type":"user","isMeta":false,"message":{"content":"fresh work"}}`,
	})
	r := Reader{ClaudeRoot: claudeRoot, now: func() time.Time { return now }}
	got := r.Recent(context.Background(), "/p", Options{Since: 24 * time.Hour})
	if len(got) != 1 || got[0].SessionID != "fresh" {
		t.Fatalf("Since=24h should keep only the fresh session, got %+v", got)
	}
}

func TestClaudeFallsBackToFirstPrompt(t *testing.T) {
	root := t.TempDir()
	writeClaude(t, root, "/p", "s", time.Now(), []string{
		`{"type":"user","isMeta":true,"message":{"content":"<command-name>/foo"}}`,
		`{"type":"user","isMeta":false,"message":{"content":[{"type":"text","text":"do the thing"}]}}`,
	})
	if s := claudeSummary(filepath.Join(root, encodeCwd("/p"), "s.jsonl")); s != "do the thing" {
		t.Errorf("summary = %q, want first real prompt when no ai-title", s)
	}
}

func writeClaude(t *testing.T, root, cwd, id string, mtime time.Time, lines []string) {
	t.Helper()
	dir := filepath.Join(root, encodeCwd(cwd))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, id+".jsonl")
	writeLines(t, path, lines)
	chtime(t, path, mtime)
}

func writeCodex(t *testing.T, root, rel, cwd, id string, mtime time.Time, extra []string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	head := `{"type":"session_meta","payload":{"cwd":"` + cwd + `","session_id":"` + id + `","timestamp":"2026-07-01T10:00:00Z"}}`
	writeLines(t, path, append([]string{head}, extra...))
	chtime(t, path, mtime)
}

func writeLines(t *testing.T, path string, lines []string) {
	t.Helper()
	content := ""
	for _, l := range lines {
		content += l + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func chtime(t *testing.T, path string, mtime time.Time) {
	t.Helper()
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}
