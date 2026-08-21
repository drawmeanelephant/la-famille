# Task Plan: 474-publishing-validation-summary

## Issue
#474 — Publishing: concise validation summary for content health
Milestone 4. Content Quality and Publishing
Companion #483 owns footer presentation; this owns categories/wiring.

## Goal
`la-famille check` answers "can I publish?" in one screenful by surfacing all four roadmap categories: broken links, missing metadata, asset health, and orphaned pages.

## Current state (verified on main, Aug 2026)
- `internal/checker/checker.go` already covers: frontmatter syntax, date format (YYYY-MM-DD), tag format (^[a-z0-9-]+$), render/slug combos, slug validity, broken internal .md links, output-path collisions, asset health (optional, --asset-health).
- Findings already sorted deterministically by File→Line→Level→Message inside checker.Validate.
- Not covered: missing title/description warnings, orphaned pages (only at build via generator.ComputeContentHealth), categories format validation.

## Tasks
- [x] Wire orphan detection into `check`: reuse ComputeContentHealth logic or extract orphan logic so check reports zero-inbound rendered pages without full build. Do not require building to output dir.
- [x] Decide orphan definition once: ComputeContentHealth flags every zero-inbound rendered page including `index`, while `content/docs/publishing.md` exempts `index`. Pick one rule (exempt `index`), apply everywhere, document in code + keep publishing.md as source of truth.
- [x] Missing title warning: `<meta property="og:title">` falls back to filename, warn so authors know they rely on fallback. Align with health: title == "".
- [x] Missing description warning where no frontmatter description is set (align with ContentHealth.MissingDescriptions).
- [x] Categories format check mirroring tags (`^[a-z0-9-]+$`), warn on uppercase/spaces since taxonomy pages are generated from categories. Reuse `content.NormalizeTaxonomyValue` logic.
- [x] Categorize findings so per-category counts feed #483 footer (broken links / missing metadata / asset health / orphans). Preserve exit semantics: exit 0 on warnings-only, exit 1 on errors (cmd/la-famille/check.go already errors when ErrorCount()>0).
- [x] Add unit tests within same package directories (checker, generator, check command).

## Static asset generation pipeline impact
- None. `check` is dry-run validation; does not write to public/. Changes to `ComputeContentHealth` affect build health metric (orphan count reduces by 1 when index is orphan) and TUI health display. No output file collisions or template changes.

## Design
### Orphan rule
- Adopt Explorer Orphan Rule from `content/docs/publishing.md`: rendered page is orphan iff inbound == 0 && id != "index". Raw render:false pages never orphan candidates (ids carry .md suffix, not matched). Sites using render:false for homepage must provide inbound link.
- Update `internal/generator/health.go:ComputeContentHealth` to exempt id=="index".
- In checker, implement `collectOrphans(fileMap)` by walking markdown AST similarly to broken-link detection but collecting inbound map for existing rendered targets, then flag zero-inbound.

### Categories for Findings
- Extend `Finding` with `Category string` field. Define constants:
  - `CategoryBrokenLink = "broken_links"`
  - `CategoryMissingMetadata = "missing_metadata"` (title, description, tag/category format, date etc? But tag format currently categorized as missing_metadata; keep consistent)
  - `CategoryAssetHealth = "asset_health"`
  - `CategoryOrphan = "orphan"`
  - Also consider `CategoryCollision` or others map to broken_links? Decision: collisions are errors not in four categories; categorize as broken_links? Actually collisions are output path collisions—treat as broken_links? Alternative: keep existing categories and add "collision" but footer only needs 4. For footer we map to those 4; collisions could be counted as broken_links or separate. Simpler: collisions -> broken_links OR new "publish" category but footer spec says 0 errors. We will map collisions to broken_links for count purposes, or keep uncategorized but not feed footer. Need distinct grouping for footer: footer spec shows "orphaned pages, missing description" counts derived from health. Broken link count is ErrorCount, but includes collisions. Proposal: map findings to 4 categories explicitly:
    - broken_links: broken internal link, output path collision
    - missing_metadata: missing title, missing description, malformed tag, malformed category, invalid date/frontmatter? But frontmatter errors are critical. Maybe invalid date/frontmatter -> missing_metadata as well.
    - asset_health: all asset findings
    - orphan: orphaned pages

### Validation wiring
- In `checker.Validate`, after fileMap gathered, for each file inspect `meta.Title`, `meta.Description`, `matter.Tags`, `matter.Categories` (need to parse categories from rawMatter similarly to tags). For categories, apply same validTagRegex.
- Title missing: LevelWarn, category missing_metadata, message `missing title: <meta property="og:title"> will fallback to filename`.
- Description missing: LevelWarn, category missing_metadata, message `missing description: page will use default_description`.

### Testing
- Update `internal/checker/checker_test.go`: add tests for missing title/description, categories validation, orphan detection, category counts, deterministic order with orphans.
- Update `internal/generator/health_test.go`: orphan now exempts index (update wantOrphans).
- Run `go test ./...` and `go vet ./...`.

## Verification
- `go test ./...`
- `go vet ./...`
- Manual `go run ./cmd/la-famille check --content ...` on fixture with index + about + orphan.

## Dependencies
- #483 footer will consume Result category counts; ensure Result exposes `CountByCategory` or fields for footer.

## Status
Completed. All tasks implemented, tests passing (`go test ./...`, `go vet ./...`).

### Changes Made
- `internal/generator/health.go:30` — documented Explorer Orphan Rule and exempted `id=="index"` from `OrphanedPages`.
- `internal/checker/checker.go` — added `Category` field + constants (`broken_links`, `missing_metadata`, `asset_health`, `orphan`), `CountByCategory`, missing title/description warnings (WARN, rendered-only), categories format check mirroring tags, orphan detection via markdown graph (zero-inbound rendered, index exempt), categorized all existing findings, added `detectOrphans` helper.
- `internal/checker/checker_test.go` — updated valid fixtures to include `description`, added 5 new tests: `MissingTitleAndDescription`, `CategoriesValidation`, `OrphanDetection`, `OrphanIndexExempt`, `CategoryCounts`.
- `cmd/la-famille/check_test.go` — updated valid fixture to include `description` to satisfy new missing-metadata rule.

### Verification
- `go test ./...` — PASS (all packages)
- `go vet ./...` — clean
- Manual `la-famille check` on fixtures confirms orphan/category/title/description warnings with correct categories and preserved exit codes (WARN → 0, ERROR → 1).
