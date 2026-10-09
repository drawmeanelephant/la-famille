# Issue #658 — Watch rebuild failures invisible, clobber stats

## Task ID
`issue-658`

## Problem
On a rebuild error the watcher invokes `onBuild(res)` with a zero `BuildResult`
(`internal/watcher/watcher.go:70-74`). The TUI's `statsUpdateMsg` handler stores
it unconditionally (`cmd/la-famille/tui.go:620-624`): stats silently reset to
zero and no diagnostic is recorded, so a broken rebuild looks like an empty site.

## Approach
- Change the watcher `onBuild` callback to `func(generator.BuildResult, error)`
  so failures are reported to subscribers (`internal/watcher/watcher.go`).
  - Success: `onBuild(res, nil)`. Failure: `onBuild(res, err)`; still no
    `BroadcastReload` (existing contract — browser keeps the last good output).
  - Callers: `main.go` passes `nil` (unchanged); watcher tests updated.
- TUI: extend `statsUpdateMsg` with `err error`; the watcher callback forwards it.
- TUI `statsUpdateMsg` handler: on `err != nil`, keep the last good stats and
  append an error diagnostic naming the failure instead of storing the zero
  result.

## Tests (red first)
- `cmd/la-famille/tui_test.go`: deliver `statsUpdateMsg` carrying an error after
  a good build — assert `m.stats` unchanged and a diagnostic was appended.
- `internal/watcher`: drive `watch` with a failing build func and assert
  `onBuild` receives the error.

## Breaking changes to the static asset pipeline
`watcher.Watch`/`watch` callback signature gains an error parameter. Internal
package; all in-repo callers updated. No output-path changes.
