// Package adapters is gvardia's agent-agnostic seam. Each harness (claude, codex,
// tmux, …) is one file implementing [Adapter]; the core knows only this interface.
// Adapters degrade gracefully: an absent CLI or unparseable source returns an
// error and is skipped, never fatal. See docs/ADAPTERS.md.
package adapters

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/KirillSachkov/gvardia/internal/config"
	"github.com/KirillSachkov/gvardia/internal/model"
)

// Adapter reports the agent sessions for one harness. Implementations must honor
// ctx, return (nil, err) when their source is absent so the caller can skip them,
// and never panic or block the UI.
type Adapter interface {
	Name() string
	Sessions(ctx context.Context) ([]model.Session, error)
}

// registry maps an adapter name to its constructor. Supporting a new harness is
// one new file plus one entry here.
var registry = map[string]func(config.Config) Adapter{
	"claude": func(config.Config) Adapter { return Claude{} },
	"codex":  func(cfg config.Config) Adapter { return Codex{Staleness: cfg.RefreshInterval.Duration * 3} },
	"tmux":   func(config.Config) Adapter { return Tmux{} },
}

// Enabled returns the adapters named in cfg.Adapters, in order, skipping unknown
// names.
func Enabled(cfg config.Config) []Adapter {
	var out []Adapter
	for _, name := range cfg.Adapters {
		if ctor, ok := registry[name]; ok {
			out = append(out, ctor(cfg))
		}
	}
	return out
}

// Failure records that one adapter could not report, so the UI can show a banner
// instead of failing the whole fleet.
type Failure struct {
	Adapter string
	Err     error
}

// CollectSessions runs every adapter concurrently and merges their sessions.
// Per-adapter failures are collected and returned, not treated as fatal.
func CollectSessions(ctx context.Context, ads []Adapter) ([]model.Session, []Failure) {
	var (
		mu       sync.Mutex
		sessions []model.Session
		failures []Failure
		wg       sync.WaitGroup
	)
	for _, a := range ads {
		wg.Add(1)
		go func(a Adapter) {
			defer wg.Done()
			s, err := a.Sessions(ctx)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				failures = append(failures, Failure{Adapter: a.Name(), Err: err})
				return
			}
			sessions = append(sessions, s...)
		}(a)
	}
	wg.Wait()
	return sessions, failures
}

// commandFunc runs an external command; adapters hold one so tests can fake it.
type commandFunc func(ctx context.Context, name string, args ...string) ([]byte, error)

// execCommand is the production commandFunc: it shells out and returns stdout.
func execCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

// shortID trims a session UUID to its first segment for a compact display name.
func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}
