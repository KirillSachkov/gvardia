# gvardia v2 — Implementation Plan (monitoring & review)

> Implements `DESIGN-v2.md`. Execute **phase by phase, in order**, committing after
> each phase; each phase's **Acceptance** is a gate. Same conventions as
> `PLAN.md`: Go 1.25+, every phase ends green
> (`go build ./... && go vet ./... && go test ./... -race`), `gofmt` clean,
> table-driven tests against captured fixtures, no I/O in Bubble Tea Update/View.

**Goal:** reframe gvardia around agent work-sessions — one row per session (live +
history), honest process-based liveness, summaries + change stats from logs, task
inferred from branch.

**Global constraints:** Go 1.25 floor; deps = stdlib + Charm v2 (`charm.land/*`) +
BurntSushi/toml + x/sync; adapters degrade gracefully (absent CLI/log ⇒ skipped,
never fatal); history is lazy + bounded + per-selected-project (collection is
disk-bound).

---

## Phase A — Work-session model + task inference

- `internal/model/model.go`: extend `Session` with `Task string`, `Live bool`
  (process-backed), `LastActivity time.Time`, `Summary string`, `WorktreePath
  string`. Add `ChangeStat{ Files, Added, Removed int }` and a `ChangeStat` field
  on `Worktree`. JSON tags with `omitempty`.
- `internal/collect/task.go`: `TaskFromBranch(branch string) string` — regex over
  common shapes: `#123`, `feat/675-...`/`675-...` → `#675`, `AUTH-12-...` → `AUTH-12`.
  Return `""` on no match.

**Tests:** table-driven `TaskFromBranch` (`feat/675-s3`→`#675`, `AUTH-12-login`→
`AUTH-12`, `dev`→``, `fix/quiz-render`→``, `#42-x`→`#42`).

**Acceptance:** model compiles with new fields; `TaskFromBranch` table green.

## Phase B — Honest codex liveness (process-based)

- `internal/adapters/codex.go`: split responsibility. A `codex` **live** session is
  now backed by a running process, not a fresh file.
  - Add `procLister` interface: `LiveCodexCwds(ctx) (map[string]int, error)` →
    cwd→pid for running codex processes. Real impl: `pgrep -x codex` then
    `lsof -a -p <pid> -d cwd -Fn` to resolve each cwd (macOS/Linux). Behind the
    interface so tests fake it.
  - `Codex.Sessions(ctx)`: get live cwds; for each, find the newest session file
    for that cwd (existing walk) → emit a `Session{Harness:"codex", Live:true,
    Status: busy if file mtime fresh else idle, PID, Cwd, SessionID, StartedAt}`.
    No live process for a cwd ⇒ not emitted here (it becomes history in Phase C).
- Keep the file walk/`readCodexMeta` (reused by history).

**Tests:** fake `procLister` returning two cwds; temp session files; assert only
process-backed cwds are emitted, `Live==true`, PID set. Absent `codex`/`lsof`
(lister error) ⇒ `(nil, err)` → skipped.

**Acceptance:** `gvardia agents --json` codex entries reflect **running** codex
processes only (cross-check with `pgrep codex` + `lsof` by hand); stale files no
longer appear as live.

## Phase C — Summaries + history reader

- `internal/history/history.go`: `Recent(ctx, cwd string, opts Options) []model.Session`
  — ended sessions for a cwd from claude + codex logs, newest first, bounded
  (`opts.Limit`, `opts.Since`). Each: `Live:false`, `Summary`, `LastActivity`,
  `SessionID`, `Harness`, `Cwd`.
- `internal/history/claude.go`: walk `~/.claude/projects/*/*.jsonl`; read `cwd`
  from file contents (first line carrying `cwd`); keep those matching the target
  cwd; `firstUserPrompt` = first `type:"user"` message text; `LastActivity` = file
  mtime.
- `internal/history/codex.go`: reuse codex session files; `firstUserPrompt` =
  first `response_item` with `payload.role=="user"` text (skip developer/system);
  `LastActivity` = file mtime.
- `internal/history/summary.go`: shared `firstUserPrompt`/truncation helper +
  `Options{ Limit int; Since time.Duration }`.
- Also expose `SummaryFor(harness, sessionID, cwd) string` so **live** sessions get
  a summary by locating their transcript by sessionID.

**Tests:** fixtures — a captured claude transcript and codex jsonl under a temp
`root`; assert `Recent` returns them for the right cwd with correct summary +
ordering; wrong cwd excluded; `Limit`/`Since` honored; malformed line skipped.

**Acceptance:** for a real project (e.g. `software-engineer-tutorial`), `history`
lists recent past sessions with a plausible task-summary matching the log's first
user prompt.

