# gvardia — Adapters (the agent-agnostic seam)

gvardia's core knows only `git` + `tmux`. Everything harness-specific lives behind
one small interface. To support a new agent, add one file — nothing else changes.

## The interface

```go
// internal/adapters/adapter.go
type Adapter interface {
    Name() string                                    // "claude", "codex", "aider", …
    Sessions(ctx context.Context) ([]model.Session, error)
}
```

A `Session` is harness-neutral:

```go
type Session struct {
    Harness   string    // adapter Name()
    Name      string    // display name (may be empty)
    PID       int       // 0 if unknown
    Cwd       string    // used to join to a Worktree
    Branch    string    // optional
    Status    Status    // Busy | Idle | Failed | Unknown
    StartedAt time.Time
}
```

The registry (`adapters.Register(a)`) collects enabled adapters; the collector runs
them concurrently and merges results. Any adapter that errors or whose CLI is
absent is skipped with a banner, never fatal.

## Writing an adapter — checklist

1. **Find the source of truth** the harness already writes:
   - a status command with JSON (`claude agents --all --json`), or
   - session files on disk (`~/.codex/sessions/**/*.jsonl`), or
   - process/tmux inspection.
2. **Map to `[]model.Session`.** Derive `Cwd` (critical — it joins to worktrees),
   `Status`, and whatever of `Name/PID/Branch/StartedAt` is available.
3. **Derive status conservatively.** Prefer explicit state; else "busy if PID alive
   or file mtime fresh, idle otherwise, unknown if you can't tell."
4. **Degrade gracefully.** CLI missing / parse fails ⇒ return `(nil, err)`; the core
   skips you. Never panic, never block the UI (respect `ctx`).
5. **Register** in the adapter registry and document the config key.

## Reference adapters (v1)

| Adapter | Source | Status derivation |
|---|---|---|
| `claude` | `claude agents --all --json` | from JSON `status` field |
| `codex`  | newest `~/.codex/sessions/**/*.jsonl` per cwd | PID alive / mtime fresh → busy |
| `tmux`   | `tmux list-panes -a -F '…'` | pane alive in a worktree → busy (fallback signal) |

## Planned adapters (ROADMAP)

`aider` (chat-history + process), `opencode` (`opencode serve` HTTP API),
`gemini` (native OTel / session logs), `goose` (session logs). Each is one file
against the interface above.

## Design note

Because the join key is `Cwd` (a filesystem path) and worktrees are the universal
substrate, **any** agent that runs inside a git worktree shows up in gvardia the
moment an adapter can report its cwd — even before a first-class adapter exists,
the `tmux` adapter catches it as a live pane. That is what makes the tool work
"with any agent."
