# gvardia — Design

A terminal cockpit over a fleet of coding agents, across all projects,
agent-agnostic. This document is the source of truth for architecture and scope.
The build plan lives in `PLAN.md`; future directions in `ROADMAP.md`.

## 1. Problem & principles

**Problem.** Running many agents in parallel is solved (harness built-ins +
worktrees). Seeing the *whole fleet across projects* in one terminal surface —
statuses, changes, and a fast path to review/attach — is not. Existing tools are
single-repo (workmux, claude-squad), GUI (Orca, jean), or single-harness
(`claude agents`).

**Principles.**
1. **Thin router, not a monolith.** Never reimplement git, diff-viewing, or an
   orchestrator. Aggregate state; shell out to proven tools for the heavy lifting
   (`lazygit`, `delta`, `tmux`, the agent CLIs).
2. **Agent-agnostic core.** Source of truth = `git` (worktrees, branches,
   commits) + `tmux` + tracker. Per-harness knowledge lives only in **adapters**.
3. **Cross-project first.** The top-level object is the *project list*; a fleet
   spans every configured root, not one repo.
4. **YAGNI.** v1 does exactly: see projects × sessions × statuses × diff, and act
   (attach / new / resume / kill / gc). Everything else is `ROADMAP.md`.
5. **Read-mostly, cheap, restartable.** All reads are re-derivable from git +
   process state; the tool holds no durable state of its own beyond config.

## 2. Tech choice

**Go + Bubble Tea (Charm).** Decided on pure technical fit (language familiarity
explicitly excluded as a factor):

- **Domain prior-art is all Go** — lazygit, k9s, gwq, claude-squad, ccmanager,
  gh, glab. Directly readable/liftable for a worktree/fleet TUI.
- **Concurrent shell-out** is the core loop (dozens of `git`/`tmux`/adapter probes
  per refresh across all repos) — goroutines + `errgroup` + channels fit natively.
- **Single static ~2MB binary**, zero runtime — best "share with anyone / teach".
- **Bubbles** components (`list`, `table`, `viewport`) are a ready k9s-style
  caucus; Bubble Tea v2 + Mode 2026 render dense tables without tearing.

Alternative considered: TypeScript + Ink + Bun. Wins only on "same stack as Claude
Code / study its source" — not a v1 goal. If ever re-chosen, the architecture,
data model, adapter interface, and PLAN phases below are language-agnostic; only
the code stubs change.

## 3. Architecture

```
        ┌────────────────────────── gvardia ──────────────────────────┐
config ─┤ collectors (concurrent)          adapters (per-harness)      │
~/.config│  worktrees ← git worktree list    claude  ← claude agents --json
         │  gitstatus ← git status/rev-list  codex   ← ~/.codex/sessions/*.jsonl
         │  (per root, per worktree)         tmux    ← tmux list-panes (liveness)
         │            │                              │                 │
         │            └──────────► model (Project/Worktree/Session) ◄──┘
         │                                   │                          │
         │                              Bubble Tea UI (3-pane cockpit)  │
         │                                   │                          │
         │  actions shell out ──► lazygit · delta · tmux · claude/codex · wt-prune
        └───────────────────────────────────────────────────────────────┘
```

Layers (map to `internal/` packages):

- **config** — `~/.config/gvardia/config.toml`: `roots` (dirs to scan, default
  `["~/code"]`), per-project `base` branch (default detect `dev` else `main`),
  enabled `adapters`, `refresh_interval`, external commands (`lazygit`, editor).
- **model** — plain types, no I/O:
  - `Project{ Name, Path, Worktrees[], LiveAgents int }`
  - `Worktree{ Path, Branch, IsPrimary, Dirty bool, Ahead, Behind int, LastCommit time.Time, BaseBranch }`
  - `Session{ Harness (claude|codex|…), Name, PID int, Cwd, Branch, Status (busy|idle|failed|unknown), StartedAt }`
