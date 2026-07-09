package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/KirillSachkov/gvardia/internal/runs"
)

// runRun implements `gvardia run ...` helpers used by launched agents to report
// status, activity, artifacts, and final reports back into the local run folder.
func runRun(args []string) error {
	if len(args) == 0 {
		return errors.New("run subcommand is required: status, event, artifact, or report")
	}
	switch args[0] {
	case "status":
		return runStatus(args[1:])
	case "event":
		return runEvent(args[1:])
	case "artifact":
		return runArtifact(args[1:])
	case "report":
		return runReport(args[1:])
	default:
		return fmt.Errorf("unknown run command %q", args[0])
	}
}

func runStatus(args []string) error {
	fs := flag.NewFlagSet("gvardia run status", flag.ContinueOnError)
	runDir := fs.String("run-dir", "", "run directory; defaults to GVARDIA_RUN_DIR")
	state := fs.String("state", string(runs.StatusRunning), "run state")
	phase := fs.String("phase", "", "current work phase")
	summary := fs.String("summary", "", "short status summary")
	needsReview := fs.Bool("needs-review", false, "mark the run as needing human review")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	dir, err := requireRunDir(*runDir)
	if err != nil {
		return err
	}
	status, err := parseRunStatus(*state)
	if err != nil {
		return err
	}
	return (runs.Store{}).WriteStatus(dir, runs.TelemetryStatus{
		State:       status,
		Phase:       *phase,
		Summary:     *summary,
		NeedsReview: *needsReview,
	})
}

func runEvent(args []string) error {
	fs := flag.NewFlagSet("gvardia run event", flag.ContinueOnError)
	runDir := fs.String("run-dir", "", "run directory; defaults to GVARDIA_RUN_DIR")
	eventType := fs.String("type", "note", "event type")
	message := fs.String("message", "", "event message")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	dir, err := requireRunDir(*runDir)
	if err != nil {
		return err
	}
	if *message == "" {
		return errors.New("event message is required")
	}
	return (runs.Store{}).AppendEvent(dir, runs.Event{Type: *eventType, Message: *message})
}

func runArtifact(args []string) error {
	fs := flag.NewFlagSet("gvardia run artifact", flag.ContinueOnError)
	runDir := fs.String("run-dir", "", "run directory; defaults to GVARDIA_RUN_DIR")
	artifactType := fs.String("type", "note", "artifact type")
	title := fs.String("title", "", "artifact title")
	file := fs.String("file", "", "file to copy into the run artifacts directory")
	body := fs.String("body", "", "inline artifact body")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	dir, err := requireRunDir(*runDir)
	if err != nil {
		return err
	}
	if *file == "" && *body == "" {
		return errors.New("artifact requires --file or --body")
	}
	_, err = (runs.Store{}).SaveArtifact(dir, runs.ArtifactInput{
		Type:  *artifactType,
		Title: *title,
		File:  *file,
		Body:  *body,
	})
	return err
}

func runReport(args []string) error {
	fs := flag.NewFlagSet("gvardia run report", flag.ContinueOnError)
	runDir := fs.String("run-dir", "", "run directory; defaults to GVARDIA_RUN_DIR")
	file := fs.String("file", "", "report markdown file to copy to report.md")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	dir, err := requireRunDir(*runDir)
	if err != nil {
		return err
	}
	if *file == "" {
		return errors.New("report file is required")
	}
	body, err := os.ReadFile(*file)
	if err != nil {
		return fmt.Errorf("read report: %w", err)
	}
	return (runs.Store{}).WriteReport(dir, body)
}

func requireRunDir(value string) (string, error) {
	if value == "" {
		value = os.Getenv("GVARDIA_RUN_DIR")
	}
	if value == "" {
		return "", errors.New("run dir is required (--run-dir or GVARDIA_RUN_DIR)")
	}
	return value, nil
}

func parseRunStatus(value string) (runs.Status, error) {
	status := runs.Status(value)
	switch status {
	case runs.StatusPending, runs.StatusRunning, runs.StatusReview, runs.StatusDone, runs.StatusFailed, runs.StatusKilled:
		return status, nil
	default:
		return "", fmt.Errorf("unknown run state %q", value)
	}
}
