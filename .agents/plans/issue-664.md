# Plan: issue #664 — serverErrorMsg yanks unrelated screens to wrong message

## Bug
`serverErrorMsg` unconditionally switches to `screenWorking` and sets
`workMsg = "Unable to start server"` — even when the user is on an
unrelated screen (e.g. Changes after a successful serve) and the server
failure is a mid-run crash, not a startup failure.

## Approach
- Always record the diagnostic and run `stopServing()` (unchanged).
- Only switch to `screenWorking` when the user is on `screenServe` — the
  screen that would otherwise keep claiming "Server Status: RUNNING" for
  a dead server (preserves the existing
  `TestTUIServerErrorReturnsToVisibleErrorState` contract).
- Set `workMsg` from the actual error (`Server error: <err>`) so the
  headline matches the failure; `workErr` continues to render details +
  recovery guidance.
- On any other screen: stay put; the diagnostics drawer is the
  non-disruptive surface per the issue's acceptance criteria.

## Files
- `cmd/la-famille/tui.go` — `serverErrorMsg` case.
- `cmd/la-famille/tui_test.go` — regression test: `serverErrorMsg` on an
  unrelated screen keeps the user there, records diagnostics, does not
  stamp `workErr`/`workMsg`; serve-screen path keeps working and the
  message reflects the real error.

## Breaking changes to static asset pipeline
None — TUI message routing only.
