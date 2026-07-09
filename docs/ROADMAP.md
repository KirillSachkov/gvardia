# gvardia — Roadmap (post-v1)

v1 (see `PLAN.md`) is deliberately minimal: projects × sessions × statuses × diff,
plus attach/new/resume/kill/gc. These are the growth directions once v1 is solid.
Each is independent — pull the next most useful, don't build ahead of need.

## Shipped after v1

- A flat global queue of managed runs across projects, sorted by attention.
- Standalone Markdown tasks and JSON/JSONL run evidence under the XDG data dir.
- Reports and typed artifacts with a selectable artifact browser.
- tmux-backed launches with cmux presentation and liveness reconciliation.

## Near-term

- **Live-watch mode** — push-based refresh instead of polling: watch `~/.claude`
  / `~/.codex` session dirs (fsnotify) + `git` HEAD refs, so status updates are
  near-instant. Keep the poll as a fallback.
- **Tracker enrichment (MR/CI)** — per branch, join open MR + pipeline status via
  `gh` / `glab` (cached, lazy). New column: `MR#123 ✓ / ✗ / ⏳`. Answers "which
  agent's work is ready / failed validation."
- **Cross-repo scan parity** (idea from `gwq`) — treat worktrees globally, not
  per-project; `gvardia status --json/--csv` as a machine feed others can pipe.

## Provenance / reports

- **Review runs** — launch a second bounded run against the first run's report and
  diff, then link both runs to the same task without automatic orchestration.
- **Commit-trailer awareness** — parse `Assisted-by:` / `Co-Authored-By:` trailers
  to attribute worktrees to the agent/model that produced them.

## Bigger bets

- **Kanban-per-worktree view** (idea from Vibe Kanban) — optional board layout:
  columns todo/doing/review/done mapped to branch state, drag = dispatch/merge.
- **Remote / multi-machine fleet** (idea from claude-fleet) — adapters over SSH so
  the cockpit spans machines, not just local `~/code`.
- **More adapters** — `aider`, `opencode`, `gemini`, `goose` (see `ADAPTERS.md`).
- **Cost/token roll-up** — consume Claude Code / Gemini native OpenTelemetry to
  add per-session/project token+cost; ship a Grafana panel recipe.
- **Session peek** — inline last-N lines of an agent's transcript in a preview
  pane (`space`), parsing the harness JSONL.

## Explicitly out of scope (commodity — don't rebuild)

Orchestration, inter-agent messaging, the agent runtime itself, worktree
isolation, per-harness single-repo status — all provided by the harnesses.
gvardia stays the thin cross-project observability + routing layer.
