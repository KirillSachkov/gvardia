package runners

import (
	"errors"
	"reflect"
	"testing"

	"github.com/KirillSachkov/gvardia/internal/config"
)

func TestDiscoverToolsMarksBuiltInsInstalledOrMissing(t *testing.T) {
	lookup := func(name string) (string, error) {
		switch name {
		case "claude":
			return "/bin/claude", nil
		case "codex":
			return "/bin/codex", nil
		default:
			return "", errors.New("missing")
		}
	}

	got := DiscoverTools(config.Config{}, lookup)
	byName := toolsByName(got)

	if !byName["claude"].Installed || byName["claude"].Path != "/bin/claude" {
		t.Errorf("claude = %+v, want installed at /bin/claude", byName["claude"])
	}
	if !byName["codex"].Installed || byName["codex"].Path != "/bin/codex" {
		t.Errorf("codex = %+v, want installed at /bin/codex", byName["codex"])
	}
	for _, name := range []string{"gemini", "opencode", "aider", "goose"} {
		if byName[name].Installed {
			t.Errorf("%s installed = true, want false", name)
		}
	}
}

func TestDiscoverToolsIncludesCustomTools(t *testing.T) {
	cfg := config.Config{
		Tools: []config.Tool{
			{Name: "local-agent", Command: "local-agent-cli"},
		},
	}
	lookup := func(name string) (string, error) {
		if name == "local-agent-cli" {
			return "/opt/bin/local-agent-cli", nil
		}
		return "", errors.New("missing")
	}

	got := DiscoverTools(cfg, lookup)
	tool := toolsByName(got)["local-agent"]

	if tool.Name != "local-agent" {
		t.Fatalf("custom tool missing from %+v", got)
	}
	if tool.Command != "local-agent-cli" || !tool.Installed || tool.Path != "/opt/bin/local-agent-cli" {
		t.Errorf("custom tool = %+v, want installed local-agent-cli", tool)
	}
	if tool.BuiltIn {
		t.Error("custom tool BuiltIn = true, want false")
	}
}

func TestRunnerProfilesReturnsBuiltInsAndCustomProfiles(t *testing.T) {
	cfg := config.Config{
		RunnerProfiles: []config.RunnerProfile{
			{Name: "claude-review", Tool: "claude", CommandTemplate: "claude --print {{prompt_path}}"},
		},
	}

	got := Profiles(cfg)
	byName := profilesByName(got)

	for _, name := range []string{"claude", "codex", "gemini", "opencode", "aider", "goose"} {
		if _, ok := byName[name]; !ok {
			t.Fatalf("builtin profile %q missing from %+v", name, got)
		}
	}
	custom := byName["claude-review"]
	if custom.Tool != "claude" || custom.CommandTemplate != "claude --print {{prompt_path}}" {
		t.Errorf("custom profile = %+v, want configured values", custom)
	}
	if custom.BuiltIn {
		t.Error("custom profile BuiltIn = true, want false")
	}
}

func TestValidateProfileRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		profile RunnerProfile
	}{
		{name: "missing name", profile: RunnerProfile{Tool: "claude", CommandTemplate: "claude {{prompt_path}}"}},
		{name: "missing tool", profile: RunnerProfile{Name: "claude", CommandTemplate: "claude {{prompt_path}}"}},
		{name: "missing template", profile: RunnerProfile{Name: "claude", Tool: "claude"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateProfile(tt.profile); err == nil {
				t.Fatal("ValidateProfile error = nil, want error")
			}
		})
	}
}

func TestValidateProfileAcceptsCompleteProfile(t *testing.T) {
	profile := RunnerProfile{Name: "claude", Tool: "claude", CommandTemplate: "claude {{prompt_path}}"}

	if err := ValidateProfile(profile); err != nil {
		t.Fatalf("ValidateProfile error = %v, want nil", err)
	}
}

func toolsByName(tools []Tool) map[string]Tool {
	out := make(map[string]Tool, len(tools))
	for _, tool := range tools {
		out[tool.Name] = tool
	}
	return out
}

func profilesByName(profiles []RunnerProfile) map[string]RunnerProfile {
	out := make(map[string]RunnerProfile, len(profiles))
	for _, profile := range profiles {
		out[profile.Name] = profile
	}
	return out
}

func TestBuiltInToolNames(t *testing.T) {
	got := BuiltInToolNames()
	want := []string{"claude", "codex", "gemini", "opencode", "aider", "goose"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("BuiltInToolNames = %v, want %v", got, want)
	}
}
