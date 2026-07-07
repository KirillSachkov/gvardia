package adapters

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/KirillSachkov/gvardia/internal/config"
	"github.com/KirillSachkov/gvardia/internal/model"
)

func TestEnabledSkipsUnknown(t *testing.T) {
	cfg := config.Default()
	cfg.Adapters = []string{"claude", "bogus", "tmux"}
	ads := Enabled(cfg)
	if len(ads) != 2 {
		t.Fatalf("got %d adapters, want 2 (bogus skipped)", len(ads))
	}
	if ads[0].Name() != "claude" || ads[1].Name() != "tmux" {
		t.Errorf("adapter order = %s,%s", ads[0].Name(), ads[1].Name())
	}
}

type fakeAdapter struct {
	name     string
	sessions []model.Session
	err      error
}

func (f fakeAdapter) Name() string { return f.name }
func (f fakeAdapter) Sessions(context.Context) ([]model.Session, error) {
	return f.sessions, f.err
}

func TestCollectSessionsMergesAndCollectsFailures(t *testing.T) {
	ads := []Adapter{
		fakeAdapter{name: "claude", sessions: []model.Session{{Harness: "claude", Cwd: "/a"}}},
		fakeAdapter{name: "codex", sessions: []model.Session{{Harness: "codex", Cwd: "/b"}}},
		fakeAdapter{name: "tmux", err: errors.New("no server")},
	}
	sessions, failures := CollectSessions(context.Background(), ads)

	if len(sessions) != 2 {
		t.Fatalf("got %d sessions, want 2", len(sessions))
	}
	if len(failures) != 1 || failures[0].Adapter != "tmux" {
		t.Fatalf("failures = %+v, want one tmux failure", failures)
	}

	// Order is nondeterministic (concurrent) — sort before asserting.
	cwds := []string{sessions[0].Cwd, sessions[1].Cwd}
	sort.Strings(cwds)
	if cwds[0] != "/a" || cwds[1] != "/b" {
		t.Errorf("cwds = %v, want [/a /b]", cwds)
	}
}
