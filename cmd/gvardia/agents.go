package main

import (
	"errors"
	"flag"
	"fmt"
)

// runAgents implements the hidden `agents` subcommand: a headless fleet dump.
// Phase 0 is a stub — Phase 2 wires the adapters and emits real sessions.
func runAgents(args []string) error {
	fs := flag.NewFlagSet("gvardia agents", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "emit the fleet as JSON")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if *asJSON {
		// No adapters wired yet (Phase 2): an empty fleet is the truthful result.
		fmt.Println("[]")
		return nil
	}
	fmt.Println("agents: no adapters wired yet (Phase 2)")
	return nil
}
