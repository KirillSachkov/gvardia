package runners

import (
	"strings"
	"testing"

	"github.com/KirillSachkov/gvardia/internal/config"
)

func TestBuiltInCodexProfileUsesFullAccess(t *testing.T) {
	profiles := Profiles(config.Default())
	profile, index := DefaultProfile(profiles, "codex")
	if index < 0 || profile.Name != "codex" {
		t.Fatalf("DefaultProfile codex = %+v, %d; want codex", profile, index)
	}
	for _, flag := range []string{"-a never", "-s danger-full-access", "-C '{{worktree_path}}'"} {
		if !strings.Contains(profile.CommandTemplate, flag) {
			t.Errorf("Codex command %q missing %q", profile.CommandTemplate, flag)
		}
	}
}

func TestDefaultProfileFallsBackToFirst(t *testing.T) {
	profiles := []RunnerProfile{{Name: "one"}, {Name: "two"}}
	profile, index := DefaultProfile(profiles, "missing")
	if index != 0 || profile.Name != "one" {
		t.Fatalf("DefaultProfile fallback = %+v, %d; want first", profile, index)
	}
}
