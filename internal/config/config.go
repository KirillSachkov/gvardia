// Package config loads gvardia's TOML configuration, applying defaults for any
// field the user leaves unset. A missing config file is not an error: the
// defaults are returned as-is.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// Config is the fully-resolved gvardia configuration. Every field has a useful
// zero value via [Default]; user config overrides only the keys it sets.
type Config struct {
	// Roots are directories scanned (shallow) for git repos. "~" is expanded.
	Roots []string `toml:"roots"`
	// RefreshInterval is how often the cockpit re-runs collectors/adapters.
	RefreshInterval Duration `toml:"refresh_interval"`
	// Adapters lists the enabled agent adapters by name, e.g. "claude".
	Adapters []string `toml:"adapters"`
	// Base maps a project name to its base branch for diff/ahead-behind. The
	// special key "default" applies to any project without an explicit entry;
	// "auto" means dev if it exists else main (resolved by the collector).
	Base map[string]string `toml:"base"`
	// Commands overrides the external CLIs gvardia shells out to.
	Commands Commands `toml:"commands"`
	// Brain is the root of the personal kanban (sachkov-os); its
	// tasks/{inbox,active,done}/*.md files are the single task source. "~" is expanded.
	Brain string `toml:"brain"`
}

// Commands holds overridable paths to the external CLIs gvardia invokes.
type Commands struct {
	Lazygit string `toml:"lazygit"`
}

// BaseBranch returns the configured base branch for a project, falling back to
// the "default" entry (which may be "auto" for the collector to resolve).
func (c Config) BaseBranch(project string) string {
	if b, ok := c.Base[project]; ok {
		return b
	}
	return c.Base["default"]
}

// Default returns the built-in configuration used when a key is unset.
func Default() Config {
	return Config{
		Roots:           []string{"~/code"},
		RefreshInterval: Duration{5 * time.Second},
		Adapters:        []string{"claude", "codex", "tmux"},
		Base:            map[string]string{"default": "auto"},
		Commands:        Commands{Lazygit: "lazygit"},
		Brain:           "~/Work/sachkov-os",
	}
}

// Load reads the TOML config at path over the defaults, so any key the file
// omits keeps its default. A non-existent file yields the defaults unchanged.
func Load(path string) (Config, error) {
	cfg := Default()
	expanded := expandHome(path)

	data, err := os.ReadFile(expanded)
	if errors.Is(err, os.ErrNotExist) {
		return finalize(cfg), nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", expanded, err)
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", expanded, err)
	}
	return finalize(cfg), nil
}

// DefaultPath is the config location gvardia reads when --config is unset:
// $XDG_CONFIG_HOME/gvardia/config.toml, falling back to ~/.config.
func DefaultPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "gvardia", "config.toml")
	}
	return expandHome(filepath.Join("~", ".config", "gvardia", "config.toml"))
}

// ExpandPath expands a leading "~" in p to the user's home directory. It is the
// exported form of the config's own path handling, for callers that take paths
// (e.g. roots) on the command line.
func ExpandPath(p string) string { return expandHome(p) }

// finalize applies post-decode normalization: expand "~" in every root and in
// the brain path.
func finalize(cfg Config) Config {
	for i, r := range cfg.Roots {
		cfg.Roots[i] = expandHome(r)
	}
	cfg.Brain = expandHome(cfg.Brain)
	return cfg
}

// expandHome replaces a leading "~" (alone or as "~/…") with the user's home
// directory. Anything else, or a lookup failure, is returned unchanged.
func expandHome(p string) string {
	if p != "~" && !strings.HasPrefix(p, "~/") {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if p == "~" {
		return home
	}
	return filepath.Join(home, p[len("~/"):])
}

// Duration wraps [time.Duration] so it decodes from a TOML string like "5s".
type Duration struct {
	time.Duration
}

// UnmarshalText parses a Go duration string (e.g. "5s", "500ms") from TOML.
func (d *Duration) UnmarshalText(text []byte) error {
	parsed, err := time.ParseDuration(string(text))
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", text, err)
	}
	d.Duration = parsed
	return nil
}
