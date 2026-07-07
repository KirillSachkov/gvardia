# gvardia — Go & Bubble Tea conventions

Lean, high-signal rules for this repo. The goal is to **not write "C#-in-Go"** and
to use Bubble Tea the way it's meant to be used. Scoped on purpose — this is a
small focused TUI, not a service; don't grow this into a rulebook.

## Mindset (coming from .NET)

- **Errors are values, not exceptions.** Return `error` as the last result; never
  `panic` for expected failures (only for truly unrecoverable init). Handle at the
  boundary (the UI/command layer), not everywhere.
- **No DI container, no inheritance.** Pass dependencies explicitly (constructor
  funcs `New…`). Compose with structs + small interfaces. There is no `services/`
  ceremony — a function is often the right unit.
- **Accept interfaces, return structs.** Define an interface where it's *consumed*
  (e.g. `Adapter` in the collector), not next to the implementation.
- **Zero values should be useful.** A freshly-declared struct should be usable or
  clearly require a `New…`. Avoid nil-pointer ceremony.

## Layout

- `cmd/gvardia/main.go` — thin (< ~50 lines): parse flags/config, wire, run.
- `internal/…` — everything private (compiler-enforced un-importable). One package
  per responsibility, **named by what it provides**: `config`, `model`, `collect`,
  `adapters`, `ui`. No `utils`, `helpers`, `common`, `base` — those are dumping
  grounds and a code smell.
- No `pkg/` unless something is *deliberately* public API for others to import.
- Don't over-nest (`internal/collect/git/status/v1/…`). Flat and descriptive wins.

## Errors

- Wrap with context: `fmt.Errorf("collect worktrees in %s: %w", repo, err)`.
- Typed/sentinel errors only when callers branch on them (`errors.Is/As`).
- A failing adapter/collector for one repo must **not** abort the batch — collect
  the error, skip that item, surface a banner. Partial fleet > no fleet.

## Concurrency

- **`context.Context` is the first parameter** of anything cancelable or that does
  I/O (`Sessions(ctx)`, collectors). Honor cancellation.
- Fan-out with `golang.org/x/sync/errgroup` + a bounded semaphore
  (`min(16, runtime.NumCPU()*2)`), not unbounded goroutines — we shell out to git
  dozens of times per refresh.
- Every goroutine you start must have a way to stop. No shared mutable state
  without a mutex or (preferred) passing data over channels / returning it.

## Bubble Tea (the Elm loop — this is where .NET habits break things)

- **`Update` and `View` must be fast and non-blocking.** Never run `git`, filesystem
  walks, or adapter calls inside them. They react to messages and render; that's it.
- **All I/O happens in a `tea.Cmd`** that returns a typed `Msg`. Collectors/adapters
  run as commands; their results arrive as messages handled in `Update` via a type
  switch (`case worktreesMsg:`, `case sessionsMsg:`, `case errMsg:`).
- **Periodic refresh:** `tea.Every(interval, …)` returns a `tickMsg`; re-issue the
  same command on receipt to keep looping (Bubble Tea ticks once per call).
- **Shell out to interactive programs** (`lazygit`, `tmux attach`, `claude/codex`)
  via `tea.ExecProcess(exec.Command(…), func(err error) tea.Msg{…})` — it pauses the
  TUI, hands over the terminal, and resumes cleanly. Do **not** try to embed them.
- **Model holds data only**; **View is pure formatting** (Lipgloss). Keep selection
  indices and filter text in the model; derive everything else in `View`.
- Prefer Bubbles components (`list`, `table`, `viewport`, `textinput`) over hand-
  rolled rendering. Reference: `charmbracelet/bubbletea-app-template`.

## External CLIs

- Wrap each (`git`, `lazygit`, `tmux`, agent CLIs) behind a small interface so tests
  fake them. Use `exec.CommandContext`.
- **Degrade gracefully:** probe availability once; if `lazygit`/`tmux`/an agent CLI
  is missing, disable that path with a banner — never crash.

## Testing

- **Table-driven tests** are the default idiom.
- Test parsers against **captured fixtures** (real `git worktree list --porcelain`,
  `claude agents --json`, codex JSONL) — no live git needed. One integration test
  against a temp repo with a couple of worktrees.
- UI flows with `teatest` (Bubble Tea's harness).
- Run `go test ./... -race`.

## Tooling & style (non-negotiable)

- `gofmt` + `goimports` always. `golangci-lint` + `go vet` green before commit.
- Exported identifiers get a doc comment starting with the identifier name.
- Minimal deps: stdlib + Charm (bubbletea/bubbles/lipgloss) + a TOML lib. Justify
  anything else.
- Small functions; if a function needs a comment to explain a block, extract it.

## References

- Effective Go — https://go.dev/doc/effective_go
- Go module layout — https://go.dev/doc/modules/layout
- Bubble Tea — https://github.com/charmbracelet/bubbletea (`Every`, `ExecProcess`)
- App template (CI/GoReleaser/lint) — https://github.com/charmbracelet/bubbletea-app-template
