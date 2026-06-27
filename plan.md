# Execution Plan: Refactor One Seam - JSON Site Data Writing

## Objective
Execute the "Refactor One Seam" routine. Extract the JSON metadata writing logic from `internal/generator/generator.go` into a new package `internal/sitedata` to improve architecture and decouple file writing from the main generation loop.

## Steps Taken (Planned)
1. Create `internal/sitedata/write.go` with a `Write` function that sorts backlinks and writes `graph.json`, `backlinks.json`, and `meta.json`.
2. Create unit tests in `internal/sitedata/write_test.go` to verify this behavior.
3. Update `internal/generator/generator.go` to call `sitedata.Write` instead of calling `jsonutil.WriteJSON` directly.
4. Add a routine report in `content/jules/reports/20260627-refactor-one-seam.md`.

## Potential Breaking Changes
- **No breaking changes to the static asset generation pipeline**. The behavior of writing the JSON metadata files (`graph.json`, `backlinks.json`, `meta.json`) remains exactly the same, merely shifted to a dedicated package. Tests will verify that the JSON output logic remains functionally identical.
