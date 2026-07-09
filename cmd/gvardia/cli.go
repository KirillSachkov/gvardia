package main

import (
	"flag"
	"fmt"

	"github.com/KirillSachkov/gvardia/internal/config"
	"github.com/KirillSachkov/gvardia/internal/ui"
)

// runCockpit is the default command (no subcommand): it loads config and launches
// the Bubble Tea cockpit.
func runCockpit(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	return ui.Run(cfg)
}

// usage prints top-level help. The hidden `agents` subcommand is intentionally
// undocumented until Phase 2 makes it a real fleet dump.
func usage(fs *flag.FlagSet) func() {
	return func() {
		out := fs.Output()
		fmt.Fprintln(out, "gvardia — a terminal cockpit over your fleet of coding agents.")
		fmt.Fprintln(out, "\nUsage:\n  gvardia [flags]\n  gvardia agents [--json]\n  gvardia projects [--json]\n  gvardia tasks\n  gvardia tools [--json]\n\nFlags:")
		fmt.Fprintln(out, "\nRun telemetry:\n  gvardia run status --run-dir DIR --state running --phase PHASE --summary TEXT\n  gvardia run event --run-dir DIR --type TYPE --message TEXT\n  gvardia run artifact --run-dir DIR --type TYPE --title TITLE --file PATH\n  gvardia run report --run-dir DIR --file PATH")
		fs.PrintDefaults()
	}
}
