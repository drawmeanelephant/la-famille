# TUI Serve & Watch Lifecycle Refactor Plan

## Task ID
`tui-serve-watch-lifecycle`

## Objective
Refactor the TUI `Serve Site` lifecycle to execute an initial build exactly once before spawning either the HTTP server or the file watcher. If the initial build fails, neither the server nor watcher will start, preserving clean states (`m.server == nil`, `m.watcherCancel == nil`). Add deterministic unit tests with bounded contexts/timeouts for serve/watch lifecycle and cancellation (`q`, `Esc`, `ctrl+c`).

## Proposed Changes

### `cmd/la-famille/tui.go`
1. Update `Serve Site` action in `Update(msg)`:
   - Check if Watch Mode is enabled (`m.cfg.WatchMode` or `choice == "Serve Site with Watch"`).
   - Execute initial build `res, err := generator.Build(m.cfg)` **exactly once**.
   - If `err != nil`:
     - Do NOT start HTTP server.
     - Do NOT start file watcher.
     - Ensure `m.server == nil` and `m.watcherCancel == nil`.
     - Add diagnostic error with `m.addDiagnostic("error", err)`.
     - Set `m.screen = screenWorking`, `m.workMsg = "Serve failed (initial build error)"`, `m.workErr = err`.
   - If initial build succeeds:
     - Set `m.stats = &res`.
     - If Watch Mode is enabled, start watcher goroutine with cancel context `m.watcherCancel`.
     - Start HTTP server goroutine with cancel context `m.serverCancel`.
     - Set `m.screen = screenServe`.
2. Ensure `stopServing()` safely cancels both `m.watcherCancel` and `m.serverCancel`, shuts down `m.server` cleanly within a bounded timeout, and resets lifecycle fields to `nil`.
3. Verify key handlers (`q`, `esc`, `ctrl+c`) trigger `stopServing()` when exiting `screenServe` or quitting.

### `cmd/la-famille/tui_test.go`
1. Test initial build failure on `Serve Site` (e.g. missing layout template):
   - Assert `m.screen == screenWorking`.
   - Assert `m.server == nil`.
   - Assert `m.watcherCancel == nil`.
   - Assert `m.workErr != nil` and recovery guidance present.
2. Test successful `Serve Site` with Watch Mode enabled:
   - Assert `m.screen == screenServe`.
   - Assert `m.server != nil` and `m.watcherCancel != nil`.
   - Test file modification triggers rebuild (`statsUpdateMsg`).
3. Test cancellation via `q`, `Esc`, and `ctrl+c`:
   - Assert `stopServing()` shuts down HTTP listener and watcher cleanly within bounded timeouts (using `getFreePort()`).
   - Assert `m.server == nil` and `m.watcherCancel == nil` post-shutdown.

## Verification Plan
- `go test ./cmd/la-famille -run TestTUI...`
- `go test ./...`
- `go vet ./...`
