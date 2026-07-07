// Package prune classifies git worktrees across the configured roots as merged,
// stale, or active, and removes the disposable ones — never a primary or dirty
// worktree. It backs both the `wt-prune` CLI and gvardia's `g` action.
package prune

import (
	"context"
	"strings"
	"time"

	"github.com/KirillSachkov/gvardia/internal/collect"
	"github.com/KirillSachkov/gvardia/internal/config"
	"github.com/KirillSachkov/gvardia/internal/model"
)

// Class is a worktree's disposition. Only Merged and Stale are ever removed.
type Class string

const (
	Primary Class = "primary" // the main checkout — never removed
	Dirty   Class = "dirty"   // uncommitted changes — never removed
	Merged  Class = "merged"  // branch merged into base — safe to remove
	Stale   Class = "stale"   // no commits within the stale window
	Active  Class = "active"  // unmerged, recent — kept
)

// Item is one classified worktree.
type Item struct {
	Project string
	Primary string // the repo's primary worktree path (where `worktree remove` runs)
	Path    string
	Branch  string
	Base    string
	Class   Class
	AgeDays int
}

// Options tune classification.
type Options struct {
	StaleDays int // a clean worktree older than this many days is Stale (0 disables)
}

// Plan classifies every worktree under cfg.Roots. It reuses the collector for
// discovery + status, then adds per-repo merged detection.
func Plan(ctx context.Context, runner collect.Runner, cfg config.Config, opts Options) ([]Item, error) {
	projects, err := collect.Collect(ctx, runner, cfg)
	if err != nil {
		return nil, err
	}
	now := time.Now()

	var items []Item
	for _, p := range projects {
		base := ""
		if len(p.Worktrees) > 0 {
			base = p.Worktrees[0].BaseBranch
		}
		merged := mergedSet(ctx, runner, p.Path, base)
		for _, w := range p.Worktrees {
			items = append(items, classify(p.Name, p.Path, base, w, merged, now, opts.StaleDays))
		}
	}
	return items, nil
}

// classify assigns a Class using precedence primary > dirty > merged > stale.
func classify(project, primary, base string, w model.Worktree, merged map[string]bool, now time.Time, staleDays int) Item {
	it := Item{Project: project, Primary: primary, Path: w.Path, Branch: w.Branch, Base: base}
	if !w.LastCommit.IsZero() {
		it.AgeDays = int(now.Sub(w.LastCommit).Hours() / 24)
	}
	switch {
	case w.IsPrimary:
		it.Class = Primary
	case w.Dirty:
		it.Class = Dirty
	case w.Branch != "" && w.Branch != base && merged[w.Branch]:
		it.Class = Merged
	case staleDays > 0 && !w.LastCommit.IsZero() && it.AgeDays >= staleDays:
		it.Class = Stale
	default:
		it.Class = Active
	}
	return it
}

// Removable reports whether an item may be removed given whether stale worktrees
// are included. Primary and Dirty are never removable.
func (i Item) Removable(includeStale bool) bool {
	switch i.Class {
	case Merged:
		return true
	case Stale:
		return includeStale
	default:
		return false
	}
}

// Remove deletes a worktree via `git worktree remove` (which refuses a dirty
// worktree without --force, and we never force).
func Remove(ctx context.Context, runner collect.Runner, item Item) error {
	_, err := runner.Run(ctx, item.Primary, "worktree", "remove", item.Path)
	return err
}

// mergedSet returns the branches merged into base for a repo.
func mergedSet(ctx context.Context, runner collect.Runner, primary, base string) map[string]bool {
	set := make(map[string]bool)
	if base == "" {
		return set
	}
	out, err := runner.Run(ctx, primary, "branch", "--merged", base, "--format=%(refname:short)")
	if err != nil {
		return set
	}
	for _, line := range strings.Split(string(out), "\n") {
		if b := strings.TrimSpace(line); b != "" {
			set[b] = true
		}
	}
	return set
}
