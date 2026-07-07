# gvardia v3 — Design (navigable cockpit: curation, worktrees, tasks, handoff, artifacts)

Extends v2 (`DESIGN-v2.md`, shipped v0.2.0). v2 made gvardia a session-centric
monitor. v3 makes it a **navigable review browser** over your curated projects,
with tasks from the brain, session handoff, and per-session artifacts. Still a
monitor/review tool — **not** a multiplexer (herdr's niche), **not** an
orchestrator. This document records the decisions; `PLAN-v3.md` is the executable
plan.

## Feature areas (all approved for build)

1. Navigation & keybinds foundation (+ bug fixes)
2. Project curation (track only selected; add/remove/create from the TUI)
3. Worktrees view (agent ↔ worktree linkage)
4. Tasks from the brain kanban (`sachkov-os/tasks`)
5. Session handoff (continue elsewhere; no embedded terminal)
6. Artifacts & reports (what an agent produced per session)

---

## 1. Navigation & keybinds

**Levels & drill-down.** The cockpit is a three-level browser:
- **L0 Projects** (left list) → `enter` drills to L1.
- **L1 Work** (right table: the project's sessions) → `enter` drills to L2.
- **L2 Session detail** (full-width: summary · task · report · artifacts · diff
  stat) → `esc` / `backspace` goes back up a level. `tab` still cycles focus.

**Keys (revised):**
- `enter` = **drill down a level** (was: open lazygit). `esc`/`backspace` = up.
- `d` = open the selected session's worktree in **lazygit** (was on `enter`).
- `↑↓` / `j k` = move within the focused level (between agents at L1).
- Existing: `h` history, `a` attach, `r` resume/handoff, `n` new, `k` kill,
  `g` gc, `/` filter, `R` refresh, `q` quit.

**Russian keyboard layout.** Keybinds must fire under a Cyrillic layout. Bubble
Tea sends the typed character (`в` for the physical `d` key), and `Key.BaseCode`
(the US-layout key) is only available with the Kitty protocol, so it is not
reliable. Solution: a `normalizeKey` table mapping ЙЦУКЕН Cyrillic runes to their
Latin equivalents for the keys gvardia binds (`й→q ц→w у→e к→r е→t … ф→a в→d …`),
applied in `handleKey` before the switch. Handle both the Latin and Cyrillic key.

**Bug fix — empty detail pane.** In v2 the bottom pane can render empty. Root
cause to confirm during implementation (likely the diff/summary content not being
set on initial selection, or a viewport height of 0). The detail must always show
at least the session summary immediately, then the diff when it loads.

## 2. Project curation

**Model (chosen): only explicitly-tracked projects.** No blanket root scan by
default. Tracked projects live in a gvardia-managed file
`~/.config/gvardia/projects.toml` (separate from the hand-written `config.toml`,
so the TUI can rewrite it without clobbering comments):

```toml
projects = [
  "~/code/education-platform",
  "~/code/software-engineer-tutorial",
]
```

- If `projects.toml` is missing/empty, fall back to scanning `config.roots` (v2
  behavior) so nothing breaks on first run; a banner suggests curating.
- **TUI actions** (on the projects list):
  - `A` add project — a path prompt (textinput) or "add current dir if a git
    repo"; validates it is a git repo; appends to `projects.toml`.
  - `X` untrack — remove from the list (never deletes the repo on disk; confirm).
  - `N`… reuse? `n` is new-agent. Use `C` create project — prompt name + parent
    dir → `git init` → append to tracked list.
- `internal/config`: load/save the tracked list; expose `TrackedProjects()`.
- `collect.Collect` gains a path that takes an explicit project list (skip
  discovery) — each path is a project root; still enumerate its worktrees.

## 3. Worktrees view (agent ↔ worktree)

A per-project **worktrees level/view** so you can see all worktrees (not just the
ones with agents) and how sessions map to them:
- At L1, a toggle (`w`) switches the right pane between **Agents** (sessions) and
  **Worktrees** (every worktree: branch, dirty/ahead-behind, change stat, and the
  agent(s) running in it, if any).
- The join already links sessions ↔ worktrees by cwd; surface it both ways.

## 4. Tasks from the brain kanban

Single task source = the brain, **not** GitLab: `sachkov-os/tasks/{inbox,active,
done}/*.md` (Markdown + YAML frontmatter). Path from config
(`brain = "~/Work/sachkov-os"`, default).

- `internal/tasks`: read task files, parse frontmatter (`title`, `status`,
  `project`, `id`/slug), body optional. Return `[]model.Task`.
- **Tasks view** (`t` opens a tasks pane/screen): browse all tasks grouped by
  status (inbox/active/done), filter by project.
- **Link agent ↔ task**: match a session to a task by (a) branch task-ref
  (`TaskFromBranch`), (b) task `project` == project name, (c) fuzzy title/slug.
  Fill the session row's `task` column and detail with the real task title.
- Read-only in v3 (browse + link). Editing tasks is a later layer.

## 5. Session handoff (continue elsewhere)

Continue a session without embedding a terminal:
- `r` resume → **copy the resume command to the clipboard** via `tea.SetClipboard`
  (`claude --resume <id>` / `codex resume <id>`, prefixed with `cd <worktree> &&`),
  and show a "copied — paste in a terminal" toast. Keeps gvardia a monitor.
- `a` attach keeps the tea.Exec behavior (tmux attach / in-place resume) for those
  who want it. So: `a` = take over here, `r` = hand off elsewhere.

## 6. Artifacts & reports

Per session, show what the agent produced:
- **Report** = the agent's end-of-session summary. Derived from the transcript:
  the last substantive assistant text message (cleaned/truncated). Extend the
  `internal/history` reader to also return `Report` alongside `Summary`.
- **Artifacts (files)** = the session's changed files: `git diff --name-status
  <base>...HEAD` on its worktree. Plus a light convention: if
  `<worktree>/.gvardia/reports/*.md` exists, list those as explicit artifacts.
- Shown in the L2 detail as sections: **summary · task · report · files · diff**.
- `model.Session` gains `Report string` and `Artifacts []Artifact{Path, Status}`.

---

## Config additions (summary)

```toml
# config.toml
roots = ["~/code"]            # fallback scan when no tracked projects
brain = "~/Work/sachkov-os"   # task source (kanban)
# ~/.config/gvardia/projects.toml (managed by the TUI)
projects = ["~/code/…", …]
```

## Non-goals (still)

Embedded terminals / workspace (herdr), GitLab task sync, LLM-generated summaries,
writing to the kanban from gvardia. All layer on the work-session unit later.
