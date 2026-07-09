package runs

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TelemetryStatus is the latest agent-reported status for a run.
type TelemetryStatus struct {
	State       Status    `json:"state"`
	Phase       string    `json:"phase,omitempty"`
	Summary     string    `json:"summary,omitempty"`
	NeedsReview bool      `json:"needsReview,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Event is one append-only run activity entry.
type Event struct {
	Time    time.Time `json:"time"`
	Type    string    `json:"type"`
	Message string    `json:"message"`
}

// RunArtifact is a useful output produced by an agent run.
type RunArtifact struct {
	Type  string `json:"type"`
	Title string `json:"title"`
	Path  string `json:"path"`
}

// ArtifactInput describes an artifact to copy into a run or index in place.
type ArtifactInput struct {
	Type  string
	Title string
	File  string
	Body  string
}

// WriteStatus writes status.json in a run directory.
func (s Store) WriteStatus(runDir string, status TelemetryStatus) error {
	if runDir == "" {
		return fmt.Errorf("run dir is required")
	}
	if status.State == "" {
		status.State = StatusRunning
	}
	if status.UpdatedAt.IsZero() {
		status.UpdatedAt = s.now()
	}
	return writeJSON(filepath.Join(runDir, "status.json"), status)
}

// AppendEvent appends an event to events.jsonl in a run directory.
func (s Store) AppendEvent(runDir string, event Event) error {
	if runDir == "" {
		return fmt.Errorf("run dir is required")
	}
	if event.Type == "" {
		event.Type = "note"
	}
	if event.Time.IsZero() {
		event.Time = s.now()
	}
	path := filepath.Join(runDir, "events.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create events dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open events: %w", err)
	}
	defer f.Close()
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode event: %w", err)
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write event: %w", err)
	}
	return nil
}

// SaveArtifact copies or writes an artifact and updates artifacts.json.
func (s Store) SaveArtifact(runDir string, in ArtifactInput) (RunArtifact, error) {
	if runDir == "" {
		return RunArtifact{}, fmt.Errorf("run dir is required")
	}
	if in.Type == "" {
		in.Type = "note"
	}
	if in.Title == "" {
		in.Title = in.Type
	}
	artifactsDir := filepath.Join(runDir, "artifacts")
	if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
		return RunArtifact{}, fmt.Errorf("create artifacts dir: %w", err)
	}

	name := safeArtifactName(in.Title)
	if in.File != "" {
		name = filepath.Base(in.File)
	}
	if filepath.Ext(name) == "" {
		name += ".md"
	}
	dest := filepath.Join(artifactsDir, name)
	switch {
	case in.File != "":
		if err := copyFile(dest, in.File); err != nil {
			return RunArtifact{}, err
		}
	default:
		if err := os.WriteFile(dest, []byte(in.Body), 0o644); err != nil {
			return RunArtifact{}, fmt.Errorf("write artifact: %w", err)
		}
	}

	artifact := RunArtifact{Type: in.Type, Title: in.Title, Path: filepath.ToSlash(filepath.Join("artifacts", filepath.Base(dest)))}
	path := filepath.Join(runDir, "artifacts.json")
	list := readArtifacts(path)
	list = upsertArtifact(list, artifact)
	if err := writeJSON(path, list); err != nil {
		return RunArtifact{}, err
	}
	return artifact, nil
}

// WriteReport writes report.md and indexes it as a report artifact.
func (s Store) WriteReport(runDir string, body []byte) error {
	if runDir == "" {
		return fmt.Errorf("run dir is required")
	}
	reportPath := filepath.Join(runDir, "report.md")
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		return fmt.Errorf("create report dir: %w", err)
	}
	if err := os.WriteFile(reportPath, body, 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	path := filepath.Join(runDir, "artifacts.json")
	list := ensureReportArtifact(readArtifacts(path))
	return writeJSON(path, list)
}

func readStatus(path string) TelemetryStatus {
	data, err := os.ReadFile(path)
	if err != nil {
		return TelemetryStatus{}
	}
	var status TelemetryStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return TelemetryStatus{}
	}
	return status
}

func readEvents(path string) []Event {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event Event
		if err := json.Unmarshal([]byte(line), &event); err == nil {
			out = append(out, event)
		}
	}
	return out
}

func readArtifacts(path string) []RunArtifact {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []RunArtifact
	if err := json.Unmarshal(data, &out); err != nil {
		return nil
	}
	return out
}

func ensureReportArtifact(list []RunArtifact) []RunArtifact {
	return upsertArtifact(list, RunArtifact{Type: "report", Title: "Final report", Path: "report.md"})
}

func upsertArtifact(list []RunArtifact, next RunArtifact) []RunArtifact {
	for i := range list {
		if list[i].Path == next.Path {
			list[i] = next
			return list
		}
	}
	return append(list, next)
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create %s dir: %w", filepath.Base(path), err)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", filepath.Base(path), err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	return nil
}

func safeArtifactName(title string) string {
	name := strings.ToLower(strings.TrimSpace(title))
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "-", ":", "-", "\t", "-")
	name = replacer.Replace(name)
	name = strings.Trim(name, "-")
	if name == "" {
		return "artifact"
	}
	return name
}

func copyFile(dest, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open artifact source: %w", err)
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create artifact: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy artifact: %w", err)
	}
	return nil
}
