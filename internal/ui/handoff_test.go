package ui

import (
	"testing"

	"github.com/KirillSachkov/gvardia/internal/model"
)

func TestHandoffCommand(t *testing.T) {
	cases := []struct {
		name string
		s    model.Session
		want string
	}{
		{"claude", model.Session{Harness: "claude", SessionID: "abc", WorktreePath: "/w"},
			"cd '/w' && claude --resume abc"},
		{"codex", model.Session{Harness: "codex", SessionID: "xyz", WorktreePath: "/w"},
			"cd '/w' && codex resume xyz"},
		{"codex-last", model.Session{Harness: "codex", WorktreePath: "/w"},
			"cd '/w' && codex resume --last"},
		{"tmux-no-worktree", model.Session{Harness: "tmux", SessionID: "work"},
			"tmux attach -t work"},
		{"claude-no-id", model.Session{Harness: "claude", WorktreePath: "/w"}, ""},
		{"unknown", model.Session{Harness: "unknown"}, ""},
	}
	for _, tc := range cases {
		if got := handoffCommand(tc.s); got != tc.want {
			t.Errorf("%s: handoffCommand = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestShellQuoteEscapes(t *testing.T) {
	if got := shellQuote("/a b/c"); got != "'/a b/c'" {
		t.Errorf("shellQuote spaces = %q", got)
	}
	if got := shellQuote("it's"); got != `'it'\''s'` {
		t.Errorf("shellQuote embedded quote = %q", got)
	}
}

func TestHandoffKeyCopiesAndToasts(t *testing.T) {
	m := ready(t) // alpha's a1 is a claude session with an id + worktree
	m2, cmd := step(m, keyText("r"))
	if cmd == nil {
		t.Fatal("r should return a SetClipboard command")
	}
	if m2.toast == "" {
		t.Error("r should set a transient toast")
	}
	// Any subsequent key dismisses the toast.
	m3, _ := step(m2, keyText("j"))
	if m3.toast != "" {
		t.Error("a following key should clear the toast")
	}
}
