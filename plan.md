# Plan: Build Correctness - Cache Invalidation Matrix

## Goal
Prove that the existing incremental build cache rebuilds when any meaningful input changes and removes stale generated output when source files disappear.

## Proposed Changes
1. **Cache Validation Enhancement:**
   - Modify `internal/generator/cache.go`: `cacheUsable` compares disk state (`generatedFiles(outputDir)`) with `cache.GeneratedFiles` to guarantee invalidation on missing or extraneous output files.

2. **Cache Invalidation Regression Test Suite:**
   - Create `internal/generator/cache_invalidation_test.go` with subtests covering:
     1. Unchanged Markdown produces a cache hit.
     2. Changed Markdown triggers a rebuild.
     3. Deleted Markdown removes its generated page and search/graph metadata.
     4. Changed templates trigger a rebuild.
     5. Changed assets trigger expected output updates (add, modify, delete).
     6. Changed configuration triggers a rebuild.
     7. Removed generated artifacts and orphan files do not survive a later build.

3. **Validation & Verification:**
   - Run `go test ./...`, `go vet ./...`, and repeated test runs for timing sensitivity.

## Potential Breaking Changes
- None. Build output contracts and cache schema (Version 1) remain intact.
