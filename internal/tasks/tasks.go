// Package tasks reads the personal kanban board — the Markdown files under
// <brain>/tasks/{inbox,active,done} — into flat model.Task values. It parses only
// the YAML frontmatter (flat scalars); the body is ignored. Read-only.
package tasks

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/KirillSachkov/gvardia/internal/model"
)

// columns are the kanban directories, in board order.
var columns = []string{"inbox", "active", "done"}

// Load walks <brainRoot>/tasks/{inbox,active,done}/*.md and returns one task per
// file, in board order. A missing brain or column directory yields no tasks
// (never an error): the kanban is optional.
func Load(ctx context.Context, brainRoot string) []model.Task {
	var out []model.Task
	for _, col := range columns {
		dir := filepath.Join(brainRoot, "tasks", col)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			select {
			case <-ctx.Done():
				return out
			default:
			}
			path := filepath.Join(dir, e.Name())
			fm := readFrontmatter(path)
			slug := strings.TrimSuffix(e.Name(), ".md")
			out = append(out, model.Task{
				ID:      firstNonEmpty(fm["id"], slug),
				Title:   firstNonEmpty(fm["title"], slug),
				Status:  col,
				Project: fm["project"],
				Path:    path,
			})
		}
	}
	return out
}

// LinkTasks fills each session's Task with the matched kanban task's title, when
// the branch-inferred reference (already in Session.Task, e.g. "#675") matches a
// task's ID. Sessions without a match keep their reference.
func LinkTasks(projects []model.Project, tasks []model.Task) {
	byID := make(map[string]model.Task, len(tasks))
	for _, t := range tasks {
		if t.ID != "" {
			byID[t.ID] = t
		}
	}
	for pi := range projects {
		for si := range projects[pi].WorkSessions {
			s := &projects[pi].WorkSessions[si]
			ref := s.Task
			if ref == "" {
				continue
			}
			if t, ok := byID[ref]; ok && t.Title != "" {
				s.Task = t.Title
			} else if t, ok := byID[strings.TrimPrefix(ref, "#")]; ok && t.Title != "" {
				s.Task = t.Title
			}
		}
	}
}

// readFrontmatter parses the flat YAML scalar block between the first two "---"
// fences into a key→value map. Non-scalar lines (list items, nested blocks) are
// skipped. A file without a leading fence yields an empty map.
func readFrontmatter(path string) map[string]string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	fm := map[string]string{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	started := false
	for sc.Scan() {
		line := sc.Text()
		if !started {
			if strings.TrimSpace(line) == "---" {
				started = true
			}
			// Anything before the opening fence (including its absence) is not
			// frontmatter; the very first non-blank line must be the fence.
			if strings.TrimSpace(line) != "" && !started {
				return fm
			}
			continue
		}
		if strings.TrimSpace(line) == "---" {
			break
		}
		if k, v, ok := splitScalar(line); ok {
			fm[k] = v
		}
	}
	return fm
}

// splitScalar splits "key: value" on the first colon, trimming quotes. Lines
// with no colon, an empty key, or a leading list marker are rejected.
func splitScalar(line string) (string, string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "#") {
		return "", "", false
	}
	i := strings.IndexByte(line, ':')
	if i < 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:i])
	if key == "" {
		return "", "", false
	}
	val := strings.TrimSpace(line[i+1:])
	val = strings.Trim(val, `"'`)
	return key, val, true
}

// firstNonEmpty returns the first non-empty string among its arguments.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
