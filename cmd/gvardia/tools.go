package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/KirillSachkov/gvardia/internal/config"
	"github.com/KirillSachkov/gvardia/internal/runners"
)

type toolsOutput struct {
	Tools    []runners.Tool          `json:"tools"`
	Profiles []runners.RunnerProfile `json:"profiles"`
}

// runTools implements `gvardia tools [--json]`: it reports the agent CLI tools
// and runner profiles gvardia can use for local runs.
func runTools(args []string, configPath string) error {
	fs := flag.NewFlagSet("gvardia tools", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "emit tools and runner profiles as JSON")
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
	out := toolsOutput{
		Tools:    runners.DiscoverTools(cfg, exec.LookPath),
		Profiles: runners.Profiles(cfg),
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	for _, tool := range out.Tools {
		status := "missing"
		if tool.Installed {
			status = "installed"
		}
		fmt.Fprintf(os.Stdout, "%-10s %-9s %s\n", tool.Name, status, tool.Command)
	}
	return nil
}
