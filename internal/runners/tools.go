// Package runners discovers installed agent CLIs and describes how gvardia can
// launch them later as local runs.
package runners

import (
	"os/exec"

	"github.com/KirillSachkov/gvardia/internal/config"
)

// LookPath resolves an executable name. Tests pass a fake implementation.
type LookPath func(file string) (string, error)

// Tool is one installed or missing agent CLI tool.
type Tool struct {
	Name      string `json:"name"`
	Command   string `json:"command"`
	Path      string `json:"path,omitempty"`
	Installed bool   `json:"installed"`
	BuiltIn   bool   `json:"builtIn"`
}

var builtInTools = []config.Tool{
	{Name: "claude", Command: "claude"},
	{Name: "codex", Command: "codex"},
	{Name: "gemini", Command: "gemini"},
	{Name: "opencode", Command: "opencode"},
	{Name: "aider", Command: "aider"},
	{Name: "goose", Command: "goose"},
}

// BuiltInToolNames returns the built-in tool names in display order.
func BuiltInToolNames() []string {
	out := make([]string, 0, len(builtInTools))
	for _, tool := range builtInTools {
		out = append(out, tool.Name)
	}
	return out
}

// DiscoverTools returns built-in plus configured tools with installed status.
func DiscoverTools(cfg config.Config, lookup LookPath) []Tool {
	if lookup == nil {
		lookup = exec.LookPath
	}

	tools := make([]Tool, 0, len(builtInTools)+len(cfg.Tools))
	index := make(map[string]int, len(builtInTools)+len(cfg.Tools))
	for _, tool := range builtInTools {
		tools = append(tools, discover(tool, true, lookup))
		index[tool.Name] = len(tools) - 1
	}
	for _, tool := range cfg.Tools {
		if tool.Command == "" {
			tool.Command = tool.Name
		}
		discovered := discover(tool, false, lookup)
		if i, ok := index[tool.Name]; ok {
			discovered.BuiltIn = tools[i].BuiltIn
			tools[i] = discovered
			continue
		}
		tools = append(tools, discovered)
		index[tool.Name] = len(tools) - 1
	}
	return tools
}

func discover(tool config.Tool, builtIn bool, lookup LookPath) Tool {
	out := Tool{Name: tool.Name, Command: tool.Command, BuiltIn: builtIn}
	if out.Command == "" {
		out.Command = out.Name
	}
	path, err := lookup(out.Command)
	if err == nil && path != "" {
		out.Path = path
		out.Installed = true
	}
	return out
}
