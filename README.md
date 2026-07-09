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
┌ PROJECTS ──────────────┬ 1 agents | 2 tasks | 3 worktrees | 4 tools | 5 history ┐
│▸education-platform 3●  │ AGENTS · all projects · runs 3 (1 review)            │
│  se-tutorial      2●   │ status   project     runner task          changes age │
│  senior-ticker    1○   │ ◆ review gvardia     codex  Ops console   +90/-12 2m │
│  OpenTicker       0    │ ✖ fail   content-eng codex  Code map      +2/-0   4m │
│  …                      │ ● run    platform    codex  Payment bug   +12/-3  7m │
├────────────────────────┴──────────────────────────────────────────────────────┤
│ Auth cleanup                                                                   │
│ review · claude/claude · gvardia/run-1 · 2m                                    │
│ report                                                                         │
│ Done: switched the token claims to snake_case and green-lit the tests.         │
│ artifacts  report.md · implementation-plan.md                                  │
│ changes    3 files · +90 -12 · d open diff                                    │
├────────────────────────────────────────────────────────────────────────────────┤
│ 1 agents · 2 tasks · 3 worktrees · 4 tools · 5 history · enter detail · ? actions│
└────────────────────────────────────────────────────────────────────────────────┘
```

The right side is a lazygit-style tabbed work area. Press `1` through `5` for
Agents, Tasks, Worktrees, Tools, and History. `enter` drills into the selected
pane, `esc` backs out, and `?` shows actions for the current tab. The detail pane
changes with the selected row: runs show reports/artifacts, tasks show task body,
worktrees show git state, and tools show installed/missing status. Keybinds work
under a Russian (ЙЦУКЕН) layout too.

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
`cmux` for new workspaces and Markdown/diff surfaces, `lazygit` and `delta` for
diff review, and `tmux` for persistent sessions.

## Quickstart

```bash
gvardia                  # launch the cockpit
gvardia tools --json     # installed/missing agent CLIs + runner profiles
gvardia agents --json    # headless: the whole fleet as JSON (scriptable)
gvardia projects --json  # headless: projects + worktrees, no agent join
gvardia tasks            # headless: the kanban, grouped by column
gvardia task create --title "Fix launch" --project gvardia --body "Reproduce and fix it."
gvardia task update --id fix-launch --status active
wt-prune ~/code          # dry-run: list merged / stale worktrees
wt-prune --yes ~/code    # remove merged worktrees (never primary or dirty)
```

## Keybindings

| Key        | Action                                                         |
|------------|----------------------------------------------------------------|
| `↑↓` `j k` | navigate the focused level (arrows work on any keyboard layout)|
| `1`        | agents tab: global attention queue across every project       |
| `2`        | tasks tab: standalone Gvardia tasks, optional project scope   |
| `3`        | worktrees tab: every worktree and the agent running there      |
| `4`        | tools tab: installed/missing agent CLIs                        |
| `5`        | history tab: live + recent ended sessions                      |
| `enter`    | drill down a level (projects → active tab → detail)            |
| `esc` `⌫`  | climb back up a level                                          |
| `tab`      | jump between the projects and active-tab levels                |
| `?`        | show contextual actions for the current tab                    |
| `d`        | open diff in cmux, lazygit, or git/delta                       |
| `e`        | browse indexed run artifacts; Enter opens the selection       |
| `o`        | open the selected run's `report.md` in a pager                  |
| `u` `t` `w` `h` | compatibility aliases for agents/tasks/worktrees/history |
| `p`        | show or hide the project drawer                               |
| `s`        | scope Agents or Tasks to the selected project                 |
| `a`        | attach in a new cmux workspace; fallback copies the command   |
| `r`        | hand off: copy `cd <wt> && <harness> resume` to the clipboard  |
| `n`        | launch a run: choose task, choose runner, start tmux session   |
| `A`        | track an existing repo as a project (curation)                 |
| `C`        | create a new project (`git init`) and track it                 |
| `X`        | untrack the selected project (never deletes the repo)          |
| `k`        | kill the live session process (SIGTERM, confirms first)        |
| `g`        | gc merged/stale worktrees via `wt-prune` (confirms first)      |
| `/`        | filter projects, tasks, or tools by current context            |
| `R`        | force refresh now                                              |
| `q`        | quit                                                           |

Run status glyphs: `● run` · `◆ review` · `✓ done` · `✖ fail` · `■ killed`.
Session glyphs: `●` busy · `○` idle · `◍` codex · `✓` ended · `✖` failed.
The `changes` column is the compact diff summary (`+added/-removed`).

**Local tasks and runs.** Gvardia owns its state outside project repositories.
The default data directory is `$XDG_DATA_HOME/gvardia`, falling back to
`~/.local/share/gvardia`. Tasks live in `tasks/*.md`; each run gets a directory
under `runs/<run-id>/` with its prompt, status, event log, indexed artifacts, and
final report. Existing project-local `.gvardia/runs` directories remain readable
as legacy data.

Launching creates a linked git worktree and starts the selected agent inside a
detached tmux session. Codex is the default runner and uses `-a never -s
danger-full-access`. Gvardia checks the tmux pane immediately and on each refresh,
so a dead process cannot remain marked as running. tmux keeps ownership of the
session; cmux only opens a new workspace for presentation. The Gvardia TUI stays
open.

### Agent evidence contract

Each launched agent receives `GVARDIA_RUN_DIR`, `GVARDIA_REPORT_PATH`,
`GVARDIA_STATUS_PATH`, `GVARDIA_EVENTS_PATH`, and `GVARDIA_ARTIFACTS_DIR`. The
generated prompt asks the agent to report meaningful phase changes, save useful
review material, run verification, and finish with a short structured report.

```bash
gvardia run status --state running --phase tests --summary "Running Go tests"
gvardia run event --type status --message "Implementation complete"
gvardia run artifact --type plan --title "Implementation plan" --file /tmp/plan.md
gvardia run report --file /tmp/report.md
```

The final report uses `Summary`, `Changes`, `Verification`, and `Risks / Next
steps`. An agent may create follow-up work with `gvardia task create`, but
Gvardia never launches that follow-up automatically.

**Curation.** By default gvardia scans `roots` for repos. Press `A`/`C` to track
specific projects instead; the tracked list lives in
`~/.config/gvardia/projects.toml` and, once non-empty, replaces the scan.

## Configuration

`gvardia` reads `~/.config/gvardia/config.toml` (override with `--config`). Every
key is optional; the defaults are shown below.

```toml
roots = ["~/code"]                 # dirs scanned when no projects are curated
refresh_interval = "5s"            # how often the cockpit re-collects
adapters = ["claude", "codex", "tmux"]
data_dir = "~/.local/share/gvardia"
task_sources = ["gvardia"]         # add "brain" only when explicit import is useful
brain = "~/Work/sachkov-os"        # optional source used by task_sources = ["gvardia", "brain"]
default_runner = "codex"

[terminal]
backend = "auto"                   # auto, cmux, or copy
open_on_launch = true
focus_new = true

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
- No `lazygit`? `d` falls back to `git diff` through `delta` (or the default
  pager).
- No `cmux`? Launch and attach copy `tmux attach -t <target>` instead of taking
  over the terminal that runs Gvardia.
- Missing runner tools show up as missing in `gvardia tools --json`; they do not
  break the cockpit.
- Per-repo git errors are skipped, never fatal to the whole scan.

## Commands

- `gvardia` — the tabbed cockpit (projects · agents/tasks/worktrees/tools/history · detail).
- `gvardia tools --json` — installed/missing agent CLI tools and runner profiles.
- `gvardia agents --json` — headless fleet dump: projects, worktrees, and the
  agent sessions joined to them.
- `gvardia projects --json` — the git-only view (worktrees + status, no agents).
- `gvardia tasks` — dump tasks from configured sources.
- `gvardia task create` / `gvardia task update` — let a human or agent write to
  the standalone Gvardia task store.
- `gvardia run status|event|artifact|report` — write structured evidence for the
  active run using `GVARDIA_RUN_DIR`.
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
internal/tasks/    standalone Markdown task store + optional brain reader
internal/runs/     XDG run envelope store + legacy project reader
internal/prompts/  task-to-agent prompt rendering
internal/terminal/ tmux lifecycle + cmux presentation services
internal/ui/       Bubble Tea model/update/view (tabbed operations cockpit)
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
