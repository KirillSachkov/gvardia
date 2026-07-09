package collect

import (
	"context"

	"github.com/KirillSachkov/gvardia/internal/runs"
)

// EnrichRuns adds git diff stats and artifact links to local runs.
func EnrichRuns(ctx context.Context, runner Runner, list []runs.Run, base string) []runs.Run {
	for i := range list {
		if list[i].WorktreePath == "" {
			continue
		}
		list[i].ChangeStat = ChangeStatFor(ctx, runner, list[i].WorktreePath, base)
		list[i].Artifacts = ChangedFiles(ctx, runner, list[i].WorktreePath, base)
	}
	return list
}
