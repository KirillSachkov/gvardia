// Package runs persists local gvardia agent runs under .gvardia/runs.
package runs

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/KirillSachkov/gvardia/internal/model"
)

// Status is a run lifecycle state.
type Status string

const (
	StatusPending Status = "pending"
	StatusRunning Status = "running"
	StatusReview  Status = "review"
	StatusDone    Status = "done"
	StatusFailed  Status = "failed"
	StatusKilled  Status = "killed"
)

// Run is one local agent execution tracked by gvardia.
type Run struct {
	ID            string           `json:"id"`
	Project       string           `json:"project"`
	ProjectPath   string           `json:"projectPath"`
	TaskID        string           `json:"taskId,omitempty"`
	TaskTitle     string           `json:"taskTitle,omitempty"`
	Runner        string           `json:"runner"`
	Tool          string           `json:"tool"`
	Status        Status           `json:"status"`
	TmuxTarget    string           `json:"tmuxTarget,omitempty"`
	WorktreePath  string           `json:"worktreePath,omitempty"`
	Branch        string           `json:"branch,omitempty"`
	PromptPath    string           `json:"promptPath"`
	MetaPath      string           `json:"metaPath"`
	ReportPath    string           `json:"reportPath"`
	StatusPath    string           `json:"statusPath,omitempty"`
	EventsPath    string           `json:"eventsPath,omitempty"`
	ArtifactsPath string           `json:"artifactsPath,omitempty"`
	ArtifactsDir  string           `json:"artifactsDir,omitempty"`
	Report        string           `json:"-"`
	Telemetry     TelemetryStatus  `json:"-"`
	Events        []Event          `json:"-"`
	RunArtifacts  []RunArtifact    `json:"-"`
	ChangeStat    model.ChangeStat `json:"-"`
	Artifacts     []model.Artifact `json:"-"`
	CreatedAt     time.Time        `json:"createdAt"`
	UpdatedAt     time.Time        `json:"updatedAt"`
}

// CreateInput is the caller-supplied data for creating a run.
type CreateInput struct {
	Project      string
	TaskID       string
	TaskTitle    string
	Runner       string
	Tool         string
	WorktreePath string
	Branch       string
	Prompt       string
	TmuxTarget   string
}

// Store writes and reads runs. Optional Now/NewID seams make tests stable.
type Store struct {
	Root  string
	Now   func() time.Time
	NewID func() string
}

// Create creates .gvardia/runs/<id>/, writes prompt.md and meta.json.
func (s Store) Create(projectPath string, in CreateInput) (Run, error) {
	now := s.now()
	id := s.newID(now)
	dir := s.runDir(projectPath, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Run{}, fmt.Errorf("create run dir: %w", err)
	}

	run := Run{
		ID:            id,
		Project:       in.Project,
		ProjectPath:   projectPath,
		TaskID:        in.TaskID,
		TaskTitle:     in.TaskTitle,
		Runner:        in.Runner,
		Tool:          in.Tool,
		Status:        StatusPending,
		TmuxTarget:    in.TmuxTarget,
		WorktreePath:  in.WorktreePath,
		Branch:        in.Branch,
		PromptPath:    filepath.Join(dir, "prompt.md"),
		MetaPath:      filepath.Join(dir, "meta.json"),
		ReportPath:    filepath.Join(dir, "report.md"),
		StatusPath:    filepath.Join(dir, "status.json"),
		EventsPath:    filepath.Join(dir, "events.jsonl"),
		ArtifactsPath: filepath.Join(dir, "artifacts.json"),
		ArtifactsDir:  filepath.Join(dir, "artifacts"),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := os.WriteFile(run.PromptPath, []byte(in.Prompt), 0o644); err != nil {
		return Run{}, fmt.Errorf("write prompt: %w", err)
	}
	if err := os.MkdirAll(run.ArtifactsDir, 0o755); err != nil {
		return Run{}, fmt.Errorf("create artifacts dir: %w", err)
	}
	run.Telemetry = TelemetryStatus{
		State:     StatusPending,
		Phase:     "created",
		Summary:   "Run created",
		UpdatedAt: now,
	}
	if err := s.WriteStatus(dir, run.Telemetry); err != nil {
		return Run{}, err
	}
	if err := os.WriteFile(run.EventsPath, nil, 0o644); err != nil {
		return Run{}, fmt.Errorf("write events: %w", err)
	}
	if err := writeJSON(run.ArtifactsPath, []RunArtifact{}); err != nil {
		return Run{}, err
	}
	if err := s.Save(run); err != nil {
		return Run{}, err
	}
	return run, nil
}

// Save writes a run's meta.json.
func (s Store) Save(run Run) error {
	if run.ID == "" {
		return errors.New("run id is required")
	}
	if run.ProjectPath == "" {
		return errors.New("run project path is required")
	}
	if run.MetaPath == "" {
		run.MetaPath = filepath.Join(s.runDir(run.ProjectPath, run.ID), "meta.json")
	}
	if run.PromptPath == "" {
		run.PromptPath = filepath.Join(s.runDir(run.ProjectPath, run.ID), "prompt.md")
	}
	if run.ReportPath == "" {
		run.ReportPath = filepath.Join(s.runDir(run.ProjectPath, run.ID), "report.md")
	}
	fillTelemetryPaths(&run)
	run.UpdatedAt = s.now()
	if run.CreatedAt.IsZero() {
		run.CreatedAt = run.UpdatedAt
	}
	if run.Status == "" {
		run.Status = StatusPending
	}
	if err := os.MkdirAll(filepath.Dir(run.MetaPath), 0o755); err != nil {
		return fmt.Errorf("create run dir: %w", err)
	}
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return fmt.Errorf("encode run: %w", err)
	}
	if err := os.WriteFile(run.MetaPath, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write run meta: %w", err)
	}
	return nil
}

