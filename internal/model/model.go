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
// worktree.
type Worktree struct {
	Path       string    `json:"path"`
	Branch     string    `json:"branch"` // empty when detached
	IsPrimary  bool      `json:"isPrimary"`
	Dirty      bool      `json:"dirty"`
	Ahead      int       `json:"ahead"`
	Behind     int       `json:"behind"`
	LastCommit time.Time `json:"lastCommit"`
	BaseBranch string    `json:"baseBranch"`
}
