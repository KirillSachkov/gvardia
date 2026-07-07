package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/KirillSachkov/gvardia/internal/adapters"
	"github.com/KirillSachkov/gvardia/internal/collect"
	"github.com/KirillSachkov/gvardia/internal/config"
	"github.com/KirillSachkov/gvardia/internal/model"
)

// runAgents implements `gvardia agents [--json]`: the headless fleet dump. It
// collects projects, runs the enabled adapters, joins sessions to worktrees, and
// prints the result. Adapters that fail (absent CLI, parse error) are reported to
// stderr as a banner, never fatal.
func runAgents(args []string, configPath string) error {
	fs := flag.NewFlagSet("gvardia agents", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "emit the fleet as JSON")
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
	sessions, failures := adapters.CollectSessions(ctx, adapters.Enabled(cfg))
	projects = collect.Join(projects, sessions)

	for _, f := range failures {
		fmt.Fprintf(os.Stderr, "gvardia: adapter %s skipped: %v\n", f.Adapter, f.Err)
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(projects)
	}

	for _, p := range projects {
		if p.LiveAgents == 0 {
			continue
		}
		fmt.Printf("%s — %d agent(s)\n", p.Name, p.LiveAgents)
		for _, w := range p.Worktrees {
			for _, s := range w.Sessions {
				fmt.Printf("  %s\n", formatSession(s, w))
			}
		}
	}
	return nil
}

// formatSession renders a one-line human summary of an agent session.
func formatSession(s model.Session, w model.Worktree) string {
	branch := w.Branch
	if branch == "" {
		branch = "(detached)"
	}
	return fmt.Sprintf("%-6s %-8s %-30s %s", s.Status, s.Harness, s.Name, branch)
}
