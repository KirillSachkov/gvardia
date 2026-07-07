// Command wt-prune lists git worktrees across your roots and removes the merged
// (and, with --stale, stale) ones. It never touches a primary or dirty worktree.
// Dry-run by default; pass --yes to actually remove.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/KirillSachkov/gvardia/internal/collect"
	"github.com/KirillSachkov/gvardia/internal/config"
	"github.com/KirillSachkov/gvardia/internal/prune"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "wt-prune:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("wt-prune", flag.ContinueOnError)
	apply := fs.Bool("yes", false, "actually remove worktrees (default: dry-run)")
	includeStale := fs.Bool("stale", false, "also remove stale worktrees")
	days := fs.Int("days", 30, "staleness threshold in days")
	configPath := fs.String("config", config.DefaultPath(), "path to config file")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	if roots := fs.Args(); len(roots) > 0 {
		expanded := make([]string, len(roots))
		for i, r := range roots {
			expanded[i] = config.ExpandPath(r)
		}
		cfg.Roots = expanded
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	plan, err := prune.Plan(ctx, collect.Git{}, cfg, prune.Options{StaleDays: *days})
	if err != nil {
		return err
	}

	removed, kept := 0, 0
	for _, it := range plan {
		label := it.Project + "/" + filepath.Base(it.Path)
		if !it.Removable(*includeStale) {
			kept++
			fmt.Printf("keep         %s — %s\n", label, keepReason(it))
			continue
		}
		if !*apply {
			fmt.Printf("would remove %s — %s\n", label, string(it.Class))
			continue
		}
		if err := prune.Remove(ctx, collect.Git{}, it); err != nil {
			fmt.Printf("FAILED       %s — %v\n", label, err)
			continue
		}
		removed++
		fmt.Printf("removed      %s — %s\n", label, string(it.Class))
	}

	fmt.Println("---")
	if *apply {
		fmt.Printf("removed: %d · kept: %d\n", removed, kept)
	} else {
		hint := ""
		if !*includeStale {
			hint = "; add --stale for stale worktrees"
		}
		fmt.Printf("dry-run — pass --yes to remove%s · kept: %d\n", hint, kept)
	}
	return nil
}

// keepReason describes why a worktree is being kept.
func keepReason(it prune.Item) string {
	switch it.Class {
	case prune.Stale:
		return fmt.Sprintf("stale %dd (use --stale)", it.AgeDays)
	case prune.Active:
		return fmt.Sprintf("active (%dd)", it.AgeDays)
	default:
		return string(it.Class)
	}
}
