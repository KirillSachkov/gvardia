package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/KirillSachkov/gvardia/internal/collect"
	"github.com/KirillSachkov/gvardia/internal/config"
	"github.com/KirillSachkov/gvardia/internal/model"
)

// runProjects implements `gvardia projects [--json]`: it collects every project
// under the configured roots and dumps the model (JSON or a human summary).
func runProjects(args []string, configPath string) error {
	fs := flag.NewFlagSet("gvardia projects", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "emit the collected projects as JSON")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	projects, err := collect.Collect(ctx, collect.Git{}, cfg)
	if err != nil {
		return err
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(projects)
	}

	for _, p := range projects {
		fmt.Printf("%s — %d worktree(s)\n", p.Name, len(p.Worktrees))
		for _, w := range p.Worktrees {
			fmt.Printf("  %s\n", formatWorktree(w))
		}
	}
	return nil
}

// formatWorktree renders a one-line human summary of a worktree.
func formatWorktree(w model.Worktree) string {
	branch := w.Branch
	if branch == "" {
		branch = "(detached)"
	}
	flags := ""
	if w.IsPrimary {
		flags += " primary"
	}
	if w.Dirty {
		flags += " dirty"
	}
	if w.Ahead > 0 || w.Behind > 0 {
		flags += fmt.Sprintf(" +%d/-%d", w.Ahead, w.Behind)
	}
	return fmt.Sprintf("%-40s %s [base %s]%s", branch, w.Path, w.BaseBranch, flags)
}
