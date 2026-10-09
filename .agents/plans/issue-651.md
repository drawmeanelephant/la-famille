# Task Plan: Issue #651 — Stub referenced_by must use node ids

## Task ID
`issue-651` (GitHub Issue #651)
Branch: `t3/fix-graph-edges` (from `origin/master` at `eb834ca`)

## Objective
Stub graph nodes emit `referenced_by: ["b.md"]` — source relPaths — while
real nodes use node ids (`"b"`). One artifact, two identity spaces. Stubs
must emit the same node-id space used by edges, backlinks.json, and the
manifest.

## Context
- `internal/stub/stub.go:73-79` sets `ReferencedBy: parents` where `parents`
  is `missingFiles[missingRelPath]` — content relPaths collected by
  `LinkTransformer` (`t.CurrentFile`).
- Node id rule is duplicated in three places: `link_transformer.go`
  `sourceID()`, `generator.go:432-435`, `manifest.go:93-96` —
  `strings.TrimSuffix(relPath, ".md")`, except `render: false` pages keep the
  relPath as their id (their edges' source endpoint is the relPath too).
- Fix: convert each parent relPath through the same rule. Add exported
  `transform.NodeID(relPath, meta)` so the rule has one home; reuse it in
  `sourceID()` and the generator's page-id computation.
- `missingFiles` parent lists are already deduped/sorted; converted ids are
  deduped again defensively in case two relPaths ever map to one id.

## Scope
- `internal/transform/link_transformer.go`: add `NodeID(relPath, meta)`.
- `internal/stub/stub.go`: `ReferencedBy` = converted parent ids.
- `internal/generator/generator.go`: reuse `transform.NodeID` for page ids.
- Update `internal/stub/claim_test.go:130` — asserts the buggy `"parent.md"`.
- New regression test in `internal/stub/stub_test.go`: rendered parent →
  `referenced_by ["parent"]`; `render:false` parent → `["raw.md"]`.

## Static-Output Impact
`graph.json` stub nodes change `referenced_by` values from `"x.md"` to `"x"`.
Schema unchanged; values now join correctly with `edges`/`backlinks.json`.
