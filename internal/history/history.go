// Package history reads past (ended) agent sessions from on-disk logs — claude
// transcripts and codex rollouts — and extracts a human summary plus last
// activity. It is lazy and bounded: callers query per selected project, never
// for the whole fleet, because scanning transcripts is expensive.
package history

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/KirillSachkov/gvardia/internal/model"
)

// defaultLimit caps sessions per harness when Options.Limit is unset.
const defaultLimit = 8

// Options bound how much history to read.
type Options struct {
	Limit int           // max sessions per harness (0 → defaultLimit)
	Since time.Duration // ignore sessions older than this (0 → no cutoff)
}

// Reader locates the agent log roots. The zero value is not usable; call New (or
// set the roots explicitly in tests).
type Reader struct {
	ClaudeRoot string // ~/.claude/projects
	CodexRoot  string // ~/.codex/sessions
	now        func() time.Time
}

// New returns a Reader pointed at the user's real log directories.
func New() Reader {
	home, _ := os.UserHomeDir()
	return Reader{
		ClaudeRoot: filepath.Join(home, ".claude", "projects"),
		CodexRoot:  filepath.Join(home, ".codex", "sessions"),
	}
}

// Recent returns ended sessions for a worktree cwd from all harnesses, newest
// first, bounded by opts.
func (r Reader) Recent(ctx context.Context, cwd string, opts Options) []model.Session {
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	var out []model.Session
	out = append(out, r.claudeSessions(cwd, limit, opts.Since)...)
	out = append(out, r.codexSessions(ctx, cwd, limit, opts.Since)...)
	sort.Slice(out, func(i, j int) bool { return out[i].LastActivity.After(out[j].LastActivity) })
	return out
}

// SummaryFor returns a summary for a known (typically live) session by locating
// its transcript from harness + id + cwd. Empty when it can't be found.
func (r Reader) SummaryFor(ctx context.Context, harness, sessionID, cwd string) string {
	if sessionID == "" {
		return ""
	}
	switch harness {
	case "claude":
		if r.ClaudeRoot == "" {
			return ""
		}
		return claudeSummary(filepath.Join(r.ClaudeRoot, encodeCwd(cwd), sessionID+".jsonl"))
	case "codex":
		if r.CodexRoot == "" {
			return ""
		}
		var summary string
		_ = filepath.WalkDir(r.CodexRoot, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".jsonl") {
				return nil //nolint:nilerr
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if _, id := codexHead(path); id == sessionID {
				summary = codexSummary(path)
				return filepath.SkipAll
			}
			return nil
		})
		return summary
	}
	return ""
}

// cutoff is the oldest LastActivity to include, or the zero time for no cutoff.
func (r Reader) cutoff(since time.Duration) time.Time {
	if since <= 0 {
		return time.Time{}
	}
	now := time.Now
	if r.now != nil {
		now = r.now
	}
	return now().Add(-since)
}

// shortID trims a session id to a compact display name.
func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// truncate clips a summary to n runes with an ellipsis.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
