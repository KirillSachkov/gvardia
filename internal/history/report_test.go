package history

import (
	"context"
	"testing"
	"time"
)

func TestReportForClaude(t *testing.T) {
	root := t.TempDir()
	writeClaude(t, root, "/p", "s", time.Now(), []string{
		`{"type":"user","isMeta":false,"message":{"content":"do the thing"}}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"first pass"}]}}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"Done: fixed the bug and added a test."}]}}`,
	})
	r := Reader{ClaudeRoot: root}
	if got := r.ReportFor(context.Background(), "claude", "s", "/p"); got != "Done: fixed the bug and added a test." {
		t.Errorf("claude report = %q, want the last assistant message", got)
	}
}

func TestReportForCodex(t *testing.T) {
	root := t.TempDir()
	writeCodex(t, root, "2026/07/01/cx.jsonl", "/p", "cx1", time.Now(), []string{
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"go"}]}}`,
		`{"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"working on it"}]}}`,
		`{"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Finished: shipped the feature."}]}}`,
	})
	r := Reader{CodexRoot: root}
	if got := r.ReportFor(context.Background(), "codex", "cx1", "/p"); got != "Finished: shipped the feature." {
		t.Errorf("codex report = %q, want the last assistant message", got)
	}
}

func TestReportForClaudeMissing(t *testing.T) {
	r := Reader{ClaudeRoot: t.TempDir()}
	if got := r.ReportFor(context.Background(), "claude", "nope", "/p"); got != "" {
		t.Errorf("missing transcript report = %q, want empty", got)
	}
	if got := r.ReportFor(context.Background(), "claude", "", "/p"); got != "" {
		t.Errorf("empty id report = %q, want empty", got)
	}
}
