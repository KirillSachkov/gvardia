package terminal

import (
	"context"
	"reflect"
	"testing"
)

func TestTmuxLaunchBuildsDetachedSessionCommand(t *testing.T) {
	var calls []call
	svc := TmuxService{
		Runner: fakeRunner(func(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
			calls = append(calls, call{dir: dir, name: name, args: args})
			return nil, nil
		}),
	}

	target, err := svc.Launch(context.Background(), LaunchSpec{
		RunID:       "run-123",
		Worktree:    "/repo/gvardia-wt",
		Command:     "claude /tmp/prompt.md",
		Target:      "custom-target",
		WindowTitle: "Build console",
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	if target != "custom-target" {
		t.Fatalf("target = %q, want custom-target", target)
	}
	if len(calls) != 2 {
		t.Fatalf("calls = %d, want 2", len(calls))
	}
	wantArgs := []string{"new-session", "-d", "-s", "custom-target", "-c", "/repo/gvardia-wt", "-n", "Build console", "sh", "-lc", "claude /tmp/prompt.md"}
	if calls[0].name != "tmux" || !reflect.DeepEqual(calls[0].args, wantArgs) {
		t.Fatalf("call = %+v, want tmux %v", calls[0], wantArgs)
	}
	wantRemain := []string{"set-option", "-t", "custom-target", "remain-on-exit", "on"}
	if calls[1].name != "tmux" || !reflect.DeepEqual(calls[1].args, wantRemain) {
		t.Fatalf("remain call = %+v, want tmux %v", calls[1], wantRemain)
	}
}

func TestTmuxLaunchExportsRunEnvironment(t *testing.T) {
	var calls []call
	svc := TmuxService{
		Runner: fakeRunner(func(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
			calls = append(calls, call{dir: dir, name: name, args: args})
			return nil, nil
		}),
	}

	_, err := svc.Launch(context.Background(), LaunchSpec{
		RunID:    "run-123",
		Worktree: "/repo/gvardia-wt",
		Command:  "claude /tmp/prompt.md",
		Env: map[string]string{
			"GVARDIA_RUN_DIR":     "/repo/.gvardia/runs/run-123",
			"GVARDIA_REPORT_PATH": "/repo/.gvardia/runs/run-123/report.md",
		},
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("calls = %d, want 2", len(calls))
	}
	got := calls[0].args[len(calls[0].args)-1]
	want := "export GVARDIA_REPORT_PATH='/repo/.gvardia/runs/run-123/report.md'; export GVARDIA_RUN_DIR='/repo/.gvardia/runs/run-123'; claude /tmp/prompt.md"
	if got != want {
		t.Fatalf("shell command = %q, want %q", got, want)
	}
}

func TestTmuxInspectReportsLiveAndDeadPane(t *testing.T) {
	outputs := [][]byte{[]byte("0|\n"), []byte("1|7\n")}
	svc := TmuxService{Runner: fakeRunner(func(context.Context, string, string, ...string) ([]byte, error) {
		out := outputs[0]
		outputs = outputs[1:]
		return out, nil
	})}

	live, err := svc.Inspect(context.Background(), "live")
	if err != nil {
		t.Fatalf("Inspect live: %v", err)
	}
	if !live.Alive || live.ExitCode != 0 {
		t.Fatalf("live state = %+v, want alive", live)
	}
	dead, err := svc.Inspect(context.Background(), "dead")
	if err != nil {
		t.Fatalf("Inspect dead: %v", err)
	}
	if dead.Alive || dead.ExitCode != 7 {
		t.Fatalf("dead state = %+v, want dead exit 7", dead)
	}
}

func TestTmuxAttachAndKillCommands(t *testing.T) {
	var calls []call
	svc := TmuxService{
		Runner: fakeRunner(func(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
			calls = append(calls, call{dir: dir, name: name, args: args})
			return nil, nil
		}),
	}

	if err := svc.Attach(context.Background(), "gvardia-run-123"); err != nil {
		t.Fatalf("Attach: %v", err)
	}
	if err := svc.Kill(context.Background(), "gvardia-run-123"); err != nil {
		t.Fatalf("Kill: %v", err)
	}

	want := []call{
		{name: "tmux", args: []string{"attach", "-t", "gvardia-run-123"}},
		{name: "tmux", args: []string{"kill-session", "-t", "gvardia-run-123"}},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %+v, want %+v", calls, want)
	}
}

type call struct {
	dir  string
	name string
	args []string
}

type fakeRunner func(context.Context, string, string, ...string) ([]byte, error)

func (f fakeRunner) Run(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	return f(ctx, dir, name, args...)
}
