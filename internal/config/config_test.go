package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	got := Default()
	if want := []string{"~/code"}; !reflect.DeepEqual(got.Roots, want) {
		t.Errorf("Roots = %v, want %v", got.Roots, want)
	}
	if got.RefreshInterval.Duration != 5*time.Second {
		t.Errorf("RefreshInterval = %v, want 5s", got.RefreshInterval.Duration)
	}
	if want := []string{"claude", "codex", "tmux"}; !reflect.DeepEqual(got.Adapters, want) {
		t.Errorf("Adapters = %v, want %v", got.Adapters, want)
	}
	if got.Base["default"] != "auto" {
		t.Errorf("Base[default] = %q, want auto", got.Base["default"])
	}
	if got.Commands.Lazygit != "lazygit" {
		t.Errorf("Commands.Lazygit = %q, want lazygit", got.Commands.Lazygit)
	}
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	missing := filepath.Join(t.TempDir(), "does-not-exist.toml")

	got, err := Load(missing)
	if err != nil {
		t.Fatalf("Load(missing) error = %v, want nil", err)
	}
	// Roots default is expanded, so it should equal <home>/code, not "~/code".
	if want := []string{filepath.Join(home, "code")}; !reflect.DeepEqual(got.Roots, want) {
		t.Errorf("Roots = %v, want %v", got.Roots, want)
	}
	if got.RefreshInterval.Duration != 5*time.Second {
		t.Errorf("RefreshInterval = %v, want 5s", got.RefreshInterval.Duration)
	}
}

func TestLoadOverrideMerge(t *testing.T) {
	// Only refresh_interval and one base override are set; every other field
	// must keep its default.
	body := `
refresh_interval = "10s"

[base]
"education-platform" = "dev"
`
	cfg := writeAndLoad(t, body)

	if cfg.RefreshInterval.Duration != 10*time.Second {
		t.Errorf("RefreshInterval = %v, want 10s (override)", cfg.RefreshInterval.Duration)
	}
	if want := []string{"claude", "codex", "tmux"}; !reflect.DeepEqual(cfg.Adapters, want) {
		t.Errorf("Adapters = %v, want default %v", cfg.Adapters, want)
	}
	if cfg.Base["default"] != "auto" {
		t.Errorf("Base[default] = %q, want default auto to survive merge", cfg.Base["default"])
	}
	if cfg.Base["education-platform"] != "dev" {
		t.Errorf("Base[education-platform] = %q, want dev", cfg.Base["education-platform"])
	}
}

func TestLoadCustomToolsAndRunnerProfiles(t *testing.T) {
	cfg := writeAndLoad(t, `
[[tools]]
name = "local-agent"
command = "local-agent-cli"

[[runner_profiles]]
name = "local-review"
tool = "local-agent"
command_template = "local-agent-cli run {{prompt_path}}"
`)

	if len(cfg.Tools) != 1 {
		t.Fatalf("Tools = %d, want 1", len(cfg.Tools))
	}
	if cfg.Tools[0].Name != "local-agent" || cfg.Tools[0].Command != "local-agent-cli" {
		t.Errorf("Tools[0] = %+v, want local-agent/local-agent-cli", cfg.Tools[0])
	}
	if len(cfg.RunnerProfiles) != 1 {
		t.Fatalf("RunnerProfiles = %d, want 1", len(cfg.RunnerProfiles))
	}
	got := cfg.RunnerProfiles[0]
	if got.Name != "local-review" || got.Tool != "local-agent" || got.CommandTemplate != "local-agent-cli run {{prompt_path}}" {
		t.Errorf("RunnerProfiles[0] = %+v, want configured profile", got)
	}
}

func TestLoadExpandsHomeInRoots(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	cfg := writeAndLoad(t, `roots = ["~/work", "/abs/path", "~"]`)

	want := []string{filepath.Join(home, "work"), "/abs/path", home}
	if !reflect.DeepEqual(cfg.Roots, want) {
		t.Errorf("Roots = %v, want %v", cfg.Roots, want)
	}
}

func TestLoadInvalidTOMLErrors(t *testing.T) {
	if _, err := writeAndLoadErr(t, `roots = [unclosed`); err == nil {
		t.Fatal("Load(invalid TOML) error = nil, want error")
	}
}

func TestLoadInvalidDurationErrors(t *testing.T) {
	if _, err := writeAndLoadErr(t, `refresh_interval = "5furlongs"`); err == nil {
		t.Fatal("Load(bad duration) error = nil, want error")
	}
}

// writeAndLoad writes body to a temp config file and returns the loaded config,
// failing the test on any load error.
func writeAndLoad(t *testing.T, body string) Config {
	t.Helper()
	cfg, err := writeAndLoadErr(t, body)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg
}

func writeAndLoadErr(t *testing.T, body string) (Config, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return Load(path)
}
