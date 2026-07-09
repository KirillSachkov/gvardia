# Agent Operations Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a global, reliable Gvardia work queue with standalone task/run
storage, full-access Codex launch, cmux handoff, and openable artifacts.

**Architecture:** Keep git, tmux, cmux, lazygit, and agent CLIs as external
tools. Gvardia persists only the work envelope and reconciles it against tmux.
Bubble Tea receives all I/O through commands and renders a global runs queue plus
an inspector.

**Tech Stack:** Go 1.25+, Bubble Tea v2, Bubbles v2, Lipgloss v2, TOML, tmux,
cmux, git, Markdown, JSON, JSONL.

## Global Constraints

- Keep Gvardia a thin router, not an autonomous orchestrator.
- Do not add a database or heavy dependency.
- Do not implement a custom diff renderer or terminal emulator.
- Preserve project-local `.gvardia/` data as legacy read-only input.
- New state defaults to `$XDG_DATA_HOME/gvardia` or
  `~/.local/share/gvardia`.
- Every behavior change follows a red, green, refactor test cycle.
- Every phase ends with focused tests and a clean diff check.

---

### Task 1: Standalone configuration, task store, and run store

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Modify: `internal/tasks/tasks.go`
- Modify: `internal/tasks/tasks_test.go`
- Modify: `internal/runs/store.go`
- Modify: `internal/runs/store_test.go`
- Modify: `cmd/gvardia/main.go`
- Modify: `cmd/gvardia/cli.go`
- Modify: `cmd/gvardia/tasks.go`
- Create: `cmd/gvardia/task.go`
- Create: `cmd/gvardia/task_test.go`

**Interfaces:**
- Produces: `config.Config.DataDir`, `TaskSources`, `DefaultRunner`, `Terminal`.
- Produces: `tasks.LoadGvardia`, `CreateGvardia`, and `UpdateGvardia`.
- Produces: `runs.Store.Root`; `Create` and `LoadProject` use it when set.

- [ ] Write failing tests for XDG defaults, Codex default, terminal defaults,
  global task create/update/reload, and root-scoped run create/load.
- [ ] Run focused package tests and confirm the missing APIs fail to compile.
- [ ] Add the smallest config, file-store, and CLI implementation.
- [ ] Run `go test ./internal/config ./internal/tasks ./internal/runs ./cmd/gvardia`.
- [ ] Run `git diff --check`.

### Task 2: Full-access Codex profile and reliable tmux state

**Files:**
- Modify: `internal/runners/profiles.go`
- Create: `internal/runners/profiles_test.go`
- Modify: `internal/terminal/tmux.go`
- Modify: `internal/terminal/tmux_test.go`
- Create: `internal/terminal/cmux.go`
- Create: `internal/terminal/cmux_test.go`

**Interfaces:**
- Produces: `runners.DefaultProfile(profiles, name)`.
- Produces: `terminal.TmuxService.Inspect(ctx, target) (PaneState, error)`.
- Produces: `terminal.CmuxService.Open(ctx, OpenSpec) error` and
  `terminal.AttachCommand(target) string`.

- [ ] Write failing tests for Codex flags, configured default selection, tmux
  remain-on-exit and pane inspection, and exact cmux workspace arguments.
- [ ] Run focused tests and confirm the expected failures.
- [ ] Implement the command builders and services behind fake runners.
- [ ] Run `go test ./internal/runners ./internal/terminal`.
- [ ] Run `git diff --check`.

### Task 3: Launch, health check, reconciliation, and terminal handoff

**Files:**
- Modify: `internal/ui/commands.go`
- Modify: `internal/ui/commands_test.go`
- Modify: `internal/ui/update.go`
- Modify: `internal/ui/update_test.go`
- Modify: `internal/ui/modal.go`
- Modify: `internal/prompts/prompts.go`
- Modify: `internal/prompts/prompts_test.go`

**Interfaces:**
- Produces: collision-resistant `newRunID`.
- Produces: pure `reconciledRunStatus(run, paneState, paneErr)`.
- Launch writes XDG run state, verifies tmux, opens cmux, and returns a fallback
  attach command when presentation fails.

- [ ] Write failing tests for unique IDs, stale-running reconciliation, default
  profile selection, fallback clipboard behavior, and the final report protocol.
- [ ] Run focused tests and confirm the expected failures.
- [ ] Implement the minimal launch and refresh changes inside `tea.Cmd` paths.
- [ ] Run `go test ./internal/ui ./internal/prompts`.
- [ ] Run `git diff --check`.

### Task 4: Global queue and compact table layout

**Files:**
- Modify: `internal/ui/model.go`
- Modify: `internal/ui/items.go`
- Modify: `internal/ui/items_test.go`
- Modify: `internal/ui/view.go`
- Modify: `internal/ui/runs_test.go`
- Modify: `internal/ui/tabs_test.go`
- Modify: `internal/ui/update.go`

**Interfaces:**
- Produces: `globalRuns()` sorted by attention and updated time.
- Produces: run rows that include project and fit Bubbles cell padding.
- Agents scope defaults to all projects; `s` toggles current project scope.

- [ ] Write failing render/model tests for cross-project ordering, project
  column visibility, default global scope, scope toggle, and width budgeting.
- [ ] Run focused UI tests and confirm failures.
- [ ] Implement global queue aggregation and compact columns.
- [ ] Run `go test ./internal/ui`.
- [ ] Run `git diff --check`.

### Task 5: Artifact browser and external diff routing

**Files:**
- Modify: `internal/ui/model.go`
- Modify: `internal/ui/modal.go`
- Modify: `internal/ui/update.go`
- Modify: `internal/ui/view.go`
- Modify: `internal/ui/commands.go`
- Modify: `internal/ui/artifacts_test.go`
- Modify: `internal/ui/actions_test.go`
- Modify: `internal/ui/commands_test.go`

**Interfaces:**
- Produces: a run-artifact browser with cursor navigation and Enter-to-open.
- Produces: `artifactPath(run, artifact)` with run-directory containment.
- Produces: cmux-first diff and Markdown opening with pager fallback.

- [ ] Write failing tests for artifact path resolution, browser navigation,
  render output, open command selection, and absence of changed-file lists.
- [ ] Run focused UI tests and confirm failures.
- [ ] Implement the modal and external viewer commands.
- [ ] Run `go test ./internal/ui`.
- [ ] Run `git diff --check`.

### Task 6: Documentation, migration notes, and full verification

**Files:**
- Modify: `README.md`
- Modify: `docs/DESIGN.md`
- Modify: `docs/ROADMAP.md`
- Update: `/Users/dev/Work/sachkov-os/wiki/gvardia-agent-work-observability.md`
- Update: `/Users/dev/Work/sachkov-os/tasks/active/2026-07-09-gvardia-agent-ops-ux-reliability.md`

**Interfaces:**
- Documents config, storage, launch, attach, artifact, diff, task CLI, fallback,
  and the agent evidence contract.

- [ ] Update user and architecture documentation with exact commands.
- [ ] Run `gofmt -w` on changed Go files and confirm `gofmt -l .` is empty.
- [ ] Run `go test ./... -race`, `go vet ./...`, and `go build ./...`.
- [ ] Run a real tmux smoke test that launches, inspects, attaches by command
  construction, and kills a disposable session.
- [ ] Run a TUI render smoke at representative widths and inspect the capture.
- [ ] Review `git diff` against every acceptance criterion and fix gaps.
- [ ] Commit project changes, update durable wiki knowledge, close the brain
  task, and push the brain commit.

