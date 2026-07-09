package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestRunToolsJSONIncludesCustomMissingTool(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	body := `
[[tools]]
name = "definitely-missing-agent"
command = "gvardia-definitely-missing-agent"

[[runner_profiles]]
name = "missing-runner"
tool = "definitely-missing-agent"
command_template = "gvardia-definitely-missing-agent {{prompt_path}}"
`
	if err := os.WriteFile(configPath, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	out := captureStdout(t, func() {
		if err := run([]string{"-config", configPath, "tools", "--json"}); err != nil {
			t.Fatalf("run tools: %v", err)
		}
	})

	var payload struct {
		Tools []struct {
			Name      string `json:"name"`
			Command   string `json:"command"`
			Installed bool   `json:"installed"`
			BuiltIn   bool   `json:"builtIn"`
		} `json:"tools"`
		Profiles []struct {
			Name            string `json:"name"`
			Tool            string `json:"tool"`
			CommandTemplate string `json:"commandTemplate"`
			BuiltIn         bool   `json:"builtIn"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, out)
	}

	if !hasTool(payload.Tools, "claude") || !hasTool(payload.Tools, "codex") ||
		!hasTool(payload.Tools, "gemini") || !hasTool(payload.Tools, "opencode") ||
		!hasTool(payload.Tools, "aider") || !hasTool(payload.Tools, "goose") {
		t.Fatalf("built-in tools missing from %+v", payload.Tools)
	}
	foundCustom := false
	for _, tool := range payload.Tools {
		if tool.Name == "definitely-missing-agent" {
			foundCustom = true
			if tool.Command != "gvardia-definitely-missing-agent" || tool.Installed || tool.BuiltIn {
				t.Errorf("custom tool = %+v, want missing custom command", tool)
			}
		}
	}
	if !foundCustom {
		t.Fatalf("custom tool missing from %+v", payload.Tools)
	}

	foundProfile := false
	for _, profile := range payload.Profiles {
		if profile.Name == "missing-runner" {
			foundProfile = true
			if profile.Tool != "definitely-missing-agent" || profile.CommandTemplate == "" || profile.BuiltIn {
				t.Errorf("custom profile = %+v, want configured custom profile", profile)
			}
		}
	}
	if !foundProfile {
		t.Fatalf("custom profile missing from %+v", payload.Profiles)
	}
}

func hasTool(tools []struct {
	Name      string `json:"name"`
	Command   string `json:"command"`
	Installed bool   `json:"installed"`
	BuiltIn   bool   `json:"builtIn"`
}, name string) bool {
	for _, tool := range tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}

func captureStdout(t *testing.T, fn func()) []byte {
	t.Helper()

	original := os.Stdout
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = write
	defer func() {
		os.Stdout = original
	}()

	fn()

	if err := write.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, read); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return buf.Bytes()
}
