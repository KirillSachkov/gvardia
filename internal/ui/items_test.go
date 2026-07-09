package ui

import (
	"testing"
	"time"

	"charm.land/bubbles/v2/table"

	"github.com/KirillSachkov/gvardia/internal/adapters"
	"github.com/KirillSachkov/gvardia/internal/model"
	"github.com/KirillSachkov/gvardia/internal/runs"
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

func TestRunRowIncludesProject(t *testing.T) {
	row := runRow(runs.Run{Project: "gvardia", Runner: "codex", TaskTitle: "Reliable launch", Branch: "gvardia/run-1"})
	if len(row) != 7 {
		t.Fatalf("run row cells = %d, want 7", len(row))
	}
	if row[1] != "gvardia" || row[2] != "codex" {
		t.Fatalf("run row = %v, want project then runner", row)
	}
}

func TestTableColumnsFitCellPadding(t *testing.T) {
	for _, tc := range []struct {
		name    string
		columns []table.Column
		width   int
	}{
		{"sessions", sessionColumns(80), 80},
		{"runs", runColumns(80), 80},
		{"tasks", taskColumns(80), 80},
		{"tools", toolColumns(80), 80},
		{"worktrees", worktreeColumns(80), 80},
	} {
		t.Run(tc.name, func(t *testing.T) {
			total := 2 * len(tc.columns)
			for _, column := range tc.columns {
				total += column.Width
			}
			if total > tc.width {
				t.Fatalf("columns consume %d cells including padding, width is %d: %+v", total, tc.width, tc.columns)
			}
			for _, column := range tc.columns {
				if column.Title == "Δ" {
					t.Fatal("ambiguous delta column title must be replaced")
				}
			}
		})
	}
}
