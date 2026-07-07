package ui

import (
	"path/filepath"
	"testing"

	"github.com/KirillSachkov/gvardia/internal/config"
	"github.com/KirillSachkov/gvardia/internal/model"
)

func TestDDiffIssuesExec(t *testing.T) {
	m := ready(t)
	_, cmd := step(m, keyText("d"))
	if cmd == nil {
		t.Fatal("d with a selection should issue a diff exec command")
	}
}

func TestDiffCommandChoosesViewer(t *testing.T) {
	wt := model.Worktree{Path: "/tmp/wt", BaseBranch: "main"}

	// A configured lazygit that does not exist forces the git fallback.
	cfg := config.Default()
	cfg.Commands.Lazygit = "definitely-not-a-real-binary-xyz"
	cmd := diffCommand(wt, cfg)

	if base := filepath.Base(cmd.Args[0]); base != "git" {
		t.Errorf("fallback command = %q, want git", base)
	}
	found := false
	for i, a := range cmd.Args {
		if a == "-C" && i+1 < len(cmd.Args) && cmd.Args[i+1] == wt.Path {
			found = true
		}
	}
	if !found {
		t.Errorf("git fallback should target -C %s, got %v", wt.Path, cmd.Args)
	}
}
