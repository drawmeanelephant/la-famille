# Task Plan: 477-boutique-example-hardening

## Issue
#477 — Publishing: add boutique example site fixture

The fixture (`assets/testdata/sites/artisanal-ceramics/`) and its test already
exist; this task hardens the test into a real end-to-end assertion suite.

## Scope (test-only, plus two comment fixes)
`cmd/la-famille/artisanal_ceramics_test.go`:
1. Fix latent bug: decode `search.json` with the real schema `t/u/g/s/h`
   (`internal/search/search.go`) instead of nonexistent `title/url/content`
   keys; assert entries cover rendered pages and exclude the unrendered note.
2. Assert `feed.xml` contains the dated journal entry specifically
   (`2026-07-15-glazing-techniques`) with RFC1123Z `pubDate`.
3. Assert `sitemap.xml` excludes `notes/unrendered-formulas`, includes
   taxonomy term pages, and has unique URLs.
4. Assert taxonomy term pages exist: `tags/<tag>/index.html`,
   `categories/<category>/index.html`.
5. Parse `graph.json` / `backlinks.json` / `meta.json`; assert non-empty
   nodes/edges/metadata instead of existence-only.
6. Fix stale comments claiming the unrendered note "is indexed into search" /
   "including unrendered notes" — `render:false` pages are excluded from
   `search.json`.

`assets/testdata/sites/artisanal-ceramics/content/notes/unrendered-formulas.md`:
correct the "indexed into search" comment (it is indexed into graph/meta only).

## Static asset pipeline impact
None — no generator changes.

## Verification
- `go test ./cmd/la-famille/ -run TestArtisanalCeramics -v`
- `go test ./... && go vet ./...`
