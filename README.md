# gvardia

**Command your fleet of coding agents from the terminal.**

`gvardia` is a local Agent Operations Console. It gives you one terminal screen
over your projects, tasks, installed agent CLIs, local runs, tmux sessions, diffs,
reports, artifacts, and history.

It is **agent-agnostic** by design. The source of truth is `git` + `tmux`.
Per-harness status still comes from adapters (`claude`, `codex`, `tmux`), and new
local runs use runner profiles for `claude`, `codex`, `gemini`, `opencode`,
`aider`, `goose`, or custom commands from config.

`gvardia` is a thin router, not an orchestrator. It never reimplements git, diff
viewing, or agent management. It aggregates state and shells out to proven tools
(`lazygit`, `delta`, `tmux`, the agent CLIs).

## What it looks like

```
┌ PROJECTS (live first) ─┬ WORK · education-platform ───────────────────────────┐
│▸education-platform 3●  │  st   harness agent          task   branch        Δ  last│
│  se-tutorial      2●   │  ● busy  claude edu-85        #675  feat/675-s3 +90/-12 2m│
│  senior-ticker    1○   │  ● busy  claude edu-18        #712  epic/pr-dial +412/-8 5m│
│  OpenTicker       0    │  ○ idle  claude edu-da         —    dev                 1h│
│  … (0 live)            │  ✓ ended codex  ab12ef90      #649  fix/quiz     +30/-4  3h│
├────────────────────────┴──────────────────────────────────────────────────────┤
│ DETAIL · edu-85                                                                 │
│ Finish OpenIddict snake_case + review fixes for s3                             │
│ busy claude · #675 · feat/675-s3 · 14 files +90 -12 · 2m                        │
│ report  Done: switched the token claims to snake_case and green-lit the tests.  │
│ artifacts (3)  M src/Auth.cs · A tests/AuthTests.cs · report .gvardia/reports/… │
├────────────────────────────────────────────────────────────────────────────────┤
│ ↑↓ nav · enter drill · esc back · d diff · w worktrees · t tasks · h history …  │
└────────────────────────────────────────────────────────────────────────────────┘
```

The main work pane now prefers local gvardia runs when they exist. A run has a
task, runner profile, worktree, tmux target, diff stat, report, and artifacts.
Existing live sessions and history are still available. Press `u` for runs, `w`
for worktrees, `h` for ended sessions, `t` for tasks, and `d` for `lazygit`.
Keybinds work under a Russian (ЙЦУКЕН) layout too.

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
gvardia tools --json     # installed/missing agent CLIs + runner profiles
gvardia agents --json    # headless: the whole fleet as JSON (scriptable)
gvardia projects --json  # headless: projects + worktrees, no agent join
gvardia tasks            # headless: the kanban, grouped by column
wt-prune ~/code          # dry-run: list merged / stale worktrees
wt-prune --yes ~/code    # remove merged worktrees (never primary or dirty)
```

## Keybindings

| Key        | Action                                                         |
|------------|----------------------------------------------------------------|
| `↑↓` `j k` | navigate the focused level (arrows work on any keyboard layout)|
| `enter`    | drill down a level (projects → sessions → detail)              |
| `esc` `⌫`  | climb back up a level                                          |
| `tab`      | jump between the projects and sessions levels                  |
| `u`        | show local gvardia runs for the selected project                |
| `d`        | open the selection's worktree in `lazygit` (fallback: `git diff`) |
| `o`        | open the selected run's `report.md` in a pager                  |
| `w`        | toggle the worktree view (every worktree + which agent runs there) |
| `t`        | open the kanban browser (`p` scope to project, `/` filter)     |
| `h`        | toggle recent ended sessions (history) in the work pane        |
| `a`        | attach in place: `tmux attach`, else resume the harness        |
| `r`        | hand off: copy `cd <wt> && <harness> resume` to the clipboard  |
| `n`        | launch a run: choose task, choose runner, start tmux session   |
| `A`        | track an existing repo as a project (curation)                 |
| `C`        | create a new project (`git init`) and track it                 |
| `X`        | untrack the selected project (never deletes the repo)          |
| `k`        | kill the live session process (SIGTERM, confirms first)        |
| `g`        | gc merged/stale worktrees via `wt-prune` (confirms first)      |
| `/`        | filter projects (or tasks) by name or branch                   |
| `R`        | force refresh now                                              |
| `q`        | quit                                                           |

Run status glyphs: `● run` · `◆ review` · `✓ done` · `✖ fail` · `■ killed`.
Session glyphs: `●` busy · `○` idle · `◍` codex · `✓` ended · `✖` failed.
The `Δ` column is the diff vs the base branch (`+added/-removed`).

**Local tasks and runs.** Project-local tasks live in `.gvardia/tasks/*.md`.
Launching a run creates `.gvardia/runs/<run-id>/prompt.md`, `meta.json`, and
`report.md`, creates a linked git worktree, and starts the selected agent command
inside a detached tmux session. The gvardia TUI stays open.

**Curation.** By default gvardia scans `roots` for repos. Press `A`/`C` to track
specific projects instead; the tracked list lives in
`~/.config/gvardia/projects.toml` and, once non-empty, replaces the scan.

## Configuration

`gvardia` reads `~/.config/gvardia/config.toml` (override with `--config`). Every
key is optional; the defaults are shown below.

```toml
roots = ["~/code"]                 # dirs scanned when no projects are curated
brain = "~/Work/sachkov-os"        # kanban source: tasks/{inbox,active,done}/*.md
refresh_interval = "5s"            # how often the cockpit re-collects
adapters = ["claude", "codex", "tmux"]

[base]                             # base branch per project for diff + ahead/behind
default = "auto"                   # auto = dev if it exists, else main
"education-platform" = "dev"       # per-project override

[commands]
lazygit = "lazygit"                # override the lazygit binary/path

[[tools]]
name = "my-agent"
command = "my-agent-cli"

[[runner_profiles]]
name = "my-agent"
tool = "my-agent"
command_template = "my-agent-cli {{prompt_path}}"
```

The curated project list is managed by the TUI (`A`/`C`/`X`) in a separate file,
`~/.config/gvardia/projects.toml`, so it never clobbers your hand-written config.

## Degrades gracefully

Nothing here is a hard dependency beyond `git`:

- An absent adapter CLI (`claude`, `codex`) or a stopped `tmux` server is skipped
  with a banner. A partial fleet beats no fleet.
- No `lazygit`? `enter` falls back to `git diff` through `delta` (or the default
  pager).
- Missing runner tools show up as missing in `gvardia tools --json`; they do not
  break the cockpit.
- Per-repo git errors are skipped, never fatal to the whole scan.

## Commands

- `gvardia` — the three-pane cockpit (projects · sessions · diff).
- `gvardia tools --json` — installed/missing agent CLI tools and runner profiles.
- `gvardia agents --json` — headless fleet dump: projects, worktrees, and the
  agent sessions joined to them.
- `gvardia projects --json` — the git-only view (worktrees + status, no agents).
- `gvardia tasks` — dump the kanban (from `brain`) grouped by column.
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
internal/model/    Project / Worktree / Session / Task domain types
internal/history/  ended-session summaries + reports from agent transcripts
internal/runners/  installed agent tool discovery + runner profiles
internal/tasks/    brain kanban reader + local .gvardia/tasks store
internal/runs/     local .gvardia/runs store
internal/prompts/  task-to-agent prompt rendering
internal/terminal/ tmux launch/attach/kill service
internal/ui/       Bubble Tea model/update/view (runs cockpit + tasks browser)
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
