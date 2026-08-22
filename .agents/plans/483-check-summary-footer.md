# Task Plan: 483-check-summary-footer

## Issue
#483 — Publishing: add summary footer to check command
Milestone 4. Content Quality and Publishing. Depends on #474 (landed as f5206f7).

## Goal
After `la-famille check` prints findings, emit a single summary line so authors can assess content health without scrolling:
```
✓ 0 errors, 3 warnings | 2 orphaned pages, 1 missing description, 0 missing dates
✗ 2 errors, 5 warnings | 1 orphaned page, 3 missing descriptions, 1 missing date
```

## Tasks
- [ ] Add `--summary` bool flag to `check` (default true; suppress with `--summary=false`)
- [ ] Footer goes to stdout after findings; reconcile with "All content validation checks passed." so clean run doesn't print two summaries
- [ ] Counts per category from #474 categorization: `Result.ErrorCount()` / `Result.WarnCount()` and `CountByCategory(CategoryOrphan)`; missing descriptions/dates derived from filtered findings (message substring)
- [ ] Exit codes unchanged: 0 on warnings-only, 1 on errors
- [ ] Unit tests in `cmd/la-famille/check_test.go`: clean site, warnings-only, errors, suppressed summary

## Scope
- `cmd/la-famille/check.go` — flag + footer formatting + pluralization
- `cmd/la-famille/check_test.go` — 4 new tests
- No changes to `internal/checker` categories (reuse existing); missing date count = findings containing "missing date" (currently 0 unless future checker adds it)

## Static asset generation pipeline impact
- None. `check` is dry-run validation; footer is stdout only, does not write to public/. No template or output file changes.

## Design
### Flag
- `checkSummary bool` var; `BoolVar(&checkSummary, "summary", true, "Show summary footer")`

### Footer
- After findings loop, if `checkSummary` true, compute:
  - errors = res.ErrorCount()
  - warnings = res.WarnCount()
  - orphans = res.CountByCategory(CategoryOrphan)
  - missingDesc = count where CategoryMissingMetadata && contains "missing description"
  - missingDates = count where contains "missing date" (or CategoryMissingMetadata + "missing date")
- Symbol: `✓` if errors==0 else `✗`
- Plural helpers: `plural(n, sing, plur)` -> n==1 ? sing : plur
- Format: `✓ 0 errors, 3 warnings | 2 orphaned pages, 1 missing description, 0 missing dates\n` to stdout
- When summary enabled, suppress "All content validation checks passed." (only print when summary==false and len findings==0). On error path, still print footer before returning error (exit 1).

### Verification
- `go test ./...`
- `go vet ./...`
- Manual `go run ./cmd/la-famille check` on fixtures: index+about (clean), plus orphan/warnings, plus broken link errors, plus --summary=false

## Status
Completed. All tasks implemented, tests passing (`go test ./...`, `go vet ./...`).

### Changes Made
- `cmd/la-famille/check.go:3-5,13-18,55-77,74-108` — added `--summary` bool flag (default true), footer to stdout after findings, pluralized symbol/format (`✓`/`✗`, `errors`/`warnings`/`orphaned pages`/`missing descriptions`/`missing dates`), reconciled clean-run to suppress "All content validation checks passed." when summary enabled, counts derived from `ErrorCount()`/`WarnCount()`/`CountByCategory(CategoryOrphan)` and filtered missing description/date messages.
- `cmd/la-famille/check_test.go:60-70,166-401` — updated `TestCheckCommand_ValidContent` to expect footer, added `TestCheckCommand_Summary_WarningsOnly`, `TestCheckCommand_Summary_Errors`, `TestCheckCommand_Summary_Suppressed`, `TestCheckCommand_Summary_Suppressed_WithWarnings`.

### Verification
- `go test ./...` — PASS (all packages, includes 4 new summary tests)
- `go vet ./...` — clean
- Manual `go run ./cmd/la-famille check --content ...` on clean (✓ 0 errors), warnings-only (✓ with orphan+desc), errors (✗ with pipe), and `--summary=false` (prints "All content validation checks passed.") — all match spec.
