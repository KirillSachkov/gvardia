# gvardia v4 — Design (the hub: push-truth monitoring over live agents)

Extends v3 (`DESIGN-v3.md`, shipped v0.3.0). v1–v3 made gvardia a **pull** monitor:
it scrapes git + agent transcripts and *guesses* what each agent is doing. That
guess is why codex looked "live" from stale files. v4 adds the missing half — a
**push** channel: agents report to a central **gvardia-hub**, so status is honest
**by construction**. This is scope **S1 (monitor-hub)** of the orchestrator study:
observe truthfully, drill into an agent, see its artifacts and reports, link
agent↔task — but **no auto-orchestration** (pipelines/spawn/roles are a later
scope, S2–S3). The reference is the MCP-orchestrator screenshot (server on :9100,
Agents · Pipeline · Artifacts · Request Log panels); we build its *monitoring*
half first, grounded in git.

## The core inversion

| | how it learns status | honesty |
|---|---|---|
| herdr | you watch raw terminals | you infer |
| **gvardia v1–v3 (pull)** | scrape git + transcripts | a guess (codex lied) |
| **gvardia v4 (push)** | **agents report to the hub** | true by construction |

v4 keeps the pull layer (worktrees, diffs, history — still valuable ground truth)
and **fuses** it with push: the hub knows what an agent *says*, git knows what it
*did*. Neither the screenshot (push-only) nor herdr has both.

## Verified feasibility (research, early 2026)

