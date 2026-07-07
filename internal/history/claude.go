package history

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/KirillSachkov/gvardia/internal/model"
)

// encodeCwd maps a working directory to claude's project-dir name, which is the
// path with separators and dots replaced by "-".
func encodeCwd(cwd string) string {
	return strings.NewReplacer("/", "-", ".", "-").Replace(cwd)
}

// claudeSessions returns ended claude sessions whose transcripts live in the
// project directory for cwd, newest first, bounded by limit/since.
func (r Reader) claudeSessions(cwd string, limit int, since time.Duration) []model.Session {
	if r.ClaudeRoot == "" {
		return nil
	}
	dir := filepath.Join(r.ClaudeRoot, encodeCwd(cwd))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	cutoff := r.cutoff(since)

	type item struct {
		path  string
		id    string
		mtime time.Time
	}
	var items []item
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if !cutoff.IsZero() && info.ModTime().Before(cutoff) {
			continue
		}
		items = append(items, item{
			path:  filepath.Join(dir, e.Name()),
			id:    strings.TrimSuffix(e.Name(), ".jsonl"),
			mtime: info.ModTime(),
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].mtime.After(items[j].mtime) })
	if len(items) > limit {
		items = items[:limit]
	}

	out := make([]model.Session, 0, len(items))
	for _, it := range items {
		out = append(out, model.Session{
			Harness:      "claude",
			Live:         false,
			Cwd:          cwd,
			SessionID:    it.id,
			Name:         shortID(it.id),
			Summary:      claudeSummary(it.path),
			Report:       claudeReport(it.path),
			LastActivity: it.mtime,
		})
	}
	return out
}

// claudeSummary extracts a session title: the AI-generated title if present,
// else the first genuine user prompt (skipping meta/command-wrapped messages).
func claudeSummary(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	var title, firstUser string
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			var rec struct {
				Type    string `json:"type"`
				AITitle string `json:"aiTitle"`
				IsMeta  bool   `json:"isMeta"`
				Message struct {
					Content json.RawMessage `json:"content"`
				} `json:"message"`
			}
			if json.Unmarshal(line, &rec) == nil {
				switch {
				case rec.Type == "ai-title" && rec.AITitle != "":
					title = rec.AITitle // keep the latest
				case rec.Type == "user" && !rec.IsMeta && firstUser == "":
					if t := userText(rec.Message.Content); t != "" && !strings.HasPrefix(t, "<") {
						firstUser = t
					}
				}
			}
		}
		if err != nil {
			break
		}
	}
	if title != "" {
		return truncate(title, 100)
	}
	return truncate(firstUser, 100)
}

// claudeReport returns the last substantive assistant text message from a claude
// transcript, cleaned and truncated for display.
func claudeReport(path string) string {
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
				Message struct {
					Content json.RawMessage `json:"content"`
				} `json:"message"`
			}
			if json.Unmarshal(line, &rec) == nil && rec.Type == "assistant" {
				if t := assistantText(rec.Message.Content); t != "" {
					last = t
				}
			}
		}
		if err != nil {
			break
		}
	}
	return truncate(strings.TrimSpace(last), 500)
}

// assistantText joins the text parts of a claude assistant message (skipping
// tool_use blocks), or returns a plain-string content as-is.
func assistantText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return strings.TrimSpace(s)
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &parts) == nil {
		var texts []string
		for _, p := range parts {
			if p.Type == "text" && strings.TrimSpace(p.Text) != "" {
				texts = append(texts, strings.TrimSpace(p.Text))
			}
		}
		return strings.Join(texts, "\n")
	}
	return ""
}

// userText pulls plain text out of a claude message content, which is either a
// string or an array of typed parts.
func userText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return strings.TrimSpace(s)
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &parts) == nil {
		for _, p := range parts {
			if p.Type == "text" && strings.TrimSpace(p.Text) != "" {
				return strings.TrimSpace(p.Text)
			}
		}
	}
	return ""
}
