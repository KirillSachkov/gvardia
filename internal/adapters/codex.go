package adapters

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/KirillSachkov/gvardia/internal/model"
)

// defaultStaleness is used when a Codex has no configured Staleness window.
const defaultStaleness = 15 * time.Second

// Codex reports sessions from ~/.codex/sessions/**/*.jsonl, keeping the newest
// file per cwd. It has no CLI: status comes from file freshness.
type Codex struct {
	Root      string        // sessions dir; empty = ~/.codex/sessions (tests override)
	Staleness time.Duration // busy if the newest file changed within this window
	now       func() time.Time
}

// Name identifies the adapter and harness.
func (Codex) Name() string { return "codex" }

// Sessions walks the session logs and returns one session per cwd (the newest).
func (c Codex) Sessions(ctx context.Context) ([]model.Session, error) {
	root := c.Root
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		root = filepath.Join(home, ".codex", "sessions")
	}
	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		return nil, nil // codex not installed here — no sessions, not an error
	}

	newest := make(map[string]codexMeta)
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		meta, ok := readCodexMeta(path)
		if !ok {
			return nil
		}
		if prev, seen := newest[meta.Cwd]; !seen || meta.ModTime.After(prev.ModTime) {
			newest[meta.Cwd] = meta
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	stale := c.Staleness
	if stale == 0 {
		stale = defaultStaleness
	}
	now := time.Now
	if c.now != nil {
		now = c.now
	}

	sessions := make([]model.Session, 0, len(newest))
	for cwd, m := range newest {
		status := model.StatusIdle
		if now().Sub(m.ModTime) < stale {
			status = model.StatusBusy
		}
		sessions = append(sessions, model.Session{
			Harness:   "codex",
			Name:      shortID(m.SessionID),
			SessionID: m.SessionID,
			Cwd:       cwd,
			Status:    status,
			StartedAt: m.Started,
		})
	}
	return sessions, nil
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
