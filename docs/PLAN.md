# gvardia — Implementation Plan

Execution-ready, phase by phase. A future coding-agent session should implement
these **in order**, committing after each phase, and treating each phase's
**Acceptance** as a gate before moving on. Read `DESIGN.md` first.

Conventions: Go 1.23+, module `github.com/<you>/gvardia`. Every phase ends green
(`go build ./... && go vet ./... && go test ./...`). Keep functions small and one
package per responsibility per `DESIGN.md#3`.

---

## Phase 0 — Bootstrap

- `go mod init github.com/<you>/gvardia`; add deps: `bubbletea`, `lipgloss`,
  `bubbles`, a TOML lib (`github.com/BurntSushi/toml`), `golang.org/x/sync/errgroup`.
- Layout per `DESIGN.md#3`: `cmd/gvardia/main.go`, `internal/{config,model,collect,adapters,ui}`.
- `internal/config`: load `~/.config/gvardia/config.toml` with defaults
  (`roots=["~/code"]`, `refresh_interval=5s`, `adapters=["claude","codex","tmux"]`);
  expand `~`. Unit-test defaults + override merge.
- `cmd/gvardia`: cobra/std-flag CLI with `--version`, `--config`, and a hidden
  `agents --json` subcommand stub (filled in Phase 2).
- CI: GitHub Actions `go build/vet/test` on push.

**Acceptance:** `gvardia --version` prints; missing config loads defaults; test
suite green.

## Phase 1 — Worktree + git-status collectors (read-only core)

- `internal/collect/worktrees.go`: for each `root`, discover git repos (dir with
  `.git`), then `git -C <repo> worktree list --porcelain` → `[]model.Worktree`
  (path, branch, isPrimary).
- `internal/collect/status.go`: `Enrich(wt)` via
  `git -C <wt> status --porcelain=v2 --branch` → dirty, ahead, behind vs upstream;
  last-commit time via `git -C <wt> log -1 --format=%ct`. Resolve `BaseBranch`
  from config (`auto` = dev else main).
- Fan-out with `errgroup` + a bounded semaphore (e.g. `min(16, NumCPU*2)`); tolerate
  per-repo errors (skip, don't fail the batch).
- Wire a real subcommand `gvardia projects --json` that dumps the collected model.

**Acceptance:** `gvardia projects --json` on `~/code` lists every project with its
worktrees, correct dirty/ahead/behind, matching `git worktree list` +
`git status` run by hand on 2–3 spot-checked repos. Runs in < ~1s on the current
machine (≈90 worktrees).

## Phase 2 — Adapters (agent status) + join

- `internal/adapters/adapter.go`: the `Adapter` interface (`DESIGN.md#3`) +
  registry keyed by name.
- `internal/adapters/claude.go`: exec `claude agents --all --json`, unmarshal to
  `[]model.Session` (`cwd`, `status`, `name`, `pid`, `sessionId`→StartedAt).
- `internal/adapters/codex.go`: walk `~/.codex/sessions/**/*.jsonl`; newest per
  `cwd`; status busy if PID alive or mtime within `refresh_interval*3`, else idle.
- `internal/adapters/tmux.go`: `tmux list-panes -a -F '…'`; mark worktrees whose
  `pane_current_path` is inside a worktree and command looks like an agent.
- `internal/collect/join.go`: attach `Session`s to `Worktree`s by cwd/branch; set
  `Project.LiveAgents`.
- Fill `gvardia agents --json` (headless fleet dump).

**Acceptance:** `gvardia agents --json` shows the real live Claude sessions (cross-
check with `claude agents --json`) **and** recent Codex sessions, each joined to
the right project/worktree with a plausible status. Absent CLI ⇒ adapter skipped,
no crash.

## Phase 3 — TUI (read-only)

- `internal/ui/model.go|update.go|view.go`: Bubble Tea 3-pane (projects `list`,
  sessions `table`, diff `viewport`) + footer, per `DESIGN.md#4`.
- Data flow: on start and every `refresh_interval` (tickMsg) run collectors+adapters
  in a `tea.Cmd`, feed results as msgs; `R` forces refresh; `/` filters.
- Diff pane: `git -C <wt> diff --stat <base>...HEAD` for the selected session.
- Graceful states: loading, empty, adapter-error banner.

**Acceptance:** `gvardia` launches into the cockpit; navigating projects updates
the sessions table; selecting a session shows its diff stat; statuses match
`agents --json`; refresh is flicker-free (Mode 2026) and < ~300ms perceived.

## Phase 4 — Diff drill-down (shell-out)

- `enter` on a session: `tea.Exec` → `lazygit -p <worktree>` (ensure delta is the
  configured pager), return to TUI cleanly on exit.
- Fallback if `lazygit` absent: `git -C <wt> -c core.pager=delta diff <base>...HEAD`.

**Acceptance:** `enter` opens lazygit rooted at the correct worktree; quitting
lazygit restores the TUI with terminal state intact.

## Phase 5 — Actions (v1 management surface)

- `a` attach: claude → `claude --resume <sessionId>`; codex → `codex resume`;
  else `tmux attach` to the matching pane. Via `tea.Exec`.
- `n` new agent: small harness picker (claude/codex) + name prompt → for claude
  `claude -w <name> --tmux` in the selected project; generic path spawns the agent
  CLI in a fresh `git worktree`.
- `r` resume (selected session's harness resume), `k` kill (`SIGTERM` PID, confirm
  modal), `g` gc → run `tools/wt-prune` for the current root then refresh.

**Acceptance:** each action performs against a real session; `k` removes the
session on next refresh; `n` produces a new worktree+session that appears in the
cockpit; destructive actions (`k`, `g`) confirm first.

## Phase 6 — wt-prune polish

- Promote `tools/wt-prune` (starter bash already in repo) or reimplement in Go as
  `cmd/wt-prune`: list worktrees across all roots classified merged / stale
  (no commits in N days) / active; remove merged+stale on confirm; never touch
  primary or dirty worktrees.

**Acceptance:** dry-run lists correctly on `~/code`; `--yes` removes only
merged/stale; primary/dirty worktrees always preserved.

## Phase 7 — Distribution & docs

- `goreleaser` config (darwin/linux arm64+amd64), Homebrew tap, `go install` path.
- README: real asciinema/screenshot; document config + keybinds; degrade-gracefully
  notes. Tag `v0.1.0`.

**Acceptance:** `go install …@latest` and a released binary both launch the
cockpit on a clean machine; README quickstart works end-to-end.

---

## Milestones

- **M1 (Phases 0–2):** headless fleet model — `gvardia agents --json` is useful on
  its own (scriptable cross-project fleet feed).
- **M2 (Phases 3–4):** the cockpit — see + review. This is the "one tool" win.
- **M3 (Phases 5–7):** manage + ship — attach/new/kill/gc, released binary.

## Testing strategy

- Collectors/adapters: unit-test parsers against fixture `git`/`--json`/JSONL
  outputs (no live git needed). One integration test that runs against a temp repo
  with a couple of worktrees.
- UI: `teatest` (Bubble Tea's test harness) for key flows (nav, filter, refresh msg).
- Keep external-CLI calls behind small interfaces so they can be faked in tests.
