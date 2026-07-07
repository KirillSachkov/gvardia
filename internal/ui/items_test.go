package ui

import (
	"testing"
	"time"

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
		{"edu", true},
		{"PLATFORM", true},
		{"feat", true},
		{"zzz", false},
	} {
		if got := p.matches(tc.q); got != tc.want {
			t.Errorf("matches(%q) = %v, want %v", tc.q, got, tc.want)
		}
	}
}

func TestSessionGlyph(t *testing.T) {
	cases := []struct {
		name string
		s    model.Session
		want string
	}{
		{"ended", model.Session{Live: false}, "✓"},
		{"busy claude", model.Session{Live: true, Harness: "claude", Status: model.StatusBusy}, "●"},
		{"idle claude", model.Session{Live: true, Harness: "claude", Status: model.StatusIdle}, "○"},
		{"codex", model.Session{Live: true, Harness: "codex", Status: model.StatusIdle}, "◍"},
		{"failed", model.Session{Live: true, Harness: "claude", Status: model.StatusFailed}, "✖"},
	}
	for _, c := range cases {
		if got := sessionGlyph(c.s); got != c.want {
			t.Errorf("%s: glyph = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestSessionRow(t *testing.T) {
	row := sessionRow(model.Session{
		Live: true, Harness: "claude", Name: "a1", Status: model.StatusBusy,
		Task: "#675", Branch: "feat/675-s3",
		ChangeStat: model.ChangeStat{Files: 3, Added: 90, Removed: 12},
	})
	if len(row) != 7 {
		t.Fatalf("row has %d cells, want 7", len(row))
	}
	if row[0] != "●" || row[1] != "claude" || row[2] != "a1" || row[3] != "#675" ||
		row[4] != "feat/675-s3" || row[5] != "+90/-12" {
		t.Errorf("row = %v", row)
	}
}

func TestRelativeAge(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "now"},
		{5 * time.Minute, "5m"},
		{3 * time.Hour, "3h"},
		{50 * time.Hour, "2d"},
	}
	for _, c := range cases {
		if got := relativeAge(c.d); got != c.want {
			t.Errorf("relativeAge(%v) = %q, want %q", c.d, got, c.want)
		}
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
