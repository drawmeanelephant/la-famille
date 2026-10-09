# Plan: issue #661 — re-entrant "Build Site" corrupts UI state

## Bug
Nothing prevents a second async work task while one is running. Reachable
mid-build via `d` (diagnostics) → `l` (changes) → `q`/`esc` (menu) →
Enter on "Build Site". The second build resets shared work state; the
first build's `workResultMsg` then lands on the second build's state.

## Approach
- Add `working bool` to `model`: set when Build Site / RAG Export
  launches (both share the same async work-state fields, so both get the
  guard), cleared in `workResultMsg`.
- On Enter while `m.working`: redirect to `screenWorking` and re-arm
  `m.spinner.Tick` (its tick chain dies while off the working screen).
  Selecting the task again surfaces the in-flight task — the "visible
  hint" the issue asks for — instead of spawning a second runner.

## Files
- `cmd/la-famille/tui.go` — `model.working`, Build Site + RAG Export
  menu cases, `workResultMsg` case.
- `cmd/la-famille/tui_test.go` — regression test: drive the model through
  the real repro path (build → `d` → `l` → `esc` → Enter on Build Site)
  and assert in-flight state is preserved, not reset.

## Breaking changes to static asset pipeline
None — `generator.Build` invocation count/ordering on disk is unchanged
(output-dir lock still serializes); this only fixes the UI state machine.
