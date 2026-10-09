# Issue #652 — check: orphan false positive for output-style links

Branch: `t3/fix-check-links`

## Finding

`detectOrphans` (`checker.go:521-602`) counts only `.md` source links and wiki
links as inbound references. A page linked only via its output URL —
`[c](/c)` — is published and reachable but still reported orphaned.

## Plan

- Build an `output → page id` map from the same slug/render-aware
  `GetOutputURL` calls `buildExpectedOutputs` uses.
- In `detectOrphans`, count internal links with no extension or `.html` as
  inbound when they resolve (via `outputTreeTarget` + `normalizeOutputCandidate`
  + the `foo.html` → `foo/index.html` alias, mirroring `expectedOutputFor`) to
  a rendered page's output. Stub outputs are not orphan candidates (not in
  `fileMap`), so no stub allowance is needed here.
- `detectManifestOrphans`: augment `page.InboundLinkCount` with inbound counts
  computed from manifest links whose `GraphTarget` is empty (output-style
  links the transformer never graphs) and `Resolved` is true.

## Breaking changes to the pipeline

Diagnostics only; fewer false-positive orphan warnings. No generated output
changes.