## Phase D — Per-worktree change stat

- `internal/collect/status.go`: in `Enrich`, also run
  `git -C <wt> diff --numstat <base>...HEAD`; sum into `Worktree.ChangeStat`
  (`Files`, `Added`, `Removed`). Parser `parseNumstat([]byte) ChangeStat` handles
  binary (`-\t-\t`) rows.

**Tests:** `parseNumstat` fixture (`3\t1\tfile.go\n-\t-\tbin.png\n`) →
`{Files:2, Added:3, Removed:1}`.

**Acceptance:** `projects --json` worktrees carry a change stat matching
`git diff --numstat <base>...HEAD | awk` on 2 spot-checked worktrees.

## Phase E — Assemble work-sessions

- `internal/collect/join.go`: replace/extend `Join` with `AssembleLive(projects,
  liveSessions)` — attach live sessions to worktrees **and** produce a per-project
  flat `[]model.Session` (`Project.WorkSessions`), one entry per live session,
  copying `Branch`, `WorktreePath`, `Task` (`TaskFromBranch`), and `ChangeStat`
  from the matched worktree. `LiveAgents` = count of live sessions.
- History merge is done in the UI (lazy, per selected project): a helper
  `MergeHistory(work []Session, hist []Session) []Session` — de-dupe by SessionID
  (a live session already shown is not duplicated from history), live first then
  history by `LastActivity` desc.

**Tests:** `AssembleLive` — one worktree, two live sessions ⇒ two WorkSessions with
branch/task/changestat filled; `MergeHistory` dedups by SessionID and orders
live-before-history.

**Acceptance:** `agents --json` shows per-session rows (not per-worktree) with
task/branch/changestat; the multi-agent-per-worktree collapse is gone.

## Phase F — UI rework (session rows + detail + history)

- `internal/ui/items.go`: right pane rows = **sessions**. Columns: state glyph
  (`●`busy/`○`idle live, `✓`ended) · harness · agent · task · branch · `+A/-R` ·
  last-active (relative). A `sessionRow(model.Session)` builder + `relativeTime`.
- `internal/ui/model.go`/`update.go`: right pane holds the selected project's
  `WorkSessions` (+ merged history when enabled); `selectedSession()` replaces
  `selectedWorktree()` for diff/actions (map session→worktree by `WorktreePath`).
  `h` toggles history: sets a flag and fires a `historyCmd(cwd)` (lazy, cached in
  a `map[string][]Session` on the model); results merged via `MergeHistory`.
- `internal/ui/commands.go`: `historyCmd(project)` runs `history.Recent` in a
  `tea.Cmd`, returns `historyMsg{project, sessions}`.
- `internal/ui/view.go`: bottom detail pane = selected session — `Summary` (task
  text) + change stat + `enter` lazygit; a `─ recent ─` divider between live and
  history rows. Footer gains `h history`.
- Actions (`a/r/k/g`) operate on `selectedSession()` (ended sessions: `a`/`r`
  resume by SessionID still valid; `k` disabled for ended — no PID).

**Tests (message-driven, package ui):** fleetMsg with WorkSessions → rows built,
one per session; `h` sets history flag + issues a cmd; `historyMsg` merges + adds
ended rows; nav selects a session and issues a diff for its worktree; ended
session `k` no-ops (no confirm).

**Acceptance (PTY smoke):** cockpit shows one row per live agent (fixes the
screenshot bug), the detail pane shows the session's task-summary + change stat,
`h` reveals recent ended sessions, `enter` opens lazygit at the right worktree,
`q` exits clean.

## Phase G — CLI JSON + end-to-end verify

- `cmd/gvardia/agents.go`: JSON now includes per-project `workSessions` with the
  new fields (task, summary, live, lastActivity, changeStat). Human view lists one
  line per live session with task + change stat.
- Update `README.md` (keybinds: `h history`; note live-vs-history + task column).

**Acceptance:** `agents --json` on `~/code` shows honest live agents with
summaries + tasks; cross-check the running claude/codex count against
`claude agents --json` + `pgrep codex`; full gate green; PTY smoke of the new
cockpit clean.

---

## Testing strategy

Parsers (task regex, numstat, claude/codex summary) → table-driven against
captured fixtures. Process liveness + history behind small interfaces so tests
fake them. UI flows → message-driven `Update` tests (as in v1). One real-git +
real-log integration where cheap. Run `-race`.

## Out of scope (later layers)

Task-from-brain (`sachkov-os/tasks`) as the single task source; embedded
terminals; notifications; LLM summaries. All graft onto the work-session unit.
