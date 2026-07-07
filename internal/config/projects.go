package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// trackedFile is the on-disk shape of the curated project list.
type trackedFile struct {
	Projects []string `toml:"projects"`
}

// TrackedPath is the location of the gvardia-managed curated project list. It is
// kept separate from config.toml so the TUI can rewrite it without clobbering the
// user's hand-written config and comments.
func TrackedPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "gvardia", "projects.toml")
	}
	return expandHome(filepath.Join("~", ".config", "gvardia", "projects.toml"))
}

// LoadTracked reads the curated project list, expanding "~" in each path. A
// missing file yields an empty list (curation is opt-in; callers fall back to a
// roots scan).
func LoadTracked() ([]string, error) {
	data, err := os.ReadFile(TrackedPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read tracked projects: %w", err)
	}
	var tf trackedFile
	if err := toml.Unmarshal(data, &tf); err != nil {
		return nil, fmt.Errorf("parse tracked projects: %w", err)
	}
	out := make([]string, 0, len(tf.Projects))
	for _, p := range tf.Projects {
		out = append(out, expandHome(p))
	}
	return out, nil
}

// SaveTracked writes the curated project list atomically (temp file + rename),
// creating the config directory if needed.
func SaveTracked(paths []string) error {
	path := TrackedPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(trackedFile{Projects: paths}); err != nil {
		return fmt.Errorf("encode tracked projects: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".projects-*.toml")
	if err != nil {
		return fmt.Errorf("temp file: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(buf.Bytes()); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("write tracked projects: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("replace tracked projects: %w", err)
	}
	return nil
}
