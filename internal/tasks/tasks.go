// Package tasks reads the personal kanban board — the Markdown files under
// <brain>/tasks/{inbox,active,done} — into flat model.Task values. It parses only
// the YAML frontmatter (flat scalars); the body is ignored. Read-only.
package tasks

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/KirillSachkov/gvardia/internal/model"
)

// columns are the kanban directories, in board order.
var columns = []string{"inbox", "active", "done"}

// LoadLocal walks <project>/.gvardia/tasks/*.md and returns local project tasks.
// Local tasks are writable and keep their Markdown body for prompt rendering.
func LoadLocal(ctx context.Context, projectPath string) []model.Task {
	dir := filepath.Join(projectPath, ".gvardia", "tasks")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	project := filepath.Base(projectPath)
	var out []model.Task
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
		fm, body := readMarkdownTask(path)
		slug := strings.TrimSuffix(e.Name(), ".md")
		out = append(out, model.Task{
			ID:      firstNonEmpty(fm["id"], slug),
			Title:   firstNonEmpty(fm["title"], slug),
			Status:  firstNonEmpty(fm["status"], "inbox"),
			Project: firstNonEmpty(fm["project"], project),
			Path:    path,
			Body:    strings.TrimSpace(body),
			Source:  "local",
		})
	}
	return out
}

// CreateLocal writes task to <project>/.gvardia/tasks/<slug>.md and returns the
// normalized task. It does not overwrite existing files.
func CreateLocal(projectPath string, task model.Task) (model.Task, error) {
	if task.Title == "" {
		return model.Task{}, fmt.Errorf("task title is required")
	}
	if task.ID == "" {
		task.ID = slugify(task.Title)
	}
	if task.Status == "" {
		task.Status = "inbox"
	}
	if task.Project == "" {
		task.Project = filepath.Base(projectPath)
	}
	task.Source = "local"

	dir := filepath.Join(projectPath, ".gvardia", "tasks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return model.Task{}, fmt.Errorf("create tasks dir: %w", err)
	}
	task.Path = filepath.Join(dir, slugify(task.ID)+".md")
	if _, err := os.Stat(task.Path); err == nil {
		return model.Task{}, fmt.Errorf("task already exists: %s", task.Path)
	} else if !os.IsNotExist(err) {
		return model.Task{}, fmt.Errorf("stat task: %w", err)
	}

	content := fmt.Sprintf("---\nid: %q\ntitle: %q\nstatus: %q\nproject: %q\n---\n\n%s\n",
		task.ID, task.Title, task.Status, task.Project, strings.TrimSpace(task.Body))
	if err := os.WriteFile(task.Path, []byte(content), 0o644); err != nil {
		return model.Task{}, fmt.Errorf("write task: %w", err)
	}
	return task, nil
}

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
				Source:  "brain",
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
	fm, _ := readMarkdownTask(path)
	return fm
}

func readMarkdownTask(path string) (map[string]string, string) {
	f, err := os.Open(path)
	if err != nil {
		return nil, ""
	}
	defer f.Close()

	fm := map[string]string{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	started := false
	closed := false
	var body strings.Builder
	for sc.Scan() {
		line := sc.Text()
		if closed {
			body.WriteString(line)
			body.WriteByte('\n')
			continue
		}
		if !started {
			if strings.TrimSpace(line) == "---" {
				started = true
			}
			// Anything before the opening fence (including its absence) is not
			// frontmatter; the very first non-blank line must be the fence.
			if strings.TrimSpace(line) != "" && !started {
				body.WriteString(line)
				body.WriteByte('\n')
				closed = true
			}
			continue
		}
		if strings.TrimSpace(line) == "---" {
			closed = true
			break
		}
		if k, v, ok := splitScalar(line); ok {
			fm[k] = v
		}
	}
	for sc.Scan() {
		body.WriteString(sc.Text())
		body.WriteByte('\n')
	}
	return fm, strings.TrimSpace(body.String())
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

var slugRE = regexp.MustCompile(`[^a-z0-9._-]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(s, "#")))
	s = slugRE.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "task"
	}
	return s
}
