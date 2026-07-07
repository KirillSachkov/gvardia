# gvardia v3 — Implementation Plan

> Implements `DESIGN-v3.md`. Execute **phase by phase, in order**; commit after
> each phase; each **Acceptance** is a gate. Conventions (same as v1/v2): Go 1.25+,
> Charm v2 at `charm.land/*`, every phase ends green
> (`go build ./... && go vet ./... && go test ./... -race`), `gofmt` clean,
> table-driven tests vs captured fixtures, no I/O in Bubble Tea Update/View.
> Branch `v3`, commit per phase, merge to main at the end, release v0.3.0.

**Context for a fresh session:** v2 (shipped v0.2.0) made the right pane one row
per work-session. Key files: `internal/ui/{model,update,view,items,commands,modal}.go`,
`internal/collect/{collect,join,assemble,status,changestat,task}.go`,
`internal/adapters/{claude,codex,tmux,adapter}.go`, `internal/history/*`,
`internal/model/model.go`, `cmd/gvardia/*`. The right-pane cursor maps to
`Model.sessionList` (a `[]model.Session`); `selectedSession()` / `worktreeFor()`
are the accessors. Keys are handled in `update.go handleKey` on `tea.KeyPressMsg`
via `msg.String()`.

---

## Phase 1 — Navigation, keybinds, RU layout, detail-pane fix

**Files:** `internal/ui/update.go`, `internal/ui/keys.go` (new), `internal/ui/view.go`,
`internal/ui/model.go`.

- **RU layout:** add `internal/ui/keys.go` with `normalizeKey(s string) string`
  mapping ЙЦУКЕН Cyrillic → Latin for every bound key
  (`й→q ц→w у→e к→r е→t н→y г→u ш→i щ→o з→p х→[ ъ→] ф→a ы→s в→d а→f п→g р→h о→j л→k д→l я→z ч→x с→c м→v и→b т→n ь→m` plus uppercase). In `handleKey`, do `key := normalizeKey(msg.String())` and switch on `key`. Keep special keys (enter/esc/tab/arrows) as-is.
- **Enter drills, d diffs:** add a `level` field to Model (`levelProjects`,
  `levelWork`, `levelDetail`). `enter` increases level (projects→work→detail);
  `esc`/`backspace` decreases. `d` = `enterDiff` (lazygit) on the selected
  session's worktree (moved off `enter`). Focus follows level.
- **Detail-pane fix:** ensure `diffForSelection` always sets the viewport to
  `detailHeader(*s)` immediately (even with empty summary/diff), and that the diff
  viewport has a non-zero height (check `layout()` geometry — likely the bottom
  pane height computes to 0 in some sizes; clamp with `max1`). Add a regression
  test that `render()` of a ready model contains the selected session's summary.
- **Footer** reflects new keys: `enter drill · esc back · d diff · …`.

**Tests:** `normalizeKey` table (`в→d`, `ф→a`, `d→d`, `enter→enter`); message-driven
`enter` raises level and `esc` lowers it; `d` issues an exec command; `render()`
after `ready(t)` contains the summary text.

**Acceptance:** on a Cyrillic layout every key still works; `enter` navigates
deeper and `esc` back; `d` opens lazygit; the detail pane always shows the session
summary (no more blank pane). PTY smoke confirms.

## Phase 2 — Project curation (tracked list, add/remove/create)

**Files:** `internal/config/projects.go` (new), `internal/collect/collect.go`,
`internal/ui/{update,model,modal,view}.go`, `cmd/gvardia/*`.

- `internal/config/projects.go`: `TrackedPath()` = `~/.config/gvardia/projects.toml`;
  `LoadTracked() []string` (expand `~`); `SaveTracked([]string) error` (atomic write).
- `collect.CollectTracked(ctx, runner, paths []string) []model.Project` — treat each
  path as a project root (skip discovery); enumerate worktrees + enrich as today.
  `collectFleet` uses tracked list when non-empty, else falls back to `Collect`
  (roots scan) with a "curate with A" banner.
- TUI (projects level): `A` add — path prompt (reuse the textinput modal pattern
  from `newAgentPrompt`), validate `git rev-parse` succeeds, append + save. `X`
  untrack (confirm modal, remove + save). `C` create — name + parent-dir prompt →
  `git init <dir>` → append + save.
- New msg `projectsChangedMsg` → re-run collect.

**Tests:** `LoadTracked`/`SaveTracked` round-trip (temp dir, `~` expansion);
`CollectTracked` on a temp repo returns exactly that project; add/untrack modal
state machine; create validates non-empty name.

**Acceptance:** with a curated `projects.toml`, only those projects show (no ~/code
scan); `A` adds a repo (persists across restart), `X` untracks (repo untouched on
disk), `C` git-inits + tracks a new project that appears in the list.

## Phase 3 — Worktrees view (agent ↔ worktree)

**Files:** `internal/ui/{model,update,view,items}.go`.

- `w` toggles the L1 right pane between **Agents** (current session rows) and
  **Worktrees**. Worktree rows: glyph (dirty/clean), branch, ahead/behind, change
  stat, and a live-agent marker (`● claude` if a session runs there, else `·`).
- `Model` gains `worktreeView bool` + a parallel `worktreeList []model.Worktree`
  for cursor mapping; `rebuildSessions` builds whichever list is active. Selecting
  a worktree drives the same detail/diff.
- Columns via a `worktreeColumns(width)`; rows via `worktreeRow2(w)`.

