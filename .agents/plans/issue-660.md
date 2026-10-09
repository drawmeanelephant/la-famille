# Plan: issue #660 — working screen footer advertises inert Enter/Esc mid-build

## Bug
`cmd/la-famille/tui.go` working-screen footer unconditionally renders
"Press Enter or Esc to return to menu", but mid-build both keys are
swallowed (`q/esc` guard requires `m.workDone()`; `enter` only exits when
`m.workDone()`). Users are trapped until the build finishes while the UI
claims otherwise.

## Approach (per issue's allowed options)
Stop advertising the keys while work is in flight — the least disruptive
fix: make the footer conditional on `m.workDone()`:

- `!m.workDone()` → footer shows diagnostics/help keys only, plus a
  "build in progress" note.
- `m.workDone()` → footer keeps "Press Enter or Esc to return to menu".

Alternative considered (let Esc/Enter exit mid-build): rejected — the
in-flight `workResultMsg` would land on unrelated screens and compound
the state-corruption surface that #661 guards.

## Files
- `cmd/la-famille/tui.go` — `View()` screenWorking footer (one spot).
- `cmd/la-famille/tui_test.go` — regression test: mid-build footer omits
  the Enter/Esc promise; completed-work footer keeps it; Esc remains a
  no-op mid-build (documents kept behavior).

## Breaking changes to static asset pipeline
None — TUI view text only.