- **collect** — concurrent, cached, refreshable:
  - `Worktrees(root)` → `git -C <repo> worktree list --porcelain` for every repo
    under each root.
  - `Enrich(wt)` → `git -C <wt> status --porcelain=v2 --branch` (dirty, ahead,
    behind) + last-commit time. Fan-out with `errgroup`, bounded concurrency.
- **adapters** — the agent-agnostic seam (see `ADAPTERS.md`):
  ```go
  type Adapter interface {
      Name() string                    // "claude", "codex", …
      Sessions(ctx context.Context) ([]model.Session, error)
  }
  ```
  - `claude`: exec `claude agents --all --json`, unmarshal (`cwd`, `status`,
    `name`, `pid`, `sessionId`).
  - `codex`: walk `~/.codex/sessions/**/*.jsonl`, newest per cwd; status = busy if
    its PID is alive / file mtime is fresh, else idle/unknown.
  - `tmux` (optional signal): `tmux list-panes -a -F '#{pane_current_path} #{pane_pid} #{pane_current_command}'`
    to mark worktrees with a live agent pane even without a harness adapter.
  - **Join**: `Session` ↔ `Worktree` by `Cwd`/`Branch`.
- **ui** — Bubble Tea `Model{ projects, selProject, sessions, selSession,
  diffPreview, focus, filter }`; `Update` handles `tea.KeyMsg`, `tickMsg`
  (interval refresh), and async collector/adapter result msgs; `View` renders the
  three panes + footer via Lipgloss/Bubbles.

## 4. TUI spec

Three panes + footer (see README mockup):

- **Projects (left):** `list` of projects under roots, each showing `N wt · M live`.
  `↑↓`/`j k` select; selection drives the sessions pane.
- **Sessions (right-top):** `table` of the selected project's worktrees/sessions:
  status glyph (● busy / ○ idle / ◍ codex / ✖ failed), harness, name, branch,
  `+add -del`. A trailing line summarizes worktrees **without** an agent
  (`N merged · M stale`).
- **Diff (bottom):** `viewport` with `git -C <wt> diff --stat <base>...HEAD` for the
  selected session. `Enter` drills in.
- **Footer:** keybind hints.

**Keybinds (v1 = read-only + actions):**

| Key | Action | Mechanic |
|---|---|---|
| `↑↓ j k` | navigate | — |
| `tab` | switch pane focus | — |
| `/` | filter (project/branch) | Bubbles textinput |
| `enter` | review diff | suspend TUI → `lazygit -p <wt>` (delta as pager) → resume |
| `a` | attach | `tmux attach -t …` / `claude --resume <id>` / `codex resume` |
| `n` | new agent | harness picker → `claude -w <name> --tmux` (or spawn in wt) |
| `r` | resume session | harness resume command |
| `k` | kill | `SIGTERM` the session PID; drops off next refresh |
| `g` | worktree gc | run `tools/wt-prune` for current root |
| `R` | force refresh | re-run collectors/adapters now |
| `q` | quit | — |

Suspend/resume pattern for `enter`/`a`: use `tea.Exec` (Bubble Tea) to hand the
terminal to the child process and return cleanly.

## 5. Configuration

`~/.config/gvardia/config.toml`:

```toml
roots = ["~/code"]              # dirs scanned for git repos (recursively, shallow)
refresh_interval = "5s"
adapters = ["claude", "codex", "tmux"]

[base]                          # base branch per project for diff/ahead-behind
default = "auto"               # auto = dev if exists else main
"education-platform" = "dev"

[commands]
lazygit = "lazygit"
```

## 6. Non-goals (v1)

- No orchestration / inter-agent messaging (commodity: `claude` agent teams).
- No custom diff/merge UI (delegate to lazygit).
- No cloud/remote agents, no web UI, no kanban board (see ROADMAP).
- No MR/CI enrichment yet (ROADMAP: `gh`/`glab` join).

## 7. Distribution

`go build ./cmd/gvardia` → single binary. `goreleaser` for cross-platform release
+ a Homebrew tap. No runtime deps beyond the external CLIs it shells out to
(degrade gracefully if `lazygit`/`tmux`/an agent CLI is absent).
