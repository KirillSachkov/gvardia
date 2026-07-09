// Package prompts renders task prompts for launched agent runs.
package prompts

import (
	"fmt"
	"strings"

	"github.com/KirillSachkov/gvardia/internal/model"
)

// Context is the data needed to render a coding-task prompt.
type Context struct {
	Task         model.Task
	ProjectName  string
	ProjectPath  string
	RunDir       string
	ReportPath   string
	StatusPath   string
	EventsPath   string
	ArtifactsDir string
}

// Render turns a task into the prompt saved for an agent run.
func Render(ctx Context) string {
	body := strings.TrimSpace(ctx.Task.Body)
	if body == "" {
		body = "No task body was provided."
	}
	return strings.TrimSpace(fmt.Sprintf(`# Task: %s

Project: %s
Project path: %s

Task body:
%s

Required report path:
%s

Gvardia run context:
- run directory: %s
- status file: %s
- events file: %s
- artifacts directory: %s

Gvardia evidence protocol:
- Use gvardia run status --state running --phase "<phase>" --summary "<one-line summary>" at meaningful phase changes.
- Use gvardia run event --type status --message "<what changed>" for important activity.
- Use gvardia run artifact --type plan --title "<title>" --file <path> for useful plans, notes, audits, logs, or review material.
- Write the final report to the required report path, or run gvardia run report --file <path>.
- Choose your own internal workflow, skills, and subagents; Gvardia only observes the work envelope.

Work requirements:
- inspect before editing;
- find existing project patterns before adding new abstractions;
- write a short plan before implementation;
- implement in small phases;
- add or update tests for changed behavior;
- run real verification commands and keep their output;
- write final report to the required report path;
- do not claim success without test output.
`, ctx.Task.Title,
		fallback(ctx.ProjectName, "(unknown)"),
		fallback(ctx.ProjectPath, "(unknown)"),
		body,
		fallback(ctx.ReportPath, ".gvardia/runs/<run-id>/report.md"),
		fallback(ctx.RunDir, ".gvardia/runs/<run-id>"),
		fallback(ctx.StatusPath, ".gvardia/runs/<run-id>/status.json"),
		fallback(ctx.EventsPath, ".gvardia/runs/<run-id>/events.jsonl"),
		fallback(ctx.ArtifactsDir, ".gvardia/runs/<run-id>/artifacts"),
	)) + "\n"
}

func fallback(value, fb string) string {
	if strings.TrimSpace(value) == "" {
		return fb
	}
	return value
}
