# Uncle Gravity TUI Fix Plan

## Task ID
`uncle-gravity-tui-fix-tasks`

## Objective
Provide a prioritized, structured roadmap of concrete tasks for Uncle Gravity to resolve the TUI Watch Mode activation bug, improve serve build error reporting, surface frontmatter syntax warnings in diagnostics, and add comprehensive unit test coverage.

## Tasks Overview

### Task 1: Fix Watch Mode Activation in `Serve Site` Command
- **Target File:** `cmd/la-famille/tui.go`
- **Issue:** Line 369 checks `if choice == "Serve Site with Watch"`, but the menu label is `"Serve Site"`. When Watch Mode is toggled ON (`m.cfg.WatchMode == true`), `watcher.Watch` is never spawned.
- **Remediation:** Update condition to `if choice == "Serve Site with Watch" || m.cfg.WatchMode` so watcher thread starts whenever Watch Mode is enabled.

### Task 2: Implement Initial Build Check & Error Handling in `Serve Site`
- **Target File:** `cmd/la-famille/tui.go`
- **Issue:** Selecting "Serve Site" currently serves existing static files without performing an initial build check, masking missing templates or build failures.
- **Remediation:** Trigger an initial build check when launching `Serve Site`. If the initial build fails, transition to `screenWorking` or `screenDiagnostics` displaying the error and recovery guidance instead of serving stale/broken outputs.

### Task 3: Surface Frontmatter Parse Failures as Diagnostic Warnings
- **Target Files:** `internal/content/metadata.go`, `cmd/la-famille/tui.go`
- **Issue:** When YAML frontmatter is malformed, `GatherMetadata` silently falls back to raw text rendering without recording a diagnostic warning.
- **Remediation:** Log and return a warning in `BuildResult` / `FileMeta` when frontmatter fallback occurs so TUI records a diagnostic item visible in the Diagnostics drawer (`d`).

### Task 4: Add Regression Unit Tests for TUI Watch & Serve Lifecycle
- **Target File:** `cmd/la-famille/tui_test.go`
- **Remediation:**
  1. Test `Toggle Watch Mode` followed by `Serve Site` spawns `watcherCancel` and triggers rebuilds on file change.
  2. Test `Serve Site` with missing/invalid template triggers initial build error and populates diagnostics.
  3. Test frontmatter parse fallback adds warning to TUI diagnostics.

## Verification Plan
- `go test ./...`
- `go vet ./...`
- Verify TUI manually with `go run ./cmd/la-famille tui`.
