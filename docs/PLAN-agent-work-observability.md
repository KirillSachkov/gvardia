# Agent Work Observability Console Plan

> **For Hermes:** implement phase by phase with TDD. Do not turn Gvardia into an
> agent runtime or model router.

**Goal:** make Gvardia a local control tower for AI coding work: launch a work
envelope, observe live status, collect evidence, and review results across
projects.

**Architecture:** Claude, Codex, Gemini, OpenCode, Aider, Goose, and their own
skills/subagents keep choosing their internal workflow. Gvardia owns the
operational envelope around that work: project, objective, worktree, tmux target,
status, events, artifacts, report, diff, checks, and history.

**Tech Stack:** Go, Bubble Tea v2, tmux, git/lazygit, Markdown/JSON under
`.gvardia/runs/<run-id>/`.

---

## Product Boundary

Gvardia does:
- discover installed agent CLIs and runner profiles;
- launch an agent in a project/worktree;
- give the agent a telemetry contract;
- collect status/events/artifacts/reports/check evidence;
- show active/review-needed work clearly;
- delegate diff review to lazygit/git diff.

Gvardia does not:
- choose models/providers;
- decide an agent's internal pipeline;
- run autonomous multi-agent orchestration;
- replace Claude/Codex skills, subagents, planning, or MCP tooling;
- replace lazygit.

## Core Model

```text
Project
  Work Envelope / Run
    objective
    runner profile
    agent CLI session
    worktree
    status
    events
    artifacts
    report
    diff/check evidence
```

The user should mostly think in one phrase:

> "Start work on this objective and show me evidence."

## Run Directory Contract

Every Gvardia-launched run gets:

```text
.gvardia/runs/<run-id>/
  meta.json
  prompt.md
  status.json
  events.jsonl
  artifacts.json
  report.md
  artifacts/
```

`status.json` is the latest small state:

```json
{
  "state": "running",
  "phase": "testing",
  "summary": "Running integration tests",
  "needsReview": false,
  "updatedAt": "2026-07-09T13:00:00Z"
}
```

`events.jsonl` is an append-only activity trail:

```json
{"time":"2026-07-09T13:00:00Z","type":"status","message":"Started inspecting project"}
```

`artifacts.json` indexes useful files:

```json
[
  {"type":"plan","title":"Implementation plan","path":"artifacts/plan.md"},
  {"type":"check","title":"go test ./...","path":"artifacts/go-test.md"},
  {"type":"report","title":"Final report","path":"report.md"}
]
```

## Phase A: Navigation And Project Drawer

**Objective:** make the UI feel like a panel cockpit rather than a table with
many global hotkeys.

**Behavior:**
- `left/right` move focus across panes.
- `tab` / `shift+tab` cycle panes.
- `enter` opens/drills into the focused pane only.
- `esc` backs out.
- `p` toggles the project drawer so the selected project can use the full screen.
- `1..5` keep switching global sections.
- Mouse click sets pane focus where possible.

**Verification:** UI tests for pane focus, drawer toggle, and arrow navigation.

## Phase B: Run Telemetry Store

**Objective:** make run evidence first-class without requiring a hub.

**Behavior:**
- Run creation writes `status.json`, `events.jsonl`, `artifacts.json`, and
  `artifacts/`.
- Store reload reads status, recent events, indexed artifacts, report, and git
  artifacts.
- Missing telemetry files degrade to a useful empty state.

**Verification:** temp-project tests for create/update/reload and malformed file
handling.

## Phase C: CLI Telemetry Helpers

**Objective:** let any agent CLI report progress without MCP.

**Commands:**

```bash
gvardia run status --state running --phase tests --summary "Running tests"
gvardia run event --type status --message "Started implementation"
gvardia run artifact --type plan --title "Plan" --file /tmp/plan.md
gvardia run report --file /tmp/report.md
```

All commands resolve the run directory from `--run-dir` or `GVARDIA_RUN_DIR`.

**Verification:** command tests using temp run dirs.

## Phase D: Prompt And Runner Environment

**Objective:** make launched agents aware of the run envelope.

**Behavior:**
- Prompt explains the observability contract, but does not prescribe the agent's
  internal workflow.
- Runner command receives environment variables:
  - `GVARDIA_RUN_ID`
  - `GVARDIA_RUN_DIR`
  - `GVARDIA_REPORT_PATH`
  - `GVARDIA_EVENTS_PATH`
  - `GVARDIA_ARTIFACTS_DIR`

**Verification:** prompt tests and tmux launch spec tests.

## Phase E: Project Ops Dashboard

**Objective:** show useful work state in one glance.

**Panels:**
- Agents/Runs: current work envelopes and live sessions.
- Activity: latest events and status.
- Evidence: report summary, artifacts, changed files, checks.
- Detail: selected item drill-in.

**UX rules:**
- avoid raw technical labels in the first view;
- show `needs review`, `running`, `done`, `failed` in plain words;
- show terminal/tmux/worktree metadata only in detail;
- raw Markdown report opens separately and is not dumped by default.

**Verification:** render tests for active, review, empty, and telemetry-rich runs.

## Phase F: MCP Later

**Objective:** expose the same telemetry operations as MCP tools only after the
file/CLI contract proves useful.

**Tools:**
- `set_status`
- `append_event`
- `save_artifact`
- `write_report`
- `record_check`

MCP is an additive writer. The run store remains local and works when hub/MCP is
off.

## Quality Gate

Every implementation phase ends with:

```bash
gofmt -w <changed go files>
go build ./...
go vet ./...
go test ./... -race
```
