package prompts

import (
	"strings"
	"testing"

	"github.com/KirillSachkov/gvardia/internal/model"
)

func TestRenderIncludesTaskProjectAndReportPath(t *testing.T) {
	got := Render(Context{
		Task: model.Task{
			Title: "Build agent ops console",
			Body:  "Add local runs and tmux launch.",
		},
		ProjectName: "gvardia",
		ProjectPath: "/repo/gvardia",
		ReportPath:  ".gvardia/runs/run-123/report.md",
	})

	for _, want := range []string{
		"Task: Build agent ops console",
		"Add local runs and tmux launch.",
		"Project: gvardia",
		"/repo/gvardia",
		".gvardia/runs/run-123/report.md",
		"inspect before editing",
		"write final report",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Render missing %q:\n%s", want, got)
		}
	}
}

func TestRenderUsesFallbackBody(t *testing.T) {
	got := Render(Context{Task: model.Task{Title: "Empty body task"}})
	if !strings.Contains(got, "No task body was provided.") {
		t.Errorf("Render without body =\n%s\nwant fallback body", got)
	}
}
