# gvardia

**Command your fleet of coding agents from the terminal.**

`gvardia` is a terminal cockpit (TUI) that gives you one screen over every coding
agent working across **all** your projects — which projects are in flight, which
agents are running (Claude Code, Codex, …), what they changed, and their status —
without juggling ten different commands.

It is **agent-agnostic** by design: the source of truth is `git` + `tmux`, and
per-harness status comes from **pluggable adapters** (`claude`, `codex`, …). Works
with any agent that runs in a git worktree and leaves commits behind.

> Status: **design + plan complete, implementation not started.** This repo
> currently contains the full specification (`docs/DESIGN.md`) and an
> execution-ready plan (`docs/PLAN.md`). Point a coding-agent session at it and
> build phase by phase.

---

## Why this exists

Running many agents in parallel is now commodity (Claude Code `claude agents`,
Codex cloud, worktree isolation, etc.). What **no** tool gives you out of the box
is a **cross-project, agent-agnostic, terminal** overview: a single cockpit that
answers "what is my whole fleet doing right now, across every repo, and let me
drill into any diff or attach to any agent."

`gvardia` is that thin layer. It does **not** reimplement git, diffs, or an
orchestrator — it aggregates and routes, shelling out to the best proven tools
(`lazygit`, `delta`, `tmux`, the agent CLIs).

## What it looks like

```
┌ gvardia ─────────────────────────────────────────────── ~/code · 4 agents live ┐
│ PROJECTS             │ SESSIONS · education-platform                            │
│▸education-platform   │ ● busy  claude  education-platform-18  epic/pr-dialogue  +412 -88│
│  84 wt · 3 live ●    │ ● busy  claude  education-platform-85  feat/675-s3       +90 -12 │
│ sharp.arena          │ ◍ busy  codex   rollout·ab12          fix/quiz-render    +30 -4  │
│  5 wt · 0 live       │ ○ idle  claude  education-platform-da  dev               ·       │
│ telegram-bot-flow    │                                                          │
│  2 wt · 0 live       │ worktrees без агента: 47 merged · 12 stale       [g] gc  │
├──────────────────────┴──────────────────────────────────────────────────────┤
│ DIFF · education-platform-18 · epic/pr-dialogue vs dev          [↵] lazygit    │
│  M backend/AccessService/…/InviteLink.cs            +38 -4                     │
│  A frontend/src/features/pr-dialogue/ui/Thread.tsx  +120  …14 files +412 -88   │
├────────────────────────────────────────────────────────────────────────────────┤
│ [↑↓]nav [↵]diff→lazygit [a]attach [n]new [r]resume [k]kill [g]gc [/]filter [q]  │
└────────────────────────────────────────────────────────────────────────────────┘
```

## Stack

- **Go + Bubble Tea** (Charm: Bubble Tea v2, Lipgloss, Bubbles). Chosen on pure
  technical fit — see `docs/DESIGN.md#tech-choice`. Single static binary, ideal
  for concurrent shell-out, and every domain reference (lazygit/k9s/gwq) is Go.
- Shells out to: `lazygit` + `delta` (diff review), `tmux` (attach), the agent
  CLIs (`claude`, `codex`).

## Install (after v1 is built)

```bash
go install github.com/<you>/gvardia/cmd/gvardia@latest
# or: brew install <you>/tap/gvardia  (goreleaser tap, planned)
gvardia            # launch the TUI
gvardia agents --json   # headless: dump fleet as JSON (scripting)
```

## Repo layout (planned)

```
cmd/gvardia/            entry point
internal/config/        ~/.config/gvardia/config.toml loader
internal/collect/       worktree + git-status collectors (concurrent)
internal/adapters/      pluggable per-harness status: claude, codex, tmux, …
internal/model/         Project / Worktree / Session domain types
internal/ui/            Bubble Tea model/update/view (3-pane cockpit)
tools/wt-prune          worktree GC across all roots (usable standalone today)
docs/                   DESIGN.md · PLAN.md · ROADMAP.md · ADAPTERS.md
```

## Building on this / teaching it

The whole stack is deliberately **portable and stack-agnostic**: git worktrees +
tmux + a small Go binary + convention. Adapters make it work with any agent. See
`docs/ADAPTERS.md` to add your own, and `docs/ROADMAP.md` for the growth path
(live-watch, MR/CI enrichment, status-table, cross-repo scan, kanban).

## License

MIT — see `LICENSE`.
