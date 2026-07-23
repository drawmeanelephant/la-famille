# Clean-User Workflow & Frontmatter Warnings Plan

## Task ID
`clean-user-and-frontmatter-warnings`

## Objective
Implement Priority 2 (Clean-user `init`/`new`/`serve` workflow fixes) and Priority 3 (Frontmatter warning propagation data-flow/API addition).

## Priority 2: Clean-User Workflow Fixes
1. **`serve` CLI Early Exit**: In `cmd/la-famille/main.go`, update `serveCmd` so that if `generator.Build(cfg)` fails during initial build, `serveCmd` returns `fmt.Errorf("initial build failed: %w", err)` immediately rather than proceeding to start the HTTP server.
2. **`init` Template Creation**: Scaffolds `templates/layout.html` during `init` (already in `main.go`).
3. **`new` Content Prefix Trimming**: Normalizes input path when creating a new page (already in `new.go`).
4. **Unit Tests**: Ensure tests in `main_test.go` and `new_test.go` cover `init`, `new`, and `serve` early exit on initial build failure.

## Priority 3: Frontmatter Warning Propagation (`BuildResult.Warnings`)
1. **`generator.BuildResult` API Update**:
   - Add `Warnings []string` field to `generator.BuildResult` struct in `internal/generator/generator.go`.
2. **`internal/content/metadata.go` Update**:
   - When YAML frontmatter parsing encounters syntax errors / unmarshal fallback in `frontmatter.Parse`, record a warning string e.g. `fmt.Sprintf("frontmatter parse warning in %s: falling back to raw markdown", relPath)`.
3. **`cmd/la-famille/tui.go` Integration**:
   - In `workResultMsg` handling or build processing, add any `res.Warnings` to `m.diagnostics` with level `"warning"`.
4. **Unit Tests**:
   - Add test in `internal/content` and `cmd/la-famille/tui_test.go` verifying frontmatter warnings are populated in `BuildResult.Warnings` and surfaced in TUI diagnostics drawer.

## Verification Plan
- `go test ./...`
- `go vet ./...`
