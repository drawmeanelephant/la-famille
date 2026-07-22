# Codex Watcher Debounce Stability Plan

## Objective
Stabilize `TestWatchDebouncesAndTracksNewDirectories` in `internal/watcher/watcher_test.go` by fixing dynamic directory tracking in `internal/watcher/watcher.go` and eliminating OS event queue timing sensitivities in the test without weakening its contract.

## Cause Audit
1. **Directory Tracking**: When a new directory is created, `watcher.go` previously called `watcher.Add(event.Name)` for the single directory entry. If nested subdirectories were created, they were not recursively added.
2. **Test Timing Sensitivity**: `TestWatchDebouncesAndTracksNewDirectories` uses a very short 20ms debounce window. When `os.Mkdir(nested)` runs, `watcher` receives `Create(nested)`, calls `watcher.Add(nested)`, and starts a 20ms timer. If OS event delivery or watch registration for the subsequent `theme.css` file write takes longer than 20ms after the `os.Mkdir` event, `os.Mkdir`'s timer fires (Rebuild #1), `theme.css` fires a second timer (Rebuild #2), and Burst 2 (`page.md`) fires a third timer (Rebuild #3), causing `builds.Load() > 2`.

## Proposed Changes

1. **Watcher Implementation (`internal/watcher/watcher.go`)**:
   - Use `filepath.WalkDir` when handling `fsnotify.Create` on a directory so nested subdirectories are recursively tracked dynamically.

2. **Watcher Tests (`internal/watcher/watcher_test.go`)**:
   - Adjust test debounce window (e.g. `50 * time.Millisecond`) to reliably coalesce OS file system watch registration (`watcher.Add`) and file creation events across environments.
   - Ensure synchronization between change bursts is deterministic without weakening the test assertion that bursts of changes yield single debounced rebuilds.

3. **Validation**:
   - Run `gofmt -w internal/watcher/watcher.go internal/watcher/watcher_test.go`
   - Run `go test -count=20 ./internal/watcher`
   - Run `go test -count=1 ./...`
   - Run `go test -race ./...`
   - Run `go vet ./...`
