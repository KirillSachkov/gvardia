package history

import (
	"bufio"
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/KirillSachkov/gvardia/internal/model"
)

// noisyPrefix reports whether text starts with a marker of injected/non-prompt
// content (instruction blocks, comments, codex's box-drawing banner).
func noisyPrefix(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return strings.ContainsRune("<#╭│╰─", r)
}

// codexSessions returns ended codex sessions for cwd, newest first, bounded.
func (r Reader) codexSessions(ctx context.Context, cwd string, limit int, since time.Duration) []model.Session {
	if r.CodexRoot == "" {
		return nil
	}
	cutoff := r.cutoff(since)

	type item struct {
		path  string
		id    string
		mtime time.Time
	}
	var items []item
	_ = filepath.WalkDir(r.CodexRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if !cutoff.IsZero() && info.ModTime().Before(cutoff) {
			return nil
		}
		mcwd, id := codexHead(path)
		if mcwd != cwd {
			return nil
		}
		items = append(items, item{path: path, id: id, mtime: info.ModTime()})
		return nil
	})

	sort.Slice(items, func(i, j int) bool { return items[i].mtime.After(items[j].mtime) })
	if len(items) > limit {
		items = items[:limit]
	}

	out := make([]model.Session, 0, len(items))
	for _, it := range items {
		out = append(out, model.Session{
			Harness:      "codex",
			Live:         false,
			Cwd:          cwd,
			SessionID:    it.id,
			Name:         shortID(it.id),
			Summary:      codexSummary(it.path),
			Report:       codexReport(it.path),
			LastActivity: it.mtime,
		})
	}
	return out
}

// codexReport returns the last substantive assistant message from a codex
// rollout, cleaned and truncated for display.
func codexReport(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	var last string
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			var rec struct {
				Type    string `json:"type"`
				Payload struct {
					Type    string `json:"type"`
					Role    string `json:"role"`
					Content []struct {
						Type string `json:"type"`
						Text string `json:"text"`
					} `json:"content"`
				} `json:"payload"`
			}
			if json.Unmarshal(line, &rec) == nil &&
				rec.Type == "response_item" && rec.Payload.Type == "message" && rec.Payload.Role == "assistant" {
				var texts []string
				for _, c := range rec.Payload.Content {
					if t := strings.TrimSpace(c.Text); t != "" {
						texts = append(texts, t)
					}
				}
				if j := strings.Join(texts, "\n"); j != "" {
					last = j
				}
			}
		}
		if err != nil {
			break
		}
	}
	return truncate(strings.TrimSpace(last), 500)
}

// codexHead reads the session_meta header (first line) for cwd + session id.
func codexHead(path string) (cwd, id string) {
	f, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer f.Close()
	line, err := bufio.NewReader(f).ReadBytes('\n')
	if err != nil && len(line) == 0 {
		return "", ""
	}
	var rec struct {
		Payload struct {
			Cwd       string `json:"cwd"`
			SessionID string `json:"session_id"`
		} `json:"payload"`
	}
	if json.Unmarshal(line, &rec) != nil {
		return "", ""
	}
	return rec.Payload.Cwd, rec.Payload.SessionID
}

// codexSummary returns the first genuine user prompt, skipping injected context
// (AGENTS.md dumps, instruction blocks, "<…>" / "#…" wrappers).
func codexSummary(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			var rec struct {
				Type    string `json:"type"`
				Payload struct {
					Type    string `json:"type"`
					Role    string `json:"role"`
					Content []struct {
						Type string `json:"type"`
						Text string `json:"text"`
					} `json:"content"`
				} `json:"payload"`
			}
			if json.Unmarshal(line, &rec) == nil &&
				rec.Type == "response_item" && rec.Payload.Type == "message" && rec.Payload.Role == "user" {
				for _, c := range rec.Payload.Content {
					if t := strings.TrimSpace(c.Text); t != "" && !noisyPrefix(t) {
						return truncate(t, 100)
					}
				}
			}
		}
		if err != nil {
			break
		}
	}
	return ""
}