**Tests:** `w` flips `worktreeView`; worktree rows built from a project's
worktrees; a worktree with a session shows the agent marker.

**Acceptance:** `w` shows every worktree of the selected project (including ones
without agents), with dirty/ahead-behind/Δ and which agent (if any) runs there;
`d`/detail still work on the selected worktree.

## Phase 4 — Tasks from the brain kanban

**Files:** `internal/tasks/tasks.go` (new), `internal/config/config.go`,
`internal/model/model.go`, `internal/collect/assemble.go` or a new join step,
`internal/ui/{model,update,view,items}.go`, `cmd/gvardia/*`.

- `internal/config`: add `Brain string` (`toml:"brain"`, default `~/Work/sachkov-os`).
- `internal/model`: `Task{ ID, Title, Status, Project, Path string }`.
- `internal/tasks/tasks.go`: `Load(ctx, brainRoot string) []model.Task` — walk
  `<brain>/tasks/{inbox,active,done}/*.md`, parse YAML frontmatter (`title`,
  `status`/dir, `project`, `id`/slug). Use a tiny frontmatter split (no new dep:
  read between the first two `---` lines, unmarshal with BurntSushi/toml? no — it
  is YAML; add a minimal YAML frontmatter parser for `key: value` scalars, or add
  `gopkg.in/yaml.v3` and justify). Prefer a minimal hand parser for flat scalars.
- **Link:** `LinkTasks(projects, tasks)` sets each `Session.Task` to the matched
  task's title/id (branch task-ref → task id, else project match). Also expose the
  full task list to the UI.
- **Tasks view:** `t` opens a tasks screen (list grouped by status, `/` filters,
  filter-by-project of the current selection). Read-only.

**Tests:** frontmatter parse fixture (title/status/project); `Load` from a temp
brain dir returns tasks per status; `LinkTasks` matches a `feat/675-*` session to
task `#675`; tasks-view toggle.

**Acceptance:** `t` lists real tasks from `~/Work/sachkov-os/tasks`; session rows
show the linked task title (not `—`) where a match exists; filtering by project
works.

## Phase 5 — Session handoff (clipboard / no embed)

**Files:** `internal/ui/{commands,update,view}.go`.

- Change `r` (resume) to **hand off**: build the command string
  `cd <worktree> && claude --resume <id>` (or `codex resume <id>`), copy via
  `tea.SetClipboard(cmd)`, and set a transient status "copied — paste in a
  terminal". Keep `a` (attach) as the in-place `tea.Exec` path.
- `handoffCommand(s model.Session) string` (testable, pure).

**Tests:** `handoffCommand` for claude/codex/tmux produces the right string;
pressing `r` sets the status and returns a SetClipboard command (assert cmd
non-nil; the Msg type is `tea` clipboard).

**Acceptance:** `r` copies a runnable resume command to the clipboard (verify by
pasting) and shows the toast; `a` still attaches in place.

## Phase 6 — Artifacts & reports

**Files:** `internal/history/{history,claude,codex}.go`, `internal/collect/changestat.go`
(or new `internal/collect/files.go`), `internal/model/model.go`,
`internal/ui/{view,update}.go`.

- `internal/model`: `Session.Report string`; `Artifact{ Path, Status string }`;
  `Session.Artifacts []Artifact`.
- **Report:** extend the history reader with `reportOf(path)` = the last
  substantive assistant text message from the transcript (claude `type:"assistant"`
  text; codex `response_item role:"assistant"`), cleaned + truncated (~500 chars).
  Attach to live sessions via `SummaryFor`-style lookup and to history sessions in
  `Recent`.
- **Files:** `collect.ChangedFiles(ctx, runner, path, base) []model.Artifact` via
  `git diff --name-status <base>...HEAD`; plus list `<worktree>/.gvardia/reports/*.md`
  if present. Attach in `AssembleLive` for session worktrees.
- **Detail (L2):** render sections — summary · task · report · files (artifacts) ·
  diff stat. Files list scrollable in the viewport.

**Tests:** `reportOf` fixture (claude + codex last assistant message);
`parseNameStatus` (`M\tfile.go\nA\tnew.go` → two artifacts); detail render contains
the report + a file list.

**Acceptance:** the detail pane shows the agent's end-of-session report and the
list of files it changed (artifacts); `.gvardia/reports/*.md` are listed when
present.

## Phase 7 — Release v0.3.0

- Update `README.md` (new keys: enter drill / esc back / d diff / w worktrees /
  t tasks / A add / C create; curation, tasks, artifacts, handoff). Update the
  mockup.
- Merge `v3` → main; `git tag -a v0.3.0`; local `goreleaser release`
  (`GITHUB_TOKEN`/`TAP_GITHUB_TOKEN` = `gh auth token`); delete the failing CI
  Release run the tag spawns; `brew upgrade` to verify.

**Acceptance:** `brew upgrade KirillSachkov/tap/gvardia` → v0.3.0; the cockpit
launches with all v3 features; README quickstart matches.

---

## Testing strategy

Parsers/mappers (normalizeKey, frontmatter, name-status, report extraction) →
table-driven fixtures. Filesystem readers (tracked list, tasks, history) behind
temp dirs. Clipboard/exec/process behind functions returning commands (assert
construction). UI flows → message-driven `Update` tests. Run `-race`.

## Sequencing note

Phase 1 first (fixes active bugs + makes navigation usable). Then 2 (curation)
removes noise. 3–6 add views/data. Each phase is independently shippable; do not
start a phase before the previous is green and committed.
