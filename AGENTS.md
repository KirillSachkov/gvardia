# AGENTS.md — context for coding-agent sessions

Status 2026-07-18: the tool IS built (branch `v4`, released binaries in `dist/`,
`cmd/` + `internal/` implemented); the wider product direction is parked (see
sachkov-os `wiki/gvardia-agent-work-observability.md`). If you are an agent
(Claude Code, Codex, …) dispatched to work here, read in this order:

1. `docs/DESIGN-v4-hub.md` — current architecture (older `DESIGN*.md` are history).
2. `docs/GO-CONVENTIONS.md` — how to write Go + Bubble Tea here. **Read before
   writing code** — it exists to prevent "C#-in-Go".
3. `docs/PLAN-v4-hub.md` + `docs/HANDOFF-v4.md` — current plan state; older
   `PLAN*.md` are superseded.
4. `docs/ROADMAP.md` — do **not** build these; direction is parked anyway.

## What gvardia is

A terminal cockpit (TUI) over a fleet of coding agents across all projects.
Agent-agnostic (git + tmux core, pluggable per-harness adapters). Thin router —
it aggregates state and shells out to `lazygit`/`delta`/`tmux`/agent CLIs. It is
**not** an orchestrator and must not become one.

## Stack & conventions

- **Go 1.25** (see `go.mod`), module `github.com/<you>/gvardia`. TUI: Bubble Tea v2 + Lipgloss +
  Bubbles. TOML config.
- Layout: `cmd/gvardia`, `internal/{config,model,collect,adapters,ui}`,
  `tools/wt-prune`, `docs/`.
- Keep packages single-responsibility (see `DESIGN.md#3`), functions small.
- External CLIs behind small interfaces so tests can fake them.
- Every phase ends green: `go build ./... && go vet ./... && go test ./...`.
- Degrade gracefully when an external CLI (`lazygit`, `tmux`, an agent CLI) is
  absent — skip with a banner, never crash.
- Follow `docs/GO-CONVENTIONS.md`: errors as values, `context` first, no I/O in
  Bubble Tea `Update`/`View` (I/O → `tea.Cmd`), shell out via `tea.ExecProcess`,
  table-driven tests, `gofmt`/`golangci-lint` clean.

## Guardrails

- Respect the v1 non-goals in `DESIGN.md#6`. Resist scope creep into
  orchestration, custom diff UIs, web/cloud, or kanban — those are ROADMAP.
- Destructive actions (`kill`, worktree `gc`) must confirm and never touch a
  primary or dirty worktree.
- Don't add heavy deps; prefer stdlib + the Charm stack + a TOML lib.

## How to start

Begin at **Phase 0** in `docs/PLAN.md`. Ask before deviating from the phase order.
