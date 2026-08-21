# Task Plan: 475-publishing-integration-fixture

## Issue
#475 — Publishing: integration fixture for RSS/sitemap/robots/canonical/search/taxonomy

## Context
The artifact-level hardening from the issue (search schema, feed ordering,
sitemap exclusions, taxonomy term pages, non-empty graph/backlinks/meta) landed
in the #477 PR. What remains unique to #475: prove the whole pipeline works
together **with and without `siteurl`** — the two documented contract modes in
`content/docs/publishing.md`.

## Scope
Refactor `cmd/la-famille/artisanal_ceramics_test.go` into a table-driven test
running the full boutique assertion suite twice:
1. `with siteurl` (`https://kintsugi.example.com`)
2. `without siteurl` (empty)

Mode-specific expectations:
- `feed.xml` item links absolute (`siteURL/...`) vs root-relative (`/...`)
- `sitemap.xml` `<loc>` absolute vs root-relative
- `robots.txt` contains `Sitemap:` line only when `siteurl` set
- rendered HTML has `<link rel="canonical">` only when `siteurl` set
  (matches `release_smoke_test.go` contract)

Everything else (search schema/exclusions, taxonomy term pages, graph/
backlinks/meta content, asset copy, unrendered-page exclusion) asserted
identically in both modes.

## Static asset pipeline impact
None — test-only refactor.

## Verification
- `go test ./cmd/la-famille/ -run TestArtisanalCeramics -v`
- `go test ./... && go vet ./...`
