package config

import (
	"path/filepath"
	"testing"
)

func TestTrackedRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("HOME", home)

	// "~/proj" must be stored verbatim and expanded to $HOME/proj on load.
	want := []string{"/abs/one", "~/proj"}
	if err := SaveTracked(want); err != nil {
		t.Fatalf("SaveTracked: %v", err)
	}
	if path := TrackedPath(); path != filepath.Join(tmp, "gvardia", "projects.toml") {
		t.Errorf("TrackedPath = %q under XDG_CONFIG_HOME", path)
	}

	got, err := LoadTracked()
	if err != nil {
		t.Fatalf("LoadTracked: %v", err)
	}
	expect := []string{"/abs/one", filepath.Join(home, "proj")}
	if len(got) != len(expect) {
		t.Fatalf("LoadTracked = %v, want %v", got, expect)
	}
	for i := range expect {
		if got[i] != expect[i] {
			t.Errorf("LoadTracked[%d] = %q, want %q", i, got[i], expect[i])
		}
	}
}

func TestLoadTrackedMissingIsEmpty(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	got, err := LoadTracked()
	if err != nil {
		t.Fatalf("LoadTracked on missing file: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("missing tracked file should yield empty, got %v", got)
	}
}
