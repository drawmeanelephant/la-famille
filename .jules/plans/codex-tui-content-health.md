# Codex Plan: TUI Content Health Observability

Enrich the TUI stats screen with content health metrics derived from build metadata and graph structures without altering build semantics or output schemas.

## Objectives
1. Compute content health metrics during site build:
   - Total word count (across rendered pages)
   - Average words per rendered page
   - Top tags frequency distribution
   - Orphaned pages (rendered pages with zero incoming backlinks)
   - Graph node and edge counts
   - Pages missing descriptions or dates
2. Store `ContentHealth` metrics on `generator.BuildResult` and cache struct.
3. Render content health metrics on the TUI Stats screen (`screenStats`).
4. Add unit and model/view tests with representative fixtures.

## Changes
- `internal/generator/health.go`: Data structures and `ComputeContentHealth` logic.
- `internal/generator/health_test.go`: Unit tests for content health computation.
- `internal/generator/generator.go`: Populate `BuildResult.Health` during build.
- `internal/generator/cache.go`: Include `Health` in `buildCache` for cache hits.
- `cmd/la-famille/tui.go`: Display content health metrics on `screenStats`.
- `cmd/la-famille/tui_test.go`: View tests for stats screen with health fixtures.