- **Go MCP SDK:** official `github.com/modelcontextprotocol/go-sdk` **v1.6.1, GA**
  (API-stability guarantee, Google collab). Server tools via `mcp.AddTool[In,Out]`.
  **Streamable HTTP** transport (`mcp.NewStreamableHTTPHandler(func(*http.Request)
  *mcp.Server)` returning one shared server) → many agent processes on one port.
  Known risk: abandoned streamable-HTTP sessions leak goroutines (issue #499) →
  the hub must reap idle sessions and watch the session count. Tool handlers run
  concurrently → hub state behind a mutex. (Alt `mark3labs/mcp-go` is popular but
  still v0.x, no GA — we take the official one.)
- **Claude Code** connects to an HTTP MCP hub via `.mcp.json` (`type: http`,
  `url`). Crucially, **hooks POST to a local HTTP endpoint deterministically**:
  `PreToolUse` / `PostToolUse` (tool_name, cwd, session_id), `Stop` (turn ended),
  `Notification` (agent needs input / completed). Hooks are the *honest-status
  backbone* — they fire whether or not the model chooses to call a tool. Caveats:
  MCP servers are **not** connected at `SessionStart` (send first status from
  `PostToolUse`); mcp_tool-hook failures are silent (non-blocking — good for not
  breaking agents). Exact hook sub-event names to re-verify against live docs.
- **Codex** supports MCP (stdio + HTTP). Docs sparse → verify hook/MCP syntax
  live during implementation. Claude Code is the first-class target; codex second.

## Architecture

```
  agent A (claude)         agent B (codex)          agent C (claude)
     │        │                │                        │      │
 [hooks]  [MCP tools]      [MCP tools]              [hooks] [MCP tools]
  POST      call             call                    POST     call
     ▼        ▼                ▼                        ▼      ▼
  ┌──────────────────────────────────────────────────────────────┐
  │                      gvardia hub  (daemon)                    │
  │  POST /hooks/event   ·   /mcp (streamable HTTP)   ·  GET /state│
  │  in-memory state  ←  append-only events.jsonl (replayable)    │
  │  agents · artifacts · reports · request log · task links      │
  └──────────────────────────────────────────────────────────────┘
     ▲ GET /state (poll/SSE)                 + pull layer (git worktrees, diff, history)
     │
  ┌───────────────────────────────────┐
  │  gvardia  TUI  (ephemeral)         │  Agents · Work · Artifacts · Log · Status
  └───────────────────────────────────┘
```

### Decisions

1. **Two processes, not one.** `gvardia hub` is an **always-on daemon** (like the
   screenshot's :9100 server) — it must outlive any TUI because agents post to it
   whenever they run. `gvardia` (the TUI) is ephemeral and reads `/state`.
   **Graceful fallback:** if the hub is unreachable, the TUI runs today's pull-only
   cockpit unchanged. The hub is additive, never required.

2. **Two inbound channels.**
   - **`POST /hooks/event`** — Claude Code hook payloads (PreToolUse / PostToolUse
     / Stop / Notification / SessionStart / SubagentStop). Deterministic; the
     honest-status backbone. **No MCP SDK needed for this** — plain `net/http`.
   - **`/mcp`** (streamable HTTP, official SDK) — agent-initiated tools:
     `set_status`, `save_artifact`, `report`, `list_tasks`, `claim_task`,
     `update_task`. Richer, agent-driven; used for artifacts/reports/tasks.

3. **State = in-memory projection over an append-only event log.** Every hook and
   tool call appends one JSON line to `~/.local/state/gvardia/hub/events.jsonl`
   (the **Request Log** panel *is* this log). Live state (agents, artifacts, task
   links) is a projection, rebuilt on restart by replaying the log. No database —
   debuggable, agent-legible, static-binary-friendly. Compaction: rotate the log
   by size; keep the projection bounded (drop agents idle > TTL).

4. **Agent identity.** Primary key = **`session_id`** from hooks (carries cwd →
   worktree → project via the existing git layer). MCP calls don't inherently
   carry the Claude session id to the server, so tools take an explicit **`run`**
   tag (the agent's session id, injected via the onboarding skill / an env the
   hooks echo) and fall back to **cwd correlation**. This correlation is the one
   genuinely tricky part — spec'd conservatively and verified live.

5. **Reports as first-class artifacts, two ways (both indexed).**
   - Agent calls MCP `report(run, title, body)` → stored as artifact `type=report`.
   - Agent writes `<worktree>/.gvardia/reports/*.md` → already read since v3
     (Phase 6); the hub also watches these.
   - An **onboarding skill** (`gvardia-report`) instructs each agent to write a
     report at session end. This realizes "every agent generates its own report."

6. **One-command onboarding.** `gvardia hub onboard [--project DIR]` writes the
   Claude Code hook config (settings.json) + `.mcp.json` pointing at the hub + the
   report skill, so pointing an agent at the hub is a single command — otherwise
   nobody wires the hooks and the hub stays empty.

### TUI (lazygit-style panels, hub mode)

When the hub is reachable, the cockpit becomes a multi-panel dashboard fused with
the git layer:
- **Agents** — every reporting agent: honest live status (`working` / `waiting`
  (needs input) / `idle` / `stopped`), harness, role/name, current task.
- **Work / Detail** — drill into the selected agent: current status line, claimed
  task, its artifacts, its report, and (from the pull layer) its worktree diff.
- **Artifacts** — typed list (plan / note / report), like the screenshot.
- **Request Log** — the live event stream (hook + tool calls), the audit trail.
- **Status** — hub health, session count (watch #499), host metrics.

Fallback (no hub) = today's projects · sessions · diff cockpit.

## Non-goals (this scope, S1)

Auto-orchestration: pipelines, stages, `run_agent_pipeline`, spawning agents by
role, retries, routing. That is S2 (light control: launch-on-task, claim) → S3
(full orchestrator) and is deliberately out of v4. v4 is honest observation +
artifacts + reports + task links + the panel UI, over agents the human still
drives. Also out: writing back to the brain kanban files (claim/update tracked in
hub state first; brain-file writes are a later, careful step).

## Config additions (summary)

```toml
# config.toml
[hub]
addr = "127.0.0.1:9100"     # hub listen address (TUI + agents use this)
enabled = true              # TUI tries the hub; falls back to pull-only if down
```
State dir: `~/.local/state/gvardia/hub/` (events.jsonl, projection snapshot).