// LoadProject reads all runs under <project>/.gvardia/runs, newest first.
func (s Store) LoadProject(projectPath string) ([]Run, error) {
	root := s.runsRoot(projectPath)
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read runs: %w", err)
	}
	var out []Run
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		meta := filepath.Join(root, e.Name(), "meta.json")
		data, err := os.ReadFile(meta)
		if err != nil {
			continue
		}
		var run Run
		if err := json.Unmarshal(data, &run); err != nil {
			continue
		}
		if s.Root != "" && filepath.Clean(run.ProjectPath) != filepath.Clean(projectPath) {
			continue
		}
		if run.ProjectPath == "" {
			run.ProjectPath = projectPath
		}
		if run.MetaPath == "" {
			run.MetaPath = meta
		}
		if run.PromptPath == "" {
			run.PromptPath = filepath.Join(root, e.Name(), "prompt.md")
		}
		if run.ReportPath == "" {
			run.ReportPath = filepath.Join(root, e.Name(), "report.md")
		}
		fillTelemetryPaths(&run)
		run.Telemetry = readStatus(run.StatusPath)
		if run.Telemetry.State != "" {
			run.Status = run.Telemetry.State
		}
		run.Events = readEvents(run.EventsPath)
		run.RunArtifacts = readArtifacts(run.ArtifactsPath)
		if report, err := os.ReadFile(run.ReportPath); err == nil {
			run.Report = strings.TrimSpace(string(report))
			if run.Report != "" {
				run.RunArtifacts = ensureReportArtifact(run.RunArtifacts)
			}
			if (run.Status == StatusRunning || run.Status == StatusPending) && run.Report != "" {
				run.Status = StatusReview
			}
		}
		out = append(out, run)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out, nil
}

func (s Store) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now().UTC()
}

func (s Store) newID(now time.Time) string {
	if s.NewID != nil {
		return s.NewID()
	}
	return "run-" + now.Format("20060102-150405")
}

func runDir(projectPath, id string) string {
	return filepath.Join(projectPath, ".gvardia", "runs", id)
}

func (s Store) runDir(projectPath, id string) string {
	if s.Root != "" {
		return filepath.Join(s.Root, "runs", id)
	}
	return runDir(projectPath, id)
}

func (s Store) runsRoot(projectPath string) string {
	if s.Root != "" {
		return filepath.Join(s.Root, "runs")
	}
	return filepath.Join(projectPath, ".gvardia", "runs")
}

// Dir returns the run directory.
func (r Run) Dir() string {
	if r.MetaPath != "" {
		return filepath.Dir(r.MetaPath)
	}
	if r.ProjectPath == "" || r.ID == "" {
		return ""
	}
	return runDir(r.ProjectPath, r.ID)
}

func fillTelemetryPaths(run *Run) {
	dir := run.Dir()
	if dir == "" {
		return
	}
	if run.StatusPath == "" {
		run.StatusPath = filepath.Join(dir, "status.json")
	}
	if run.EventsPath == "" {
		run.EventsPath = filepath.Join(dir, "events.jsonl")
	}
	if run.ArtifactsPath == "" {
		run.ArtifactsPath = filepath.Join(dir, "artifacts.json")
	}
	if run.ArtifactsDir == "" {
		run.ArtifactsDir = filepath.Join(dir, "artifacts")
	}
}
