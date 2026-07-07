# gvardia v4 — Implementation Plan (the hub: push-truth monitoring)

> Implements `DESIGN-v4-hub.md` (scope **S1 monitor-hub**). Execute **phase by
> phase, in order**; commit after each; each **Acceptance** is a gate.
> Conventions (same as v1–v3): Go 1.25+, Charm v2 at `charm.land/*`, every phase
> ends green (`go build ./... && go vet ./... && go test ./... -race`), `gofmt`
> clean, table-driven tests vs captured fixtures, no I/O in Bubble Tea Update/View.
> Branch `v4`, commit per phase, merge to main at the end, release v0.4.0.

**Context for a fresh session:** v3 (shipped v0.3.0) is a pull-only cockpit.
Key existing files: `internal/ui/{model,update,view,items,commands,modal,keys,
tasksview}.go`, `internal/collect/{collect,assemble,changestat,files,task}.go`,
`internal/history/*`, `internal/tasks/tasks.go`, `internal/model/model.go`,
`internal/config/{config,projects}.go`, `cmd/gvardia/{main,cli,agents,projects,
tasks}.go`. v3 already reads `.gvardia/reports/*.md` and `Session.Report`
(Phase 6), the brain kanban (`internal/tasks`), and worktrees. v4 ADDS a daemon
(`gvardia hub`) and a hub-mode dashboard; the pull cockpit stays as the fallback.

