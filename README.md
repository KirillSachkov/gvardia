# gvardia

**Command your fleet of coding agents from the terminal.**

`gvardia` is a terminal cockpit (TUI) that gives you one screen over every coding
agent working across **all** your projects: which projects are in flight, which
agents are running (Claude Code, Codex, …), what they changed, and their status,
without juggling ten different commands.

It is **agent-agnostic** by design. The source of truth is `git` + `tmux`, and
per-harness status comes from **pluggable adapters** (`claude`, `codex`, `tmux`).
Any agent that runs in a git worktree and leaves commits behind shows up.

`gvardia` is a thin router, not an orchestrator. It never reimplements git, diff
viewing, or agent management. It aggregates state and shells out to proven tools
(`lazygit`, `delta`, `tmux`, the agent CLIs).

## What it looks like

```
┌ gvardia ─────────────────────────────────────────────── ~/code · 4 agents live ┐
│ PROJECTS             │ SESSIONS · education-platform                            │
│▸education-platform   │ ● busy  claude  education-platform-18  epic/pr-dialogue ✱↑412↓88│
│  84 wt · 3 live      │ ● busy  claude  education-platform-85  feat/675-s3       ↑90↓12 │
│ sharp.arena          │ ◍ idle  codex   ab12ef90             fix/quiz-render    ↑30↓4  │
│  5 wt · 0 live       │ ○ idle  claude  education-platform-da  dev                      │
│ telegram-bot-flow    │                                                          │
│  2 wt · 0 live       │                                                          │
├──────────────────────┴──────────────────────────────────────────────────────┤
│ DIFF · epic/pr-dialogue vs dev                                                 │
│  backend/AccessService/…/InviteLink.cs            | 38 ++++--                   │
│  frontend/src/features/pr-dialogue/ui/Thread.tsx  | 120 +++  …14 files, +412 -88│
├────────────────────────────────────────────────────────────────────────────────┤
│ ↑↓ nav · tab focus · enter diff · a attach · n new · k kill · g gc · / filter · q│
└────────────────────────────────────────────────────────────────────────────────┘
```

## Install

Prebuilt binaries and a Homebrew tap ship with each release.

```bash
# Homebrew (macOS / Linux): installs gvardia + wt-prune
brew install KirillSachkov/tap/gvardia

# From source (needs Go 1.25+)
go install github.com/KirillSachkov/gvardia/cmd/gvardia@latest
go install github.com/KirillSachkov/gvardia/cmd/wt-prune@latest

# Or grab a binary from the Releases page and put it on your PATH.
```

Optional but recommended companions (gvardia degrades gracefully without them):
`lazygit` and `delta` for diff review, `tmux` for attach.

## Quickstart

```bash
gvardia                  # launch the cockpit
gvardia agents --json    # headless: the whole fleet as JSON (scriptable)
gvardia projects --json  # headless: projects + worktrees, no agent join
wt-prune ~/code          # dry-run: list merged / stale worktrees
wt-prune --yes ~/code    # remove merged worktrees (never primary or dirty)
```

## Keybindings

| Key        | Action                                                         |
|------------|----------------------------------------------------------------|
| `↑↓` `j k` | navigate the focused pane                                      |
| `tab`      | switch focus between projects and sessions                     |
| `enter`    | open the selected worktree in `lazygit` (fallback: `git diff`) |
| `a`        | attach: `tmux attach`, else resume the session's harness       |
| `r`        | resume the session (`claude --resume`, `codex resume`)         |
| `n`        | new agent: pick a harness, name it, spawn it                   |
| `k`        | kill the session process (SIGTERM, confirms first)             |
| `g`        | gc merged/stale worktrees via `wt-prune` (confirms first)      |
| `/`        | filter projects by name or branch                              |
| `R`        | force refresh now                                              |
| `q`        | quit                                                           |

Status glyphs: `●` busy · `○` idle · `◍` codex · `✖` failed. A branch shows `✱`
when dirty and `↑n↓m` for ahead/behind its base.

## Configuration

`gvardia` reads `~/.config/gvardia/config.toml` (override with `--config`). Every
key is optional; the defaults are shown below.

```toml
roots = ["~/code"]                 # dirs scanned (shallow) for git repos
refresh_interval = "5s"            # how often the cockpit re-collects
adapters = ["claude", "codex", "tmux"]

[base]                             # base branch per project for diff + ahead/behind
default = "auto"                   # auto = dev if it exists, else main
"education-platform" = "dev"       # per-project override

[commands]
lazygit = "lazygit"                # override the lazygit binary/path
```

## Degrades gracefully

Nothing here is a hard dependency beyond `git`:

- An absent adapter CLI (`claude`, `codex`) or a stopped `tmux` server is skipped
  with a banner. A partial fleet beats no fleet.
- No `lazygit`? `enter` falls back to `git diff` through `delta` (or the default
  pager).
- Per-repo git errors are skipped, never fatal to the whole scan.

## Commands

- `gvardia` — the three-pane cockpit (projects · sessions · diff).
- `gvardia agents --json` — headless fleet dump: projects, worktrees, and the
  agent sessions joined to them.
- `gvardia projects --json` — the git-only view (worktrees + status, no agents).
- `wt-prune [roots…]` — worktree GC across your roots. Dry-run by default;
  `--yes` removes merged worktrees, `--stale` also removes stale ones, `--days N`
  sets the staleness threshold. Never touches a primary or dirty worktree.

## Repo layout

```
cmd/gvardia/       entry point (TUI + agents/projects subcommands)
cmd/wt-prune/      worktree GC CLI
internal/config/   ~/.config/gvardia/config.toml loader
internal/collect/  worktree + git-status collectors (concurrent)
internal/adapters/ pluggable per-harness status: claude, codex, tmux, …
internal/model/    Project / Worktree / Session domain types
internal/ui/       Bubble Tea model/update/view (3-pane cockpit)
internal/prune/    worktree classification (merged/stale/active)
docs/              DESIGN.md · PLAN.md · ROADMAP.md · ADAPTERS.md
```

## Stack

**Go + Bubble Tea** (Charm: Bubble Tea v2, Lipgloss v2, Bubbles v2). Single static
binary, ideal for concurrent shell-out, and every domain reference (lazygit, k9s,
gwq) is Go. See `docs/DESIGN.md` for the full rationale, and `docs/ADAPTERS.md` to
add your own harness (it is one file against a small interface).

## License

MIT. See `LICENSE`.
