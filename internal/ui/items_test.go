package ui

import (
	"testing"

	"github.com/KirillSachkov/gvardia/internal/adapters"
	"github.com/KirillSachkov/gvardia/internal/model"
)

func TestProjectItemMatches(t *testing.T) {
	p := projectItem{project: model.Project{
		Name:      "education-platform",
		Worktrees: []model.Worktree{{Branch: "feat/675-s2"}},
	}}
	for _, tc := range []struct {
		q    string
		want bool
	}{
		{"edu", true},      // name
		{"PLATFORM", true}, // case-insensitive
		{"feat", true},     // branch
		{"zzz", false},
	} {
		if got := p.matches(tc.q); got != tc.want {
			t.Errorf("matches(%q) = %v, want %v", tc.q, got, tc.want)
		}
	}
}

func TestGlyph(t *testing.T) {
	cases := []struct {
		name string
		wt   model.Worktree
		want string
	}{
		{"no agent", model.Worktree{}, " "},
		{"busy claude", wtWith("claude", model.StatusBusy), "●"},
		{"idle claude", wtWith("claude", model.StatusIdle), "○"},
		{"codex", wtWith("codex", model.StatusIdle), "◍"},
		{"failed", wtWith("claude", model.StatusFailed), "✖"},
	}
	for _, c := range cases {
		if got := glyph(c.wt); got != c.want {
			t.Errorf("%s: glyph = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestWorktreeRow(t *testing.T) {
	row := worktreeRow(model.Worktree{Branch: "main", Dirty: true, Ahead: 2, Sessions: []model.Session{
		{Harness: "claude", Name: "agent-1", Status: model.StatusBusy},
	}})
	if len(row) != 4 {
		t.Fatalf("row has %d cells, want 4", len(row))
	}
	if row[0] != "●" || row[1] != "claude" || row[2] != "agent-1" {
		t.Errorf("row = %v", row)
	}
	if want := "main ✱ ↑2↓0"; row[3] != want {
		t.Errorf("branch cell = %q, want %q", row[3], want)
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 3); got != "he…" {
		t.Errorf("truncate(hello,3) = %q, want he…", got)
	}
	if got := truncate("hi", 10); got != "hi" {
		t.Errorf("truncate(hi,10) = %q, want hi", got)
	}
}

func TestFailureBanner(t *testing.T) {
	if got := failureBanner(nil); got != "" {
		t.Errorf("empty banner = %q", got)
	}
	got := failureBanner([]adapters.Failure{{Adapter: "tmux"}, {Adapter: "codex"}})
	if got != "adapters skipped: tmux, codex" {
		t.Errorf("banner = %q", got)
	}
}

func wtWith(harness string, status model.Status) model.Worktree {
	return model.Worktree{Sessions: []model.Session{{Harness: harness, Status: status}}}
}
