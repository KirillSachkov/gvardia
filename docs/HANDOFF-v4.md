# gvardia v4 — Handoff (start here)

You are picking up **gvardia v4 (the hub: push-truth monitoring)**. This is the
single entry point — read it, then the two design/plan docs, then start Phase H0.
Nothing here assumes context from the session that wrote it.

## What gvardia is

A Go + Bubble Tea **terminal cockpit over a fleet of coding agents** (Claude Code,
Codex) across the user's git projects. Module `github.com/KirillSachkov/gvardia`.
Shipped to Homebrew (`KirillSachkov/tap/gvardia`). Today (through **v0.3.0**) it is
a **pull** monitor: it scrapes git worktrees + agent transcripts and *infers* what
each agent is doing. v4 adds the missing half — a **push** channel so status is
honest by construction.

## Where things stand (2026-07)

- **Shipped: v0.3.0** on `main`, pushed to origin, live on Homebrew.
  - v1 (cockpit), v2 (monitoring: one row per work-session, honest liveness,
    summaries, history), v3 (navigable: drill-down nav, RU keyboard layout,
    project curation, worktree view, brain-kanban tasks, clipboard handoff,
    artifacts/reports).
- **v4 is DESIGNED + PLANNED, NOT built.** Two docs are committed to `main`:
  - `docs/DESIGN-v4-hub.md` — the pull→push inversion, verified feasibility,
    architecture, 6 decisions, non-goals.
  - `docs/PLAN-v4-hub.md` — executable phased plan (H0→H5), each phase gated.
- No `v4` branch exists yet. `main` @ `e6070eb`, working tree clean.

## Read order

1. This file.
2. `docs/DESIGN-v4-hub.md` — the *why* and the shape (S1 monitor-hub scope:
   honest status + artifacts + reports + agent↔task links + lazygit panel UI;
   **no auto-orchestration** — that is deliberately deferred to S2/S3).
3. `docs/PLAN-v4-hub.md` — the *how*, phase by phase. It has its own
   "Context for a fresh session" + "Verified dependencies" sections.

## Start here (first action)

```bash
cd /Users/dev/code/gvardia
git checkout -b v4
# Phase H0: hub skeleton + hooks channel — the push-truth proof.
# New files: internal/hub/{hub,state,event,store}.go + _test.go, cmd/gvardia/hub.go
# Modify:    cmd/gvardia/main.go (route `hub`), internal/config/config.go (add [hub])
```

Execute **phase by phase, in order** (H0→H5), commit after each. Do not start a
phase before the previous is green and committed. H0 first because it delivers the
whole thesis (honest push status) with **zero MCP SDK**, purely over Claude Code
hooks, and is the cheapest thing to validate on the real fleet.

## Build / test gate (every phase ends green)

```bash
go build ./... && go vet ./... && go test ./... -race
gofmt -l .   # must print nothing
```

Tests are table-driven vs. captured fixtures. **No I/O in Bubble Tea Update/View** —
side effects only in `tea.Cmd`. Value-receiver `Update` mutates a local `m`, returns it.

## Verified dependencies — do NOT re-research (confirm APIs compile)

- **Go 1.25+ floor** (`go.mod` says `go 1.25.0`). Forced by the Charm v2 stack.
- **Charm Bubble Tea v2 at `charm.land/*`** import paths — NOT
  `github.com/charmbracelet`. `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`,
  `charm.land/lipgloss/v2`. `View() tea.View` (wrap strings via `tea.NewView`),
  AltScreen is a `tea.View` field, keys are `tea.KeyPressMsg` matched via
  `msg.String()`.
