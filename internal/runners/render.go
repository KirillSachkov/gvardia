package runners

import "strings"

// CommandData is substituted into a runner profile command template.
type CommandData struct {
	PromptPath   string
	WorktreePath string
	ReportPath   string
	TaskTitle    string
}

// RenderCommand replaces the placeholders supported by Phase 1 launch profiles.
func RenderCommand(profile RunnerProfile, data CommandData) string {
	replacer := strings.NewReplacer(
		"{{prompt_path}}", data.PromptPath,
		"{{worktree_path}}", data.WorktreePath,
		"{{report_path}}", data.ReportPath,
		"{{task_title}}", data.TaskTitle,
	)
	return replacer.Replace(profile.CommandTemplate)
}
