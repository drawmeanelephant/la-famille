# Plan: Release Readiness End-to-End Smoke Test

## Goal
Implement a deterministic integration smoke test fixture (`release-smoke`) that builds a representative La Famille site through the generator/CLI path and validates key generated artifacts without brittle full-file snapshots.

## Proposed Changes
1. **Fixture Creation:**
   - Create `assets/testdata/sites/release-smoke/content/` containing Markdown pages exercising frontmatter (title, author, date, tags, categories, description), internal links between pages, and static assets.

2. **Fixture Test Runner Guardrail:**
   - Update `cmd/la-famille/fixture_test.go` to skip snapshot comparison if a site fixture directory does not contain an `expected/` subdirectory.

3. **Smoke Test Implementation:**
   - Add `cmd/la-famille/release_smoke_test.go` with `TestReleaseSmoke`:
     - Builds the `release-smoke` site using `generator.Build` with site configuration (`SiteURL: "https://example.com"`).
     - Validates existence and valid schema/structure of:
       - Rendered HTML pages (`index.html`, `about/index.html`, post pages)
       - `graph.json` (nodes & edges)
       - `backlinks.json` (backlink mappings)
       - `meta.json` (page metadata)
       - `search.json` (search index array)
       - Taxonomy pages (`tags/index.html`, `tags/release/index.html`, `categories/tech/index.html`)
       - RSS feed (`feed.xml`)
       - `sitemap.xml`
       - `robots.txt`
       - Canonical URL & OpenGraph metadata (`<link rel="canonical">`, `<meta property="og:...">`)
     - Validates determinism by re-building to a separate directory and comparing output files byte-for-byte.

## Potential Breaking Changes
- None. Static asset generation pipeline behavior remains unchanged.
