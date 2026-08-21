# Task Plan: 482-publish-check-required-artifacts

## Issue
#482 — Publishing: validate generated public/ tree with publish-check

## Scope
`internal/publisher/manifest.go` (`Check`):

1. **Core artifact presence** — always require:
   `sitemap.xml`, `robots.txt`, `search.json`, `graph.json`, `backlinks.json`,
   `meta.json`.
2. **Conditional `feed.xml`** — required only when the tree itself proves it
   should exist: any `meta.json` entry with `render:true` and non-empty
   `date`. Keeps the package property of validating the output tree alone
   (no source checkout), and matches the generator's stale-feed deletion.
3. **Staging rejection** — error when a `.staging-*` directory appears inside
   the output tree (atomic-build leftovers).
4. Keep existing behavior: symlinks rejected, cache file rejected, local
   href/src resolution, graph explorer companions.

Tests (`internal/publisher/manifest_test.go`):
- update existing happy-path fixtures to include the required set
- new: missing required file lists the filename
- new: feed.xml demanded iff meta.json shows a dated rendered page
- new: planted `.staging-*` directory rejected
- integration: `internal/generator/publish_contract_test.go` already runs
  `publisher.Check` over a real build output (happy path)

Docs: one sentence in `content/docs/publishing.md` Safe-to-Publish section.

## Static asset pipeline impact
None — publisher validation only; generated trees from correct builds pass
unchanged.

## Verification
- `go test ./internal/publisher/ ./internal/generator/ -v`
- `go test ./... && go vet ./...`
