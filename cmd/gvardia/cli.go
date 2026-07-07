package main

import (
	"flag"
	"fmt"

	"github.com/KirillSachkov/gvardia/internal/config"
)

// runCockpit is the default command (no subcommand). Phase 3 replaces this with
// the Bubble Tea UI; for now it loads the config and reports readiness so the
// binary is useful and proves config loading end-to-end.
func runCockpit(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	fmt.Printf("gvardia %s — cockpit UI lands in Phase 3.\n", version)
	fmt.Printf("scanning %d root(s): %v\n", len(cfg.Roots), cfg.Roots)
	fmt.Println("try: gvardia --version | gvardia agents --json")
	return nil
}

// usage prints top-level help. The hidden `agents` subcommand is intentionally
// undocumented until Phase 2 makes it a real fleet dump.
func usage(fs *flag.FlagSet) func() {
	return func() {
		out := fs.Output()
		fmt.Fprintln(out, "gvardia — a terminal cockpit over your fleet of coding agents.")
		fmt.Fprintln(out, "\nUsage:\n  gvardia [flags]\n\nFlags:")
		fs.PrintDefaults()
	}
}
