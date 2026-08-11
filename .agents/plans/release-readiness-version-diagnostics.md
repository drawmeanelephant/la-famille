# Release Readiness & Diagnostics Task Plan

## Task ID
`release-readiness-version-diagnostics`

## Objective
Implement Priority 1 (Release Readiness) items from the roadmap:
1. Expose build version and commit information in CLI diagnostic outputs (`la-famille check`) and TUI diagnostics screen.
2. Document supported `build`, `serve`, `watch`, `check`, `new`, and `rag` workflows in one concise getting-started guide path (`content/docs/setup.md` and related quickstart docs).

## Guardrails
- Keep standard Go idioms (`gofmt`, `go vet`).
- Do not break existing CLI output contracts for `la-famille check` (error counts, findings format, exit codes).
- Use `.agents/plans/release-readiness-version-diagnostics.md`, never root `plan.md`.

## Proposed Changes
- `cmd/la-famille/check.go`: Header output for `la-famille check` includes version and commit metadata from `currentBuildInfo()`.
- `cmd/la-famille/tui.go`: Header output for the TUI Diagnostics screen includes `currentBuildInfo()` version string.
- `cmd/la-famille/check_test.go`: Add tests verifying build version header present in `check` command output.
- `cmd/la-famille/tui_test.go`: Add tests verifying version string in TUI diagnostics drawer.
- `content/docs/setup.md`: Update getting started path to concisely cover `init`, `new`, `build`, `serve --watch`, `check`, and `rag` workflows.
- `README.md`: Ensure Quickstart reflects the unified workflow path.

## Verification
- `go test ./...`
- `go vet ./...`
