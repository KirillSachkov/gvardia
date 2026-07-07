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
	"github.com/KirillSachkov/gvardia/internal/history"
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
	projects = collect.AssembleLive(ctx, collect.Git{}, projects, sessions)
	attachSummaries(ctx, history.New(), projects)

	for _, f := range failures {
		fmt.Fprintf(os.Stderr, "gvardia: adapter %s skipped: %v\n", f.Adapter, f.Err)
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(projects)
	}

	for _, p := range projects {
		if len(p.WorkSessions) == 0 {
			continue
		}
		fmt.Printf("%s — %d agent(s)\n", p.Name, len(p.WorkSessions))
		for _, s := range p.WorkSessions {
			fmt.Printf("  %s\n", formatSession(s))
		}
	}
	return nil
}

// attachSummaries fills each live work-session's Summary from its transcript.
func attachSummaries(ctx context.Context, hist history.Reader, projects []model.Project) {
	for pi := range projects {
		for si := range projects[pi].WorkSessions {
			s := &projects[pi].WorkSessions[si]
			if s.Summary == "" {
				s.Summary = hist.SummaryFor(ctx, s.Harness, s.SessionID, s.Cwd)
			}
		}
	}
}

// formatSession renders a one-line human summary of a work-session.
func formatSession(s model.Session) string {
	branch := s.Branch
	if branch == "" {
		branch = "(detached)"
	}
	task := s.Task
	if task == "" {
		task = "—"
	}
	return fmt.Sprintf("%-6s %-8s %-24s %-6s %-24s +%d/-%d  %s",
		s.Status, s.Harness, s.Name, task, branch, s.ChangeStat.Added, s.ChangeStat.Removed, s.Summary)
}
