package collect

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/KirillSachkov/gvardia/internal/model"
)

// ChangedFiles lists a worktree's artifacts: the files changed vs its base
// branch (git name-status) plus any explicit report files under
// <worktree>/.gvardia/reports/*.md. Like ChangeStatFor, it is called only for
// session-bearing worktrees.
func ChangedFiles(ctx context.Context, runner Runner, path, base string) []model.Artifact {
	rangeArg := "HEAD"
	if base != "" && base != "HEAD" {
		rangeArg = base + "...HEAD"
	}
	var arts []model.Artifact
	if out, err := runner.Run(ctx, path, "diff", "--name-status", rangeArg); err == nil {
		arts = parseNameStatus(out)
	}
	return append(arts, reportArtifacts(path)...)
}

// parseNameStatus parses `git diff --name-status` output into artifacts. Renames
// and copies (R100/C75, tab-separated old→new) keep the destination path.
func parseNameStatus(data []byte) []model.Artifact {
	var arts []model.Artifact
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 2 || fields[0] == "" {
			continue
		}
		arts = append(arts, model.Artifact{
			Status: fields[0][:1], // "M", "A", "D", "R", "C"
			Path:   fields[len(fields)-1],
		})
	}
	return arts
}

// reportArtifacts lists <worktree>/.gvardia/reports/*.md as explicit artifacts.
func reportArtifacts(worktree string) []model.Artifact {
	dir := filepath.Join(worktree, ".gvardia", "reports")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var arts []model.Artifact
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		arts = append(arts, model.Artifact{
			Status: "report",
			Path:   filepath.Join(".gvardia", "reports", e.Name()),
		})
	}
	return arts
}
