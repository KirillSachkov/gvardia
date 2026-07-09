package terminal

import (
	"context"
	"reflect"
	"testing"
)

func TestCmuxOpenBuildsNewWorkspaceCommand(t *testing.T) {
	var got call
	svc := CmuxService{Runner: fakeRunner(func(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
		got = call{dir: dir, name: name, args: args}
		return nil, nil
	})}

	err := svc.Open(context.Background(), OpenSpec{
		Name: "Reliable launch", Cwd: "/repo/worktree",
		Command: "tmux attach -t 'gvardia-run-123'", Focus: true,
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	want := call{name: "cmux", args: []string{
		"new-workspace", "--name", "Reliable launch", "--cwd", "/repo/worktree",
		"--command", "tmux attach -t 'gvardia-run-123'", "--focus", "true",
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("cmux call = %+v, want %+v", got, want)
	}
}

func TestAttachCommandQuotesTarget(t *testing.T) {
	if got, want := AttachCommand("gvardia run's"), "tmux attach -t 'gvardia run'\\''s'"; got != want {
		t.Fatalf("AttachCommand = %q, want %q", got, want)
	}
}
