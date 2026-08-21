# Task Plan: 470 TUI Diagnostics → Next CLI Action Linking

## Task ID
`470-tui-diagnostics-linking` — GitHub Issue #470, Milestone 2 TUI Workflow Polish
Closes: https://github.com/drawmeanelephant/la-famille/issues/470

## Objective
Diagnostics drawer (`d`) is not a dead end — each finding tells the operator what to run next (broken link, missing asset, frontmatter syntax, case-only collision → next CLI/docs action).

## Context
- `internal/content/metadata.go:84-89` already appends `frontmatter parse warning in %s: %v, falling back to raw markdown` to `FileMeta.Warnings` on `frontmatter.Parse` failure; `DecodeFrontmatter` warnings and date invalid warnings similarly stored.
- `internal/generator/generator.go:93-96` `BuildResult.Warnings []string` exists; `build:167-173` aggregates `meta.Warnings` sorted, `591-592` appends `claims.Warnings()` (case-only collision warning) sorted, and `cache.go:37,273` persists warnings. `internal/content` + `internal/generator` warn path already wired.
- `cmd/la-famille/tui.go:119-123` `diagnostic{level, message, source}` has no `nextAction`; `addDiagnostic` extracts `source` via regex but does not compute next CLI hint. `workResultMsg:484-504` adds `diagnostic` for `err` and `ErrorCount` warning but never iterates `res.Warnings` into `m.diagnostics`. So warnings are generated and cached but never surfaced in TUI drawer.
- `internal/checker/checker.go:30-41` findings are grouped with `Level` (`ERROR`/`WARN`); issue says reuse `level` for TUI color (already case-sensitive check `item.level == "warning"` → yellow else red is close, but checker uses uppercase `WARN`/`ERROR`).
- Need linkage: broken internal link → `la-famille check`; missing asset → `la-famille check --asset-health`; frontmatter syntax → `fix frontmatter in <path>`; case-only collision → `docs/issues-420-422.md` (filesystem behavior).

## Scope

### In scope
- `cmd/la-famille/tui.go`:
  - Extend `diagnostic` struct with `nextAction string` (export not needed, keep unexported field).
  - Add helper `diagnosticNextAction(message string) string` mapping:
    - `frontmatter` (parse warning, `frontmatter warning`) → `fix frontmatter in <path>` (extract path from message after `in ` up to `:` or use `source`).
    - `broken internal link` / `broken link` → `la-famille check` (or `go run ./cmd/la-famille check`).
    - `missing referenced asset` / `missing asset` / `unusually large raster` / `suspicious image extension` / `asset case-collision` → `la-famille check --asset-health`.
    - `case-insensitive filesystem` / `case-only` / `would be the same file on a case-insensitive filesystem` → `see docs/issues-420-422.md — case-only collision (same file on case-insensitive FS)`.
    - Fallback: empty or generic recovery via existing `getRecoveryGuidance`.
  - Update `addDiagnostic(level, err)` to populate `nextAction` via helper; also add `addDiagnosticWarning(level, msg string)` or direct loop for string warnings (non-error) that sets level `"warning"` and `nextAction`.
  - In `Update` `workResultMsg` handler, iterate `msg.res.Warnings` (if any) and append to `m.diagnostics` with level `"warning"` and computed `nextAction`. Preserve existing `ErrorCount` warning addition but also ensure warnings are sorted/stabled.
  - Update `View` `screenDiagnostics:857-882` to render `Next: <nextAction>` (or `Action: <nextAction>`) beneath each diagnostic when `nextAction != ""`, otherwise fall back to `getRecoveryGuidance`. Keep `Source: ` line.
  - Normalize `level` comparison for color: accept both `"warning"`/`"warn"`/`"WARN"` caseless as warning (so checker findings if later wired reuse correctly). Current view checks `item.level == "warning"` → change to case-insensitive or normalize stored level to lower.
- `internal/content/metadata.go`: verify warning message already matches required `fmt.Sprintf("frontmatter parse warning in %s: falling back to raw markdown", relPath)` shape — current code includes `%v` err detail; keep as-is but ensure test expectation accounts for `, falling back to raw markdown` suffix. If strict string match needed, adjust to include err or not per issue — keep existing richer message (contains fallback) and document.
- Tests:
  - `internal/content` + `internal/generator`: add `TestGatherMetadata_FrontmatterParseWarningPopulatesWarnings` and `TestBuild_WarningsIncludeFrontmatterAndCaseCollision` verifying `BuildResult.Warnings` contains frontmatter warning and is sorted, and persists via cache.
  - `cmd/la-famille/tui_test.go`: add `TestTUIWorkResultWarningsPopulateDiagnostics` — construct `workResultMsg{res: &BuildResult{Warnings: []string{"frontmatter parse warning in broken.md: ... falling back to raw markdown", "output path warning: ... case-insensitive filesystem"}}, msg: "Build complete"}` and assert diagnostics length, level `"warning"`, `nextAction` substrings (`fix frontmatter`, `issues-420-422`), and rendered View contains those hints.

### Out of scope
- Changing `internal/checker` logic or wiring full `checker.Validate` into TUI.
- New visual theme system (roadmap.md:29 deferred).
- Changing `templates/layout.html` (focus-visible already handled in #469).

## Static-Output Impact
- Build cache shape unchanged (already has `warnings`). Diagnostics rendering change is TUI-only.
- No output tree path collisions — warnings additive.

## Breaking Changes
None — additive diagnostics field; `BuildResult.Warnings` already exported.

## Tests & Verification
- `go test ./...` — new content/generator/TUI tests pass.
- `go vet ./...`
- Manual: `go run ./cmd/la-famille tui` → create `content/broken.md` with malformed frontmatter `---\ntitle: [\n---\n`, build via TUI, open diagnostics (`d`) → warning listed with `Next: fix frontmatter in broken.md`.
- Manual case-only warning: create two pages `one.md` slug Foo and `two.md` slug foo on case-sensitive host → build → diagnostics shows `output path warning ... Next: see docs/...`.

## Steps
1. Add `nextAction` field + mapping helper in `tui.go`.
2. Wire `res.Warnings` into `workResultMsg` diagnostics propagation.
3. Update `View` diagnostics rendering to show next action.
4. Add unit tests in `internal/content`, `internal/generator`, `cmd/la-famille`.
5. Run `go test ./... && go vet ./...`.

## Status
- [x] Plan created
- [x] `diagnostic.nextAction` + helper added
- [x] `workResultMsg` warnings propagation
- [x] `View` shows next action linkage
- [x] Tests covering warnings → diagnostics
- [x] Validation passing
