# Plan: issue #663 — stale confetti renders frozen over next build's spinner

## Bug
`m.confetti` is never reset when a build starts. The confetti tick chain
dies when the user leaves the working screen mid-animation, so the next
build renders the previous run's frozen confetti above its spinner.

## Approach
- Reset `m.confetti = 0` in the Build Site and RAG Export init blocks
  (both end in `workResultMsg` → confetti, so both must reset).
- Render confetti only when `m.workDone()` — confetti is a completion
  celebration; this guarantees a running build can never show leftover
  frames even if a future path skips the reset.

## Files
- `cmd/la-famille/tui.go` — Build Site / RAG Export init blocks,
  screenWorking view confetti guard.
- `cmd/la-famille/tui_test.go` — regression test: pre-seed `m.confetti`,
  start a build, assert it resets to 0; assert mid-build view has no
  confetti glyph while a completed view does.

## Breaking changes to static asset pipeline
None — TUI state/view only.
