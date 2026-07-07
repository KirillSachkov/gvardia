package collect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseNameStatus(t *testing.T) {
	arts := parseNameStatus([]byte("M\tinternal/ui/view.go\nA\tinternal/ui/keys.go\nR100\told.go\tnew.go\n\n"))
	if len(arts) != 3 {
		t.Fatalf("got %d artifacts, want 3: %+v", len(arts), arts)
	}
	if arts[0].Status != "M" || arts[0].Path != "internal/ui/view.go" {
		t.Errorf("artifact[0] = %+v", arts[0])
	}
	if arts[1].Status != "A" || arts[1].Path != "internal/ui/keys.go" {
		t.Errorf("artifact[1] = %+v", arts[1])
	}
	if arts[2].Status != "R" || arts[2].Path != "new.go" { // rename keeps the destination
		t.Errorf("artifact[2] = %+v", arts[2])
	}
}

func TestReportArtifactsListsMarkdown(t *testing.T) {
	wt := t.TempDir()
	dir := filepath.Join(wt, ".gvardia", "reports")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "summary.md"), []byte("done"), 0o600); err != nil {
		t.Fatal(err)
	}
	arts := reportArtifacts(wt)
	if len(arts) != 1 || arts[0].Status != "report" {
		t.Fatalf("reportArtifacts = %+v, want one report", arts)
	}
	if arts[0].Path != filepath.Join(".gvardia", "reports", "summary.md") {
		t.Errorf("report path = %q", arts[0].Path)
	}
}
