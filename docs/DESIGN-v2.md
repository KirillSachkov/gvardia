# gvardia v2 — Design (monitoring & review)

Extends v1 (`DESIGN.md`). v1 shipped the read-only cockpit + actions + wt-prune.
v2 reframes gvardia around **monitoring and reviewing agent work across all
projects**: what is running, what ran, what each agent did, on which task /
branch / worktree, and what changed. It is a monitor/review tool, **not** a
multiplexer (that niche is filled by herdr) and **not** an orchestrator.

## 1. Why (the gap)

Juggling many projects, it is hard to track what work is happening where: which
agents run now, what they did, on which branch, for which task. Live
multiplexers (herdr) drive agents but keep no history, no git, no diffs. gvardia
is the complement: git-grounded, cross-project, retrospective.

## 2. The unit: a work-session

The central object is an **agent session as a unit of work**, live or historical:

```
Session{
  Harness, Name, SessionID,
  Task,                 // inferred from branch ("feat/675-s3" -> "#675"); "" if none
  Branch, WorktreePath,
  State,                // running(busy|idle) | ended
  StartedAt, LastActivity,
  Summary,              // first user prompt from the transcript (the "task text")
  ChangeStat,           // files, +N/-M vs base branch
}
```

Rows in the UI are sessions (one per session, **not** per worktree — v1 collapsed
multiple agents on one worktree into a single row; that is the bug this fixes).

## 3. Honest liveness

A session's `State` reflects a real process, not a stale file:

- **claude** — `claude agents --json` already reports live processes with
  status/pid. Keep as-is.
- **codex** — v1 guessed "busy" from session-file mtime, which lies (a month-old
  file looked live). v2 detects a real process: `pgrep codex` + `lsof -a -p <pid>
  -d cwd` to resolve each running codex's cwd, matched to a worktree. No matching
  process => the session is `ended` (history), not live.

"Waiting for input" is **not** a v2 goal: neither harness exposes it reliably.
States are `busy` / `idle` (live) and `ended` (historical).

## 4. History & summaries (from persisted logs)

Agent logs persist on disk and carry the whole conversation:

- **claude** — `~/.claude/projects/<encoded-cwd>/*.jsonl`. Read `cwd` from the
  file contents (not the dir name, whose dash-encoding is ambiguous). Each file
  is a past session.
- **codex** — `~/.codex/sessions/**/*.jsonl` (already parsed in v1 for the
  header). Events: `session_meta`, `response_item`, `event_msg`, `turn_context`.

From a log we extract:
- **Summary** = the first user prompt (skip developer/system messages: for codex,
  the first `response_item` with `role: user`).
- **LastActivity** = file mtime (or last event timestamp).

**Performance.** History is expensive (many files) and v1 collection is already
disk-bound (~4.5s on ~/code). So history loads **lazily for the selected project
only**, bounded (recent N sessions / last K days), and cached. The live overview
stays cheap; history is fetched when a project is focused.

## 5. Task inference (this iteration)

`Task` is inferred cheaply from the branch name via regex (`feat/675-*`,
`AUTH-12-*`, `#675`, etc.) — no external API. Shown as a column and in the detail
pane. Empty when nothing matches.

**Future (next layer, not this iteration):** the single source of tasks will be
the **brain (`sachkov-os/tasks`)**, not GitLab — one canonical task store. gvardia
will read those tasks and join them to sessions/branches, so the board shows
"task X -> agent Y on branch Z, status, diff". Designed to graft onto the same
work-session unit; out of scope here.

## 6. Screen

```
┌ PROJECTS (live first) ─┬ WORK · education-platform ───────────────────────────┐
│▸education-platform 3●  │  st   agent            task    branch          Δ  last │
│  se-tutorial      2●   │  ● busy claude edu-85  #675  feat/675-s3   +90/-12 2m │  live
│  senior-ticker    1○   │  ● busy claude edu-18  #712  epic/pr-dialog +412/-8 5m│
│  OpenTicker       0    │  ○ idle claude edu-da   —    dev                 ·  1h │
│  … (0 live)            │  ─ recent ─────────────────────────────────────────── │
│                        │  ✓ ended codex ab12    #649  fix/quiz-render  +30 3h  │  history
├────────────────────────┴──────────────────────────────────────────────────────┤
│ DETAIL · edu-85 · #675 · feat/675-s3                                           │
│ task: "Finish OpenIddict snake_case + review fixes for s3"                      │
│ 14 files · +90 -12 · last active 2m ago                        [enter] lazygit  │
├────────────────────────────────────────────────────────────────────────────────┤
│ ↑↓ nav · tab · enter lazygit · h history · / filter · a attach · R · q          │
└────────────────────────────────────────────────────────────────────────────────┘
```

- Right pane: one row per session, live first, then a "recent" (history) section.
  Columns: state · harness · agent · **task** · branch · Δ · last-active.
- Bottom: detail of the selected session — the summary (task text from the log),
  change stat; `enter` opens lazygit at its worktree.
- `h` toggles including history in the right pane (default: live + a few recent).

## 7. Kept from v1 (not broken)

Projects list, actions (`a/r/k/g/n` with confirms), `wt-prune`, config,
degrade-gracefully, `agents`/`projects --json`. v2 adds; it does not remove.

## 8. Non-goals (this iteration)

Embedded terminals / workspace (herdr), external trackers (GitLab), LLM-generated
summaries, notifications. These are later layers on the same work-session unit.
