// Package model holds gvardia's plain data types. It performs no I/O and has no
// dependencies beyond the standard library, so every layer can share it freely.
package model

import "time"

// Project is a git repository under a configured root, with all its worktrees.
type Project struct {
	Name       string     `json:"name"`
	Path       string     `json:"path"`
	Worktrees  []Worktree `json:"worktrees"`
	LiveAgents int        `json:"liveAgents"`
}

// Worktree is a single git worktree (the primary checkout or a linked one) plus
// its status relative to its base branch. The zero value is a valid, empty
// worktree. Sessions are the agent sessions joined to this worktree (Phase 2).
type Worktree struct {
	Path       string    `json:"path"`
	Branch     string    `json:"branch"` // empty when detached
	IsPrimary  bool      `json:"isPrimary"`
	Dirty      bool      `json:"dirty"`
	Ahead      int       `json:"ahead"`
	Behind     int       `json:"behind"`
	LastCommit time.Time `json:"lastCommit"`
	BaseBranch string    `json:"baseBranch"`
	Sessions   []Session `json:"sessions,omitempty"`
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
	Harness   string    `json:"harness"` // adapter Name(): "claude", "codex", …
	Name      string    `json:"name"`
	PID       int       `json:"pid"`
	Cwd       string    `json:"cwd"`
	Branch    string    `json:"branch"`
	Status    Status    `json:"status"`
	StartedAt time.Time `json:"startedAt"`
}
