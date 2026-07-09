package collect

import (
	"context"
	"strings"
	"testing"

	"github.com/KirillSachkov/gvardia/internal/runs"
)

func TestEnrichRunsAddsChangeStatAndArtifacts(t *testing.T) {
	runner := runMap{
		"diff --numstat main...HEAD":     []byte("2\t1\tinternal/ui/view.go\n"),
		"diff --name-status main...HEAD": []byte("M\tinternal/ui/view.go\nA\tinternal/runs/store.go\n"),
	}
	got := EnrichRuns(context.Background(), runner, []runs.Run{{
		ID: "run-1", WorktreePath: "/repo/wt", Branch: "feature", Status: runs.StatusRunning,
	}}, "main")

	if len(got) != 1 {
		t.Fatalf("EnrichRuns returned %d runs, want 1", len(got))
	}
	if got[0].ChangeStat.Files != 1 || got[0].ChangeStat.Added != 2 || got[0].ChangeStat.Removed != 1 {
		t.Errorf("ChangeStat = %+v, want 1 file +2 -1", got[0].ChangeStat)
	}
	if len(got[0].Artifacts) != 2 || got[0].Artifacts[0].Path != "internal/ui/view.go" {
		t.Errorf("Artifacts = %+v, want changed files", got[0].Artifacts)
	}
}

type runMap map[string][]byte

func (r runMap) Run(_ context.Context, _ string, args ...string) ([]byte, error) {
	return r[strings.Join(args, " ")], nil
}
