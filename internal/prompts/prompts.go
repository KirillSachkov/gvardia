// Package prompts renders task prompts for launched agent runs.
package prompts

import (
	"fmt"
	"strings"

	"github.com/KirillSachkov/gvardia/internal/model"
)

// Context is the data needed to render a coding-task prompt.
type Context struct {
	Task        model.Task
	ProjectName string
	ProjectPath string
	ReportPath  string
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

Work requirements:
- inspect before editing;
- find existing project patterns before adding new abstractions;
- write a short plan before implementation;
- implement in small phases;
- add or update tests for changed behavior;
- run real verification commands and keep their output;
- write final report to the required report path;
- do not claim success without test output.
`, ctx.Task.Title, fallback(ctx.ProjectName, "(unknown)"), fallback(ctx.ProjectPath, "(unknown)"), body, fallback(ctx.ReportPath, ".gvardia/runs/<run-id>/report.md"))) + "\n"
}

func fallback(value, fb string) string {
	if strings.TrimSpace(value) == "" {
		return fb
	}
	return value
}
