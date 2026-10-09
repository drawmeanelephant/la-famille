# Issue #662 — Serve Site initial build blocks the event loop

## Task ID
`issue-662`

## Problem
Selecting "Serve Site" runs `generator.Build(m.cfg)` synchronously inside
`Update` (`cmd/la-famille/tui.go:524`), freezing the Bubble Tea event loop for
the whole build; Ctrl+C is queued but unhandled until the build returns. Build
Site already builds asynchronously via `buildProgressCmd` + `workResultMsg`.

## Approach
- Add `servePending bool` to the model.
- On "Serve Site" selection: set `screenWorking` with the same progress state as
  Build Site, set `servePending = true`, and return
  `tea.Batch(buildProgressCmd(m.cfg), m.spinner.Tick)` — the UI keeps ticking
  and Ctrl+C works during the build.
- In the `workResultMsg` handler, after the existing stats/diagnostics logic:
  if `servePending` is set, clear it and:
  - error: show `screenWorking` with `workMsg = "Unable to start serve (initial
    build failed)"` and `workErr` (same failure state as before);
  - success: call `startServing()` — extracted helper doing what the tail of the
    old synchronous branch did (spawn watcher when WatchMode, build the mux via
    `serveMux`, start the HTTP server goroutine, `screen = screenServe`,
    `frame = 0`, return `tickCmd()`).
- Existing serve lifecycle tests are updated to drive the async result message.

## Tests (red first)
- `cmd/la-famille/tui_test.go`: selecting Serve returns immediately on
  `screenWorking` with a non-nil cmd (build not run synchronously), then a
  `workResultMsg` transitions to `screenServe` with a running server; an error
  result leaves `screenWorking` with `workErr` and no server.

## Breaking changes to the static asset pipeline
None. Serve start-up ordering is unchanged (build → watcher → server).
