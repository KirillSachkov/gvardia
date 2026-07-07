# AGENTS.md — context for coding-agent sessions

This repo is **spec-first**: the design and plan are complete; the tool is not yet
built. If you are an agent (Claude Code, Codex, …) dispatched to work here, read
in this order:

1. `docs/DESIGN.md` — architecture, scope, tech choice. Source of truth.
2. `docs/PLAN.md` — implement **phase by phase, in order**; each phase's
   *Acceptance* is a gate. Commit after each phase.
3. `docs/ROADMAP.md` — do **not** build these yet; they are post-v1.

## What gvardia is

A terminal cockpit (TUI) over a fleet of coding agents across all projects.
Agent-agnostic (git + tmux core, pluggable per-harness adapters). Thin router —
it aggregates state and shells out to `lazygit`/`delta`/`tmux`/agent CLIs. It is
**not** an orchestrator and must not become one.

## Stack & conventions

- **Go 1.23+**, module `github.com/<you>/gvardia`. TUI: Bubble Tea v2 + Lipgloss +
  Bubbles. TOML config.
- Layout: `cmd/gvardia`, `internal/{config,model,collect,adapters,ui}`,
  `tools/wt-prune`, `docs/`.
- Keep packages single-responsibility (see `DESIGN.md#3`), functions small.
- External CLIs behind small interfaces so tests can fake them.
- Every phase ends green: `go build ./... && go vet ./... && go test ./...`.
- Degrade gracefully when an external CLI (`lazygit`, `tmux`, an agent CLI) is
  absent — skip with a banner, never crash.

## Guardrails

- Respect the v1 non-goals in `DESIGN.md#6`. Resist scope creep into
  orchestration, custom diff UIs, web/cloud, or kanban — those are ROADMAP.
- Destructive actions (`kill`, worktree `gc`) must confirm and never touch a
  primary or dirty worktree.
- Don't add heavy deps; prefer stdlib + the Charm stack + a TOML lib.

## How to start

Begin at **Phase 0** in `docs/PLAN.md`. Ask before deviating from the phase order.
