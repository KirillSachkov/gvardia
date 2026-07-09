package ui

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/KirillSachkov/gvardia/internal/config"
	"github.com/KirillSachkov/gvardia/internal/model"
	"github.com/KirillSachkov/gvardia/internal/runs"
	"github.com/KirillSachkov/gvardia/internal/terminal"
)

func TestDDiffIssuesExec(t *testing.T) {
	m := ready(t)
	_, cmd := step(m, keyText("d"))
	if cmd == nil {
		t.Fatal("d with a selection should issue a diff exec command")
	}
}

func TestNewRunIDUsesSubSecondPrecision(t *testing.T) {
	base := time.Date(2026, 7, 9, 12, 0, 0, 123456000, time.UTC)
	first := newRunID(base)
	second := newRunID(base.Add(time.Microsecond))
	if first == second {
		t.Fatalf("newRunID collided: %q", first)
	}
	if !strings.HasPrefix(first, "run-20260709-120000-") {
		t.Fatalf("newRunID = %q, want timestamp prefix", first)
	}
}

func TestReconciledRunStatus(t *testing.T) {
	cases := []struct {
		name   string
		run    runs.Run
		pane   terminal.PaneState
		err    error
		status runs.Status
	}{
		{"live stays running", runs.Run{Status: runs.StatusRunning}, terminal.PaneState{Alive: true}, nil, runs.StatusRunning},
		{"dead without report fails", runs.Run{Status: runs.StatusRunning}, terminal.PaneState{ExitCode: 7}, nil, runs.StatusFailed},
		{"dead with report needs review", runs.Run{Status: runs.StatusRunning, Report: "done"}, terminal.PaneState{}, nil, runs.StatusReview},
		{"missing target fails", runs.Run{Status: runs.StatusPending}, terminal.PaneState{}, errors.New("missing"), runs.StatusFailed},
		{"done remains done", runs.Run{Status: runs.StatusDone}, terminal.PaneState{}, errors.New("missing"), runs.StatusDone},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := reconciledRunStatus(tc.run, tc.pane, tc.err)
			if got != tc.status {
				t.Fatalf("status = %q, want %q", got, tc.status)
			}
		})
	}
}

func TestOpenTerminalCopyBackendReturnsFallbackCommand(t *testing.T) {
	cfg := config.Default()
	cfg.Terminal.Backend = "copy"
	msg := openTerminal("run", "/repo/wt", "tmux attach -t 'run-1'", cfg)()
	fallback, ok := msg.(terminalFallbackMsg)
	if !ok {
		t.Fatalf("openTerminal copy message = %T, want terminalFallbackMsg", msg)
	}
	if fallback.command != "tmux attach -t 'run-1'" || fallback.err != nil {
		t.Fatalf("fallback = %+v, want exact command without error", fallback)
	}
}

func TestDiffCommandChoosesViewer(t *testing.T) {
	wt := model.Worktree{Path: "/tmp/wt", BaseBranch: "main"}

	// A configured lazygit that does not exist forces the git fallback.
	cfg := config.Default()
	cfg.Commands.Lazygit = "definitely-not-a-real-binary-xyz"
	cfg.Terminal.Backend = "copy"
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

func TestDiffCommandPrefersCmux(t *testing.T) {
	wt := model.Worktree{Path: "/tmp/wt", BaseBranch: "main"}
	lookup := func(name string) (string, error) {
		if name == "cmux" {
			return "/usr/local/bin/cmux", nil
		}
		return "", errors.New("missing")
	}
	cmd := diffCommandWithLookPath(wt, config.Default(), lookup)
	if filepath.Base(cmd.Args[0]) != "cmux" {
		t.Fatalf("diff command = %v, want cmux", cmd.Args)
	}
	joined := strings.Join(cmd.Args, " ")
	for _, want := range []string{"diff", "--branch", "--repo /tmp/wt", "--base main"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("cmux diff args = %v, missing %q", cmd.Args, want)
		}
	}
}
