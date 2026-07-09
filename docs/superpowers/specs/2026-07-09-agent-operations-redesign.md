# Gvardia Agent Operations Redesign

**Date:** 2026-07-09
**Status:** Approved for implementation

## Goal

Make Gvardia a reliable local console for starting and reviewing coding-agent
work across projects. The default screen is one global queue of managed runs.
The user can launch Codex, keep Gvardia open, attach in a separate cmux
workspace, open produced artifacts, and hand diff review to cmux or lazygit.

## Product boundary

Gvardia owns the work envelope:

```text
Task -> Run -> worktree + tmux session -> status + events + artifacts + report
```

Codex, Claude, or another runner owns its internal planning and execution.
Gvardia does not route models, run an autonomous multi-agent pipeline, embed an
agent terminal, or implement a custom diff engine.

## Decisions

### 1. Global work queue

The Agents tab shows managed runs from every tracked project by default. Rows
are sorted by attention:

1. needs review;
2. failed;
3. running or pending;
4. recently completed or killed.

The table includes project, runner, objective, branch, compact change summary,
and age. The project drawer remains a filter and launch context. A scope toggle
switches between all projects and the selected project.

This replaces the current behavior where the Agents tab silently changes
between project runs and discovered sessions. Raw discovered sessions remain in
History, where attach and resume still work.

### 2. Separate local state

New Gvardia state lives under the platform data directory:

```text
$XDG_DATA_HOME/gvardia/
  tasks/*.md
  runs/<run-id>/
    meta.json
    prompt.md
    status.json
    events.jsonl
    artifacts.json
    report.md
    artifacts/
```

The fallback is `~/.local/share/gvardia`. This keeps `.gvardia/` out of project
repositories. Existing project-local runs are read as legacy data, but all new
runs use the data directory.

Tasks use one Markdown file per task because they remain easy for humans and
agents to inspect, edit, back up, and sync. SQLite is deferred until concurrent
writers or query volume create a measured need. Brain tasks become an explicit
optional source instead of the default merged queue.

### 3. Runner and terminal defaults

Configuration adds:

```toml
data_dir = "~/.local/share/gvardia"
task_sources = ["gvardia"]
default_runner = "codex"

[terminal]
backend = "auto"
open_on_launch = true
focus_new = true
```

The built-in Codex profile uses explicit full-access flags:

```text
codex -a never -s danger-full-access -C <worktree> <prompt>
```

This removes approval prompts for Gvardia telemetry paths outside the worktree.
Users can override the profile when they want a sandboxed run.

tmux remains the persistent session owner. cmux is only the presentation layer.
Launch and attach open a new cmux workspace that runs `tmux attach -t <target>`.
If cmux is missing or fails, Gvardia copies the attach command and keeps the TUI
open.

### 4. Reliable lifecycle

Run IDs include sub-second entropy so two fast launches do not collide. Launch
creates the worktree and run envelope, starts a detached tmux session, enables
`remain-on-exit`, and inspects the pane before reporting success.

Each refresh reconciles active run state with tmux:

- live pane: running;
- dead or missing pane with a report: needs review;
- dead or missing pane without a report: failed;
- explicitly killed run: killed.

The run event log records launch and unexpected exit evidence. Stale `running`
rows are therefore repaired after restart.

### 5. Artifacts and diffs

Changed files are not artifacts. The detail panel shows only a compact change
summary and the hint `d open diff`.

The Artifacts action opens a selectable browser over indexed run artifacts.
Enter opens the selected item:

- Markdown: a cmux Markdown surface when available, then `glow` or a pager;
- text and JSON: a pager;
- images: the platform opener.

Diff review uses this preference order:

1. `cmux diff --branch` when cmux is available;
2. lazygit in the run worktree;
3. `git diff` with delta when installed.

No custom diff renderer is added.

### 6. Task and evidence protocol

Humans and agents can create Gvardia tasks through a small CLI. A run prompt
contains the task body, paths, status commands, artifact commands, and final
report contract. Reports use four useful sections: summary, changes,
verification, and risks or next steps.

An agent may create a follow-up task, but Gvardia does not automatically spawn
another agent. One task can accumulate several runs later, including an
implementation run and a review run, without changing the current runtime.

## Alternatives considered

### Three permanent top lists

Agents, tasks, and worktrees can be visible at once on very wide terminals, but
the relationship between three independent selections becomes ambiguous and
the columns clip at ordinary widths. The selected queue plus inspector layout
uses terminal space better. The tabs keep the other lists one key away.

### Embedded terminal and diff panes

This would resemble Dux or amux, but it would duplicate cmux, tmux, and lazygit,
and it would make Gvardia responsible for terminal emulation and diff behavior.
External surfaces preserve the thin-router boundary.

### SQLite immediately

Agent Deck and Vibe Kanban demonstrate that SQLite works for local agent UIs.
Gvardia currently has one local user, small records, and a strong need for
agent-readable evidence. Markdown and JSON files are the smaller implementation.

## Acceptance criteria

- The Agents tab opens as a global queue across tracked projects.
- Codex is the selected default runner and starts without approval prompts.
- A run that exits immediately cannot stay falsely marked running.
- Launch and attach keep Gvardia open and open a new cmux workspace when cmux is
  available.
- The fallback copies a complete tmux attach command.
- Artifacts can be selected and opened.
- Changed-file lists no longer masquerade as artifacts.
- `d` opens an external diff viewer and the UI explains the delta symbol in
  plain language or removes it.
- New tasks and runs live outside git repositories by default.
- Brain sync is opt-in.
- Existing tests, build, vet, race tests, and a real tmux smoke test pass.

