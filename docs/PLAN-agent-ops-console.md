# Agent Operations Console Implementation Plan

> **For Hermes:** Use subagent-driven-development skill to implement this plan task-by-task.

**Goal:** evolve gvardia from a pull monitoring cockpit into a local Agent Operations Console that can discover agent tools, launch supervised runs, and keep the human in control from one TUI.

**Architecture:** keep the existing pull cockpit intact and add local operational primitives underneath it: runner discovery, local tasks, run store, tmux terminal service, prompt rendering, and a runs-first dashboard. The v4 hub/MCP work remains a later milestone after local runs are useful.

**Tech Stack:** Go 1.25+, Bubble Tea v2, existing git/tmux/lazygit shell-outs, TOML config, JSON/Markdown local state under `.gvardia/`.

---

## Current State

gvardia v0.3.0 already has:
- Project and worktree collection through git.
- Claude, Codex, and tmux adapters for pull-based session discovery.
- A read-only brain task browser from `~/Work/sachkov-os/tasks`.
- TUI actions for attach/resume/kill, lazygit/git diff, worktree view, history, reports, and artifacts.
- v4 hub/MCP design docs, but no hub code yet.

Missing for the Agent Operations Console:
- Installed agent tool discovery beyond existing session adapters.
- Runner profiles that define how to launch a tool.
- Local project tasks under `.gvardia/tasks/*.md`.
- Persistent run store under `.gvardia/runs/<run-id>/`.
- tmux-backed launch service that keeps the gvardia TUI open.
- Prompt rendering from task + project context.
- Runs dashboard focused on active/review-needed work.

## Non-Goals For This Plan

- No new agent runtime.
- No provider/model router.
- No autonomous multi-agent orchestration.
- No replacement for lazygit/git diff.
- No hub/hooks/MCP until local run operations are useful.

## Phase 1: Installed Tools + Runner Profiles + `gvardia tools`

**Objective:** establish the launch vocabulary: what agent CLIs exist, whether they are installed, and what runner profiles are available.

**Files:**
- Create: `internal/runners/tools.go`
- Create: `internal/runners/profiles.go`
- Create: `internal/runners/*_test.go`
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Create: `cmd/gvardia/tools.go`
- Modify: `cmd/gvardia/main.go`
- Modify: `cmd/gvardia/cli.go`

**Behavior:**
- Built-in tools: `claude`, `codex`, `gemini`, `opencode`, `aider`, `goose`.
- Tool discovery uses a small `LookPath` seam for tests.
- Config supports custom tool/runner definitions.
- Runner profile validation rejects missing name/tool/command template.
- `gvardia tools --json` prints built-in plus custom tools with installed/missing status and runner profiles.
- Existing `gvardia`, `gvardia agents`, `gvardia projects`, and `gvardia tasks` behavior remains unchanged.

**Verification:**
- Unit tests for fake tool discovery, profile validation, and config parsing.
- CLI smoke via `go run ./cmd/gvardia tools --json`.
- Quality gate: `gofmt -w <changed go files> && go build ./... && go vet ./... && go test ./... -race`.

## Phase 2: Local Project Tasks

**Objective:** add writable local tasks under `.gvardia/tasks/*.md` while keeping the existing brain task source read-only.

**Files:**
- Extend: `internal/tasks`
- Add: local task parser/store tests
- Update: `cmd/gvardia/tasks.go`
- Later UI integration in `internal/ui`

**Behavior:**
- Task source abstraction supports local project tasks and existing brain tasks.
- Local tasks parse frontmatter and Markdown body.
- Local task creation/update is scoped to the selected project.
- Brain tasks remain read-only.

**Verification:** parser/write/reload tests; existing brain task tests remain green.

## Phase 3: Run Store

**Objective:** persist local agent runs independently of hub state.

**Files:**
- Create: `internal/runs`
- Extend: `internal/model` or define run-local types in `internal/runs`

**Behavior:**
- Run creation creates `.gvardia/runs/<run-id>/`.
- Store writes `meta.json`, `prompt.md`, and later `report.md`.
- Store reload reconstructs runs and status from disk.
- Runs link project, task, runner, worktree, tmux target, report, artifacts, and timestamps.

**Verification:** create/update/reload tests with temp projects.

## Phase 4: tmux Terminal Service

**Objective:** launch and control agent runs without swallowing the gvardia TUI.

**Files:**
- Create: `internal/terminal`
- Add tests with fake command runner

**Behavior:**
- tmux is the first terminal backend.
- Launch creates a session/window/pane for the rendered runner command.
- Attach and kill operate by tmux target.
- Missing tmux degrades to a clear error.

**Verification:** command construction tests; no live tmux required for unit tests.

## Phase 5: Prompt Rendering

**Objective:** render a task into an agent prompt with required report paths and project rules.

**Files:**
- Create: `internal/prompts`

**Behavior:**
- Template includes task title/body, project path, workflow requirements, report path, and verification/reporting expectations.
- Output is deterministic and saved to each run's `prompt.md`.

**Verification:** golden or table-driven prompt tests.

## Phase 6: Launch Flow

**Objective:** from the TUI, choose project → task → runner → launch run.

**Files:**
- Extend: `internal/ui`
- Use: `internal/runners`, `internal/tasks`, `internal/runs`, `internal/terminal`, `internal/prompts`

**Behavior:**
- All I/O happens in `tea.Cmd`.
- Launch creates/selects a worktree, writes run files, and starts tmux.
- Dashboard refresh shows the new run while the TUI stays open.
- Errors show banners, not crashes.

**Verification:** message-driven UI tests for select task, select runner, launch success/failure.

## Phase 7: Runs Dashboard

**Objective:** make the TUI feel like a lazygit-grade operations cockpit.

**Files:**
- Extend: `internal/ui`

**Behavior:**
- Panes: Projects, Tasks/Runs, Detail, Diff/Report/Artifacts.
- Active/review-needed runs are visible within 10 seconds.
- Stable keybindings, visible footer, filter/search, empty states, error banners.
- Destructive actions confirm first.
- Diff/review delegates to lazygit/git diff.

**Verification:** UI render tests for empty/error/active/review states; confirmation tests.

## Phase 8: Reports, Artifacts, History

**Objective:** unify transcript-derived reports, git artifacts, and run-local report files.

**Files:**
- Extend: `internal/runs`, `internal/history`, `internal/collect`, `internal/ui`

**Behavior:**
- Run detail reads `report.md`.
- Artifacts include changed files and explicit report/artifact files.
- Ended runs remain visible in history.

**Verification:** run artifact/report tests plus existing history regression tests.

## Phase 9: Hub/MCP Later

**Objective:** once local runs are useful, reintroduce the v4 push channel as an additive live-status layer.

**Files:** follow `docs/PLAN-v4-hub.md`, adjusted to consume the run store.

**Behavior:**
- Hub enriches local run state, but local run store works without it.
- MCP/hooks are not required for Phase 1-8 workflows.

**Verification:** hub acceptance tests from the v4 plan.
