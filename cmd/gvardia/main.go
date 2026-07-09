// Command gvardia is a terminal cockpit over a fleet of coding agents across all
// your projects. This file wires flags to subcommands; the work lives in the
// internal packages.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/KirillSachkov/gvardia/internal/config"
)

// version is overridden at release time via -ldflags "-X main.version=…".
var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "gvardia:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("gvardia", flag.ContinueOnError)
	showVersion := fs.Bool("version", false, "print version and exit")
	configPath := fs.String("config", config.DefaultPath(), "path to config file")
	fs.Usage = usage(fs)

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if *showVersion {
		fmt.Println("gvardia", version)
		return nil
	}

	switch fs.Arg(0) {
	case "agents":
		return runAgents(fs.Args()[1:], *configPath)
	case "projects":
		return runProjects(fs.Args()[1:], *configPath)
	case "tasks":
		return runTasks(fs.Args()[1:], *configPath)
	case "tools":
		return runTools(fs.Args()[1:], *configPath)
	case "run":
		return runRun(fs.Args()[1:])
	case "":
		return runCockpit(*configPath)
	default:
		return fmt.Errorf("unknown command %q (try --help)", fs.Arg(0))
	}
}
