// Package model holds gvardia's plain data types. It performs no I/O and has no
// dependencies beyond the standard library, so every layer can share it freely.
package model

import "time"

// Project is a git repository under a configured root, with all its worktrees.
type Project struct {
	Name         string     `json:"name"`
	Path         string     `json:"path"`
	Worktrees    []Worktree `json:"worktrees"`
	LiveAgents   int        `json:"liveAgents"`
	WorkSessions []Session  `json:"workSessions,omitempty"` // flat, live-first session list
}

// Worktree is a single git worktree (the primary checkout or a linked one) plus
// its status relative to its base branch. The zero value is a valid, empty
// worktree. Sessions are the agent sessions joined to this worktree (Phase 2).
type Worktree struct {
	Path       string     `json:"path"`
	Branch     string     `json:"branch"` // empty when detached
	IsPrimary  bool       `json:"isPrimary"`
	Dirty      bool       `json:"dirty"`
	Ahead      int        `json:"ahead"`
	Behind     int        `json:"behind"`
	LastCommit time.Time  `json:"lastCommit"`
	BaseBranch string     `json:"baseBranch"`
	ChangeStat ChangeStat `json:"changeStat"`
	Sessions   []Session  `json:"sessions,omitempty"`
}

// ChangeStat summarizes a worktree's diff against its base branch.
type ChangeStat struct {
	Files   int `json:"files"`
	Added   int `json:"added"`
	Removed int `json:"removed"`
}

// Task is one card from the personal kanban (sachkov-os/tasks). Status is the
// kanban column, taken from the containing directory (inbox/active/done).
type Task struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	Project string `json:"project,omitempty"`
	Path    string `json:"path"`
}

// Status is an agent session's coarse state, derived conservatively per adapter.
type Status string

// Session states. Unknown is the zero value: use it when a state can't be told.
const (
	StatusUnknown Status = "unknown"
	StatusBusy    Status = "busy"
	StatusIdle    Status = "idle"
	StatusFailed  Status = "failed"
)

// Session is a harness-neutral agent session, reported by an adapter and joined
// to a Worktree by Cwd. Fields an adapter can't determine keep their zero value.
type Session struct {
	Harness      string     `json:"harness"` // adapter Name(): "claude", "codex", …
	Name         string     `json:"name"`
	SessionID    string     `json:"sessionId,omitempty"` // harness resume key (empty for tmux)
	Task         string     `json:"task,omitempty"`      // inferred from branch, e.g. "#675"
	PID          int        `json:"pid"`
	Cwd          string     `json:"cwd"`
	Branch       string     `json:"branch"`
	WorktreePath string     `json:"worktreePath,omitempty"`
	Live         bool       `json:"live"` // backed by a running process (else history)
	Status       Status     `json:"status"`
	StartedAt    time.Time  `json:"startedAt"`
	LastActivity time.Time  `json:"lastActivity,omitempty"`
	Summary      string     `json:"summary,omitempty"`    // first user prompt from the transcript
	ChangeStat   ChangeStat `json:"changeStat,omitempty"` // copied from the joined worktree
}