**Verified dependencies (do not re-research; confirm APIs live):**
- Official MCP SDK `github.com/modelcontextprotocol/go-sdk` **v1.6.1** (GA).
  Server: `mcp.NewServer`, `mcp.AddTool[In,Out]`, `mcp.NewStreamableHTTPHandler(
  func(*http.Request) *mcp.Server, opts)` mounted at `/mcp`. Reap idle sessions
  (issue #499). Handlers concurrent → state behind a mutex.
- Claude Code hooks POST JSON to a URL (`PreToolUse`/`PostToolUse`/`Stop`/
  `Notification`), carry `session_id`, `cwd`, `tool_name`, `hook_event_name`.
  `.mcp.json` `type:"http"` for the MCP hub. MCP not up at `SessionStart` → first
  status from `PostToolUse`. Verify exact hook payload keys live before parsing.

---

## Phase H0 — Hub skeleton + hooks channel (the push-truth proof)

**Files:** `internal/hub/{hub,state,event,store}.go` (new), `internal/hub/*_test.go`,
`cmd/gvardia/hub.go` (new), `cmd/gvardia/main.go` (route `hub`),
`internal/config/config.go` (add `Hub`).

- `internal/config`: `Hub struct { Addr string `toml:"addr"`; Enabled bool
  `toml:"enabled"` }`; default `Addr:"127.0.0.1:9100"`, `Enabled:true`.
- `internal/model`: `Agent{ ID, Harness, Name, Cwd, Status, StatusText,
  Task string; LastActivity time.Time; Artifacts []Artifact }`;
  `Event{ TS time.Time; Kind, Agent, Tool, Text, Cwd string }`.
- `internal/hub/state.go`: `State` = mutex-guarded `map[string]*model.Agent` +
  a bounded `[]model.Event` ring (request log). `Apply(Event)` projects an event
  onto the agent (create/update by `Agent` key = session_id; map hook events →
  Status: PostToolUse→`working`, Stop→`idle`, Notification(needs input)→`waiting`,
  Notification(completed)/SessionEnd→`stopped`). `Snapshot()` returns a copy for
  `/state`.
- `internal/hub/store.go`: append-only `events.jsonl` under
  `~/.local/state/gvardia/hub/`; `Append(Event)`, `Replay() []Event` (rebuild
  projection on startup). Size-rotate.
- `internal/hub/hub.go`: `Server{state,store,cfg}`; routes `POST /hooks/event`
  (decode Claude hook JSON → normalize to `Event` → `store.Append` + `state.Apply`),
  `GET /state` (JSON snapshot), `GET /healthz`. `Run(ctx)` on `cfg.Hub.Addr`.
  A janitor goroutine drops agents idle > TTL.
- `cmd/gvardia/hub.go`: `gvardia hub` runs the daemon (foreground, ctrl-c to stop);
  `gvardia hub install-hooks [--project DIR]` writes a Claude Code hook block into
  `<DIR>/.claude/settings.json` (or `~/.claude/settings.json`) POSTing
  PostToolUse/Stop/Notification to `http://<addr>/hooks/event`, merging without
  clobbering existing hooks; prints next steps.

**Tests:** `State.Apply` maps each hook kind → the right Status (table-driven);
`Replay` reconstructs agents from a temp `events.jsonl`; `/hooks/event` handler
decodes a captured Claude hook payload fixture and updates state; install-hooks
merges into an existing settings.json without dropping prior hooks.

**Acceptance:** start `gvardia hub`; run `gvardia hub install-hooks` in a real
project; launch a real Claude Code session there and do work — `curl
localhost:9100/state` shows that agent flipping `working`→`idle`→`waiting` live,
honestly, with its cwd/tool. **This is the S0 push-truth proof, delivered in H0.**

## Phase H1 — MCP server + artifacts/reports tools

**Files:** `internal/hub/mcp.go` (new), `internal/hub/hub.go` (mount `/mcp`),
`cmd/gvardia/hub.go` (`install-mcp`), `go.mod` (add the SDK).

- `go get github.com/modelcontextprotocol/go-sdk@v1.6.1`; confirm the streamable
  HTTP handler + `AddTool` signatures compile (they are the load-bearing API).
- `internal/hub/mcp.go`: build one shared `*mcp.Server`; register tools (typed
  in/out structs), each writing through `state`+`store` (thread-safe):
  - `set_status{run, text}` → Event kind `status`.
  - `save_artifact{run, title, type, path?, body?}` → Event kind `artifact`
    (type ∈ plan|note|report); attach to the agent + a global artifacts list.
  - `report{run, title, body}` → `save_artifact` with type `report`.
  Correlate `run` → agent by session_id, else by cwd. Mount via
  `mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv }, nil)`
  at `/mcp` on the same mux as the hooks endpoint.
- Session reaper: track streamable sessions; drop/monitor to avoid the #499 leak;
  expose the count on `/state` (hub health).
- Also index `<worktree>/.gvardia/reports/*.md` into the agent's artifacts
  (reuse `collect.reportArtifacts`), so file-convention reports show up too.
- `gvardia hub install-mcp [--project DIR]` writes `.mcp.json` with
  `{"mcpServers":{"gvardia":{"type":"http","url":"http://<addr>/mcp"}}}`.

**Tests:** each tool handler updates state (call the handler directly with a typed
request, assert the projected agent/artifact); `save_artifact` type validation;
report → artifact(type=report); `run`→agent correlation by id and by cwd fallback.
(MCP transport itself is the SDK's concern — test our handlers, not the wire.)

**Acceptance:** with the hub running and `.mcp.json` installed, a real agent calls
`save_artifact` / `report` (or you call the tool via an MCP client) and it appears
in `/state`; a `.gvardia/reports/*.md` file also appears as an artifact.

## Phase H2 — Tasks: kanban ↔ agent links

**Files:** `internal/hub/mcp.go` (task tools), `internal/hub/state.go` (link),
`internal/tasks/tasks.go` (reuse `Load`).

- Tools: `list_tasks{}` → the brain kanban (via `tasks.Load(cfg.Brain)`);
  `claim_task{run, id}` → record agent→task link in hub state (NOT writing brain
  files yet — link lives in the hub); `update_task{run, id, status}` → recorded as
  an Event (surfaced; brain-file write deferred per Non-goals).
- `state.Apply` sets `Agent.Task` on claim; `/state` exposes the link both ways.

**Tests:** `list_tasks` returns tasks from a temp brain; `claim_task` sets the
agent's Task and the reverse link; unknown task id is rejected cleanly.

**Acceptance:** an agent calls `claim_task` and the hub shows that agent working
that task (agent↔task link), visible in `/state`.

## Phase H3 — TUI hub dashboard (lazygit-style panels)

**Files:** `internal/ui/{model,update,view}.go`, `internal/ui/hubclient.go` (new),
`internal/ui/hubview.go` (new).

- `internal/ui/hubclient.go`: `fetchState(addr) (HubState, error)` (GET /state,
  short timeout); a `tea.Cmd` + ticker like `collectFleet`. On error → hub down.
- Model: `hubUp bool`, `hub HubState`. On startup try the hub; poll on the tick.
- `internal/ui/hubview.go` + view: when `hubUp`, render the dashboard —
  **Agents** (honest status glyphs: ● working · ◐ waiting · ○ idle · ✖ stopped),
  **Detail** (drill into agent: status line · task · artifacts · report · worktree
  diff from the pull layer), **Artifacts**, **Request Log** (recent events),
  **Status** (hub health + session count). Reuse `normalizeKey`, the level/drill
  model, and the diff viewport from v3.
- **Fallback:** `!hubUp` → the existing pull cockpit unchanged.

**Tests:** message-driven — a `hubStateMsg` populates the agents panel; `render()`
in hub mode contains a known agent's status + a report marker; hub-down renders
the pull cockpit; status-glyph mapping table.

**Acceptance:** with the hub running and a live agent, `gvardia` shows the panel
dashboard with honest live status, the agent's artifacts + report, and drill-in
fuses the worktree diff. Killing the hub falls back to the pull cockpit.

## Phase H4 — Onboarding skill + report convention + docs

**Files:** `cmd/gvardia/hub.go` (`onboard`), `docs/` (skill snippet), `README.md`.

- `gvardia hub onboard [--project DIR]` = `install-hooks` + `install-mcp` + drop a
  `gvardia-report` instruction (AGENTS.md/skill snippet) telling the agent to call
  `set_status` as it works and write a `.gvardia/reports/<slug>.md` + `report(...)`
  at session end. One command points a project's agents at the hub.
- README/DESIGN: hub quickstart (`gvardia hub` → `gvardia hub onboard` → `gvardia`),
  the panel UI, the push+pull model, honest caveats (hooks setup per project).

**Tests:** `onboard` writes all three artifacts into a temp project idempotently.

**Acceptance:** one `gvardia hub onboard` in a fresh project makes a new agent show
up in the dashboard with live status and, at the end, a report that appears in its
artifacts — with zero manual config.

## Phase H5 — Release v0.4.0

- goreleaser: `gvardia` already ships the daemon (`gvardia hub` subcommand) — no
  new binary. Merge `v4` → main; `git tag -a v0.4.0`; local `goreleaser release`
  (`GITHUB_TOKEN`/`TAP_GITHUB_TOKEN` = `gh auth token`); delete the failing CI
  Release run; `brew upgrade` to verify `gvardia --version` → 0.4.0 and
  `gvardia hub --help` works.

**Acceptance:** `brew upgrade KirillSachkov/tap/gvardia` → v0.4.0; `gvardia hub`
runs the daemon; onboarding + dashboard work end-to-end on the real fleet.

---

## Sequencing note

H0 first — it delivers the whole thesis (honest push status) with zero MCP SDK,
purely over hooks, and is the cheapest thing to validate on the real fleet. H1
adds the MCP tools (artifacts/reports), H2 the task links, H3 the dashboard UI,
H4 onboarding. Each phase is independently shippable and the pull cockpit is never
broken (the hub is additive with a graceful fallback). Do not start a phase before
the previous is green and committed.

## Risks carried from the design

- **#499 session leak** — reap idle streamable-HTTP sessions; surface the count.
- **Identity correlation** (`run`→agent) is the trickiest bit — verify live in H1.
- **Hook payload keys / Notification sub-events** — capture a real payload in H0
  and parse against that fixture, not against assumed field names.
- **Codex** — hooks/MCP support is thinner than Claude's; Claude Code is the
  first-class target, codex best-effort.
