// Package collect discovers git repositories under the configured roots and
// enriches their worktrees with status (dirty, ahead/behind, last commit). All
// git access goes through a [Runner] so the parsers can be tested against
// captured fixtures. Collection fans out concurrently with a bounded semaphore
// and tolerates per-repo failures: a partial fleet beats no fleet.
package collect

import (
	"context"
	"path/filepath"
	"runtime"
	"sort"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/KirillSachkov/gvardia/internal/config"
	"github.com/KirillSachkov/gvardia/internal/model"
)

// Collect discovers every project under cfg.Roots and returns them with fully
// enriched worktrees, sorted by name. Individual repo errors are skipped.
func Collect(ctx context.Context, runner Runner, cfg config.Config) ([]model.Project, error) {
	return collectFrom(ctx, runner, cfg, DiscoverRepos(cfg.Roots))
}

// CollectTracked enriches an explicit list of project roots, skipping discovery.
// Each path is treated as a repo root; its linked worktrees are found via git.
// Paths that are not git repos are silently skipped (a partial fleet beats none).
func CollectTracked(ctx context.Context, runner Runner, cfg config.Config, paths []string) ([]model.Project, error) {
	return collectFrom(ctx, runner, cfg, paths)
}

// collectFrom is the shared enrichment pipeline used by both Collect (roots scan)
// and CollectTracked (explicit list): list worktrees per candidate, collapse to
// one project per primary, then enrich every worktree concurrently.
func collectFrom(ctx context.Context, runner Runner, cfg config.Config, candidates []string) ([]model.Project, error) {
	sem := make(chan struct{}, concurrency())

	// Phase 1: list worktrees per candidate, collapsing to one entry per primary.
	var mu sync.Mutex
	byPrimary := make(map[string]*model.Project)

	g, gctx := errgroup.WithContext(ctx)
	for _, repo := range candidates {
		repo := repo
		g.Go(func() error {
			sem <- struct{}{}
			defer func() { <-sem }()

			out, err := runner.Run(gctx, repo, "worktree", "list", "--porcelain")
			if err != nil {
				return nil // skip this candidate, keep the batch alive
			}
			wts := parseWorktrees(out)
			if len(wts) == 0 {
				return nil
			}
			primary := wts[0].Path
			mu.Lock()
			if _, ok := byPrimary[primary]; !ok {
				byPrimary[primary] = &model.Project{
					Name:      filepath.Base(primary),
					Path:      primary,
					Worktrees: wts,
				}
			}
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Phase 2: enrich every worktree of every project.
	eg, ectx := errgroup.WithContext(ctx)
	for _, p := range byPrimary {
		p := p
		base := resolveBaseBranch(ectx, runner, p.Path, cfg.BaseBranch(p.Name))
		for i := range p.Worktrees {
			i := i
			eg.Go(func() error {
				sem <- struct{}{}
				defer func() { <-sem }()
				p.Worktrees[i] = Enrich(ectx, runner, p.Worktrees[i], base)
				return nil
			})
		}
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	projects := make([]model.Project, 0, len(byPrimary))
	for _, p := range byPrimary {
		projects = append(projects, *p)
	}
	sort.Slice(projects, func(i, j int) bool { return projects[i].Name < projects[j].Name })
	return projects, nil
}

// concurrency bounds concurrent git shell-outs at min(16, NumCPU*2).
func concurrency() int {
	n := runtime.NumCPU() * 2
	if n > 16 {
		n = 16
	}
	if n < 1 {
		n = 1
	}
	return n
}
