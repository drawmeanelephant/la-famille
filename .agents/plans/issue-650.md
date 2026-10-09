# Task Plan: Issue #650 — Dedupe graph edges, backlinks, inbound count

## Task ID
`issue-650` (GitHub Issue #650)
Branch: `t3/fix-graph-edges` (from `origin/master` at `eb834ca`)

## Objective
graph.json edges, backlinks.json lists, and site-manifest.json
`inbound_link_count` are multisets: a page linking to the same target three
times emits `[['a','b'],['a','b'],['a','b']]`, `"b": ["a","a","a"]`, and
`inbound_link_count: 3`. Make all three speak in distinct pages.

## Context
- `internal/transform/link_transformer.go:93-94,246-247` appends one edge and
  one backlink per link occurrence (Markdown and wiki paths).
- `internal/generator/generator.go:670-676` `collectRenderOutputs` sorts edges
  (by source only) and backlink lists but never dedupes.
- `internal/sitedata/manifest.go:123` `InboundLinkCount = len(backlinks[id])`
  counts occurrences.
- `collectRenderOutputs` runs before every consumer: `renderBacklinksPanels`
  (vault_navigation.go), `WriteGraphFiles` (graph.json/backlinks.json),
  `NewManifest` (inbound_link_count), `graphexplorer.Write`, and
  `ComputeContentHealth` — deduping `bc.g.Edges`/`bc.backlinks` there fixes
  all downstream artifacts at once.
- Edge sort keys on source only, so edges sharing a source keep worker
  insertion order — nondeterministic across builds. Sorting by (from, to)
  makes dedupe possible and output fully deterministic.

## Scope
- `internal/generator/generator.go`: sort edges by both endpoints and compact
  (`slices.Compact`); sort + compact each backlink list. No schema change.
- New test `internal/generator/build_content_test.go` (or new file): build a
  site where `a.md` links to `/b` three times (plus a wiki duplicate), assert
  graph.json has exactly one `["a","b"]` edge, backlinks.json `b == ["a"]`,
  and manifest `inbound_link_count == 1`.

## Static-Output Impact
`graph.json`/`backlinks.json` schema unchanged — lists are deduped, edge order
is now fully sorted (previously nondeterministic for same-source edges).
`site-manifest.json` `inbound_link_count` now counts distinct inbound pages.
Consumers that already deduped see identical semantics.
