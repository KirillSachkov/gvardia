package adapters

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/KirillSachkov/gvardia/internal/model"
)

// defaultStaleness is used when a Codex has no configured Staleness window.
const defaultStaleness = 15 * time.Second

// procLister reports the working directories of running codex processes. It is
// the seam that makes liveness testable without spawning real processes.
type procLister interface {
	LiveCwds(ctx context.Context) (map[string]int, error) // cwd -> pid
}

// Codex reports sessions of the codex CLI. A session is live only when a codex
// process is actually running (v1 guessed from file mtime, which lied); the
// session files supply the id/summary/start time. Ended sessions are history
// (see internal/history), not reported here.
type Codex struct {
	Root      string        // sessions dir; empty = ~/.codex/sessions (tests override)
	Staleness time.Duration // busy if the newest file changed within this window
	now       func() time.Time
	lister    procLister // nil = codexProcLister (pgrep + lsof)
}

// Name identifies the adapter and harness.
func (Codex) Name() string { return "codex" }

// Sessions returns one session per running codex process, enriched from the
// newest matching session file. No running codex ⇒ no sessions.
func (c Codex) Sessions(ctx context.Context) ([]model.Session, error) {
	lister := c.lister
	if lister == nil {
		lister = codexProcLister{}
	}
	live, err := lister.LiveCwds(ctx)
	if err != nil {
		return nil, err
	}
	if len(live) == 0 {
		return nil, nil
	}

	newest := c.newestByCwd(ctx)
	stale := c.Staleness
	if stale == 0 {
		stale = defaultStaleness
	}
	now := time.Now
	if c.now != nil {
		now = c.now
	}

	sessions := make([]model.Session, 0, len(live))
	for cwd, pid := range live {
		s := model.Session{Harness: "codex", Live: true, Cwd: cwd, PID: pid, Name: "codex"}
		if m, ok := newest[cwd]; ok {
			s.SessionID = m.SessionID
			s.Name = shortID(m.SessionID)
			s.StartedAt = m.Started
			s.LastActivity = m.ModTime
			if now().Sub(m.ModTime) < stale {
				s.Status = model.StatusBusy
			} else {
				s.Status = model.StatusIdle
			}
		} else {
			s.Status = model.StatusBusy // running process, no session file yet
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

// newestByCwd walks the session logs and returns the newest file per cwd.
func (c Codex) newestByCwd(ctx context.Context) map[string]codexMeta {
	root := c.Root
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		root = filepath.Join(home, ".codex", "sessions")
	}
	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	newest := make(map[string]codexMeta)
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		m, ok := readCodexMeta(path)
		if !ok {
			return nil
		}
		if prev, seen := newest[m.Cwd]; !seen || m.ModTime.After(prev.ModTime) {
			newest[m.Cwd] = m
		}
		return nil
	})
	return newest
}

// codexProcLister is the production procLister: `pgrep -x codex` + `lsof` cwd.
type codexProcLister struct{}

// LiveCwds returns cwd→pid for each running codex process (excluding cwd "/").
func (codexProcLister) LiveCwds(ctx context.Context) (map[string]int, error) {
	out, err := exec.CommandContext(ctx, "pgrep", "-x", "codex").Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 1 {
			return map[string]int{}, nil // pgrep: no matches, not an error
		}
		return nil, err
	}
	res := make(map[string]int)
	for _, f := range strings.Fields(string(out)) {
		pid, convErr := strconv.Atoi(f)
		if convErr != nil {
			continue
		}
		if cwd := lsofCwd(ctx, pid); cwd != "" && cwd != "/" {
			res[cwd] = pid
		}
	}
	return res, nil
}

// lsofCwd resolves a process's current working directory via lsof.
func lsofCwd(ctx context.Context, pid int) string {
	out, err := exec.CommandContext(ctx, "lsof", "-a", "-p", strconv.Itoa(pid), "-d", "cwd", "-Fn").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "n") {
			return strings.TrimPrefix(line, "n")
		}
	}
	return ""
}

type codexMeta struct {
	Cwd       string
	SessionID string
	Started   time.Time
	ModTime   time.Time
}

// readCodexMeta reads the session_meta header (the first JSONL line) of a codex
// rollout file. The first line can be large (embedded instructions), so it uses
// an unbounded reader rather than bufio.Scanner.
func readCodexMeta(path string) (codexMeta, bool) {
	f, err := os.Open(path)
	if err != nil {
		return codexMeta{}, false
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return codexMeta{}, false
	}
	line, err := bufio.NewReader(f).ReadBytes('\n')
	if err != nil && len(line) == 0 {
		return codexMeta{}, false
	}

	var rec struct {
		Type    string `json:"type"`
		Payload struct {
			Cwd       string `json:"cwd"`
			SessionID string `json:"session_id"`
			Timestamp string `json:"timestamp"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(line, &rec); err != nil || rec.Payload.Cwd == "" {
		return codexMeta{}, false
	}
	started, _ := time.Parse(time.RFC3339, rec.Payload.Timestamp)
	return codexMeta{
		Cwd:       rec.Payload.Cwd,
		SessionID: rec.Payload.SessionID,
		Started:   started,
		ModTime:   fi.ModTime(),
	}, true
}
