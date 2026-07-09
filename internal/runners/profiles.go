package runners

import (
	"errors"

	"github.com/KirillSachkov/gvardia/internal/config"
)

// RunnerProfile describes one way to launch an agent tool.
type RunnerProfile struct {
	Name            string `json:"name"`
	Tool            string `json:"tool"`
	CommandTemplate string `json:"commandTemplate"`
	BuiltIn         bool   `json:"builtIn"`
}

var builtInProfiles = []RunnerProfile{
	{Name: "claude", Tool: "claude", CommandTemplate: "claude {{prompt_path}}", BuiltIn: true},
	{Name: "codex", Tool: "codex", CommandTemplate: "codex {{prompt_path}}", BuiltIn: true},
	{Name: "gemini", Tool: "gemini", CommandTemplate: "gemini {{prompt_path}}", BuiltIn: true},
	{Name: "opencode", Tool: "opencode", CommandTemplate: "opencode {{prompt_path}}", BuiltIn: true},
	{Name: "aider", Tool: "aider", CommandTemplate: "aider {{prompt_path}}", BuiltIn: true},
	{Name: "goose", Tool: "goose", CommandTemplate: "goose {{prompt_path}}", BuiltIn: true},
}

// Profiles returns built-in plus configured runner profiles.
func Profiles(cfg config.Config) []RunnerProfile {
	profiles := make([]RunnerProfile, 0, len(builtInProfiles)+len(cfg.RunnerProfiles))
	index := make(map[string]int, len(builtInProfiles)+len(cfg.RunnerProfiles))
	for _, profile := range builtInProfiles {
		profiles = append(profiles, profile)
		index[profile.Name] = len(profiles) - 1
	}
	for _, profile := range cfg.RunnerProfiles {
		next := RunnerProfile{
			Name:            profile.Name,
			Tool:            profile.Tool,
			CommandTemplate: profile.CommandTemplate,
		}
		if i, ok := index[next.Name]; ok {
			next.BuiltIn = profiles[i].BuiltIn
			profiles[i] = next
			continue
		}
		profiles = append(profiles, next)
		index[next.Name] = len(profiles) - 1
	}
	return profiles
}

// ValidateProfile checks that a runner profile has the fields needed to launch.
func ValidateProfile(profile RunnerProfile) error {
	if profile.Name == "" {
		return errors.New("runner profile name is required")
	}
	if profile.Tool == "" {
		return errors.New("runner profile tool is required")
	}
	if profile.CommandTemplate == "" {
		return errors.New("runner profile command_template is required")
	}
	return nil
}