- **Official MCP Go SDK** `github.com/modelcontextprotocol/go-sdk` **v1.6.1 (GA)**
  — for Phase H1+. Server: `mcp.NewServer`, `mcp.AddTool[In,Out]`,
  `mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server, opts)` mounted at
  `/mcp`. Handlers run concurrently → hub state behind a mutex. Reap idle
  streamable-HTTP sessions (issue **#499** leaks goroutines) and surface the count.
- **Claude Code hooks** POST JSON to a URL (`PreToolUse`/`PostToolUse`/`Stop`/
  `Notification`), carrying `session_id`, `cwd`, `tool_name`, `hook_event_name`.
  `.mcp.json` `type:"http"` wires the MCP hub. MCP is **not** connected at
  `SessionStart` → first status comes from `PostToolUse`. **Capture a real hook
  payload in H0 and parse against that fixture — do not trust assumed field names.**

## Release recipe (Phase H5 — done locally, NOT via CI)

```bash
# after merging v4 → main:
git tag -a v0.4.0 -m "v0.4.0 — the hub"
GITHUB_TOKEN=$(gh auth token) TAP_GITHUB_TOKEN=$(gh auth token) goreleaser release --clean
```

- `gh` CLI is authed as **KirillSachkov** (`repo` scope). Homebrew tap
  `KirillSachkov/homebrew-tap`.
- goreleaser changelog is `use: git` (needs the tag pushed first for the compare API).
- **The API-created tag spawns a FAILING CI Release run** — there is deliberately
  **no `TAP_GITHUB_TOKEN` CI secret** (don't store the broad `gho_` token). Delete
  that failing run: `gh run delete <id>`. This is expected, not a bug.
- Verify: `brew upgrade KirillSachkov/tap/gvardia` → `gvardia --version` = 0.4.0,
  `gvardia hub --help` works.
- `gvardia` already ships the daemon as the `hub` subcommand — no new binary.

## Gotchas that will bite you

- **`charm.land/*`, not `github.com/charmbracelet`.** The docs say "Go 1.23+";
  reality is 1.25 because of the v2 stack.
- **The shell aliases `ls`** to an icon renderer — command-substitution on `ls`
  output gets a glyph prefix. Use `find` / `/bin/ls` for clean paths in scripts.
- **Identity correlation (`run`→agent) is the one genuinely tricky bit** (H1):
  MCP tool calls don't inherently carry the Claude `session_id`, so tools take an
  explicit `run` tag (injected via onboarding) and fall back to `cwd`. Verify live.
- **Destructive UI actions** (k kill / g gc / n new / a attach) are unit-tested but
  were NOT fired against the live fleet — they SIGTERM / mutate real worktrees.
- **Codex support is thinner** than Claude's for hooks/MCP. Claude Code is the
  first-class target; codex is best-effort — verify its syntax live.

## Cross-agent handoff notes

- **Brain = source of truth.** The user's canonical "brain" is the git repo
  `sachkov-os` (this Mac: `/Users/dev/Work/sachkov-os`; VPS: Hermes). On conflict,
  the brain wins over local assumptions/caches/memory. gvardia's own plan lives in
  THIS repo (the docs above) — that is the code source of truth for v4.
- gvardia reads the brain kanban directly (`<brain>/tasks/{inbox,active,done}/*.md`,
  flat YAML frontmatter) as its task source — NOT GitLab. Config `brain` defaults to
  `~/Work/sachkov-os`. H2 links agents↔those tasks.
- If you are **not** Claude Code (e.g. Codex/Hermes): Claude's auto-memory under
  `~/.claude/projects/...` is per-machine Claude-only — ignore it; everything you
  need is in these repo docs.
- The graph snapshot `graphify-out/` exists here — for orientation questions,
  `graphify query "<q>"` / `graphify explain`; for precise symbol/reference
  navigation use serena (`find_symbol` / `find_referencing_symbols`).

## Definition of done for v4

`brew upgrade KirillSachkov/tap/gvardia` → v0.4.0; `gvardia hub` runs the daemon;
one `gvardia hub onboard` in a fresh project makes a real agent show up in the
dashboard with **honest live status**, its artifacts + a report, and drill-in fuses
the worktree diff; killing the hub falls back to today's pull cockpit unchanged.
