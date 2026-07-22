# Codex Search Index Audit Plan

## Objective
Audit and improve La Famille’s client-side search index and search UI wiring:
- Validate page URLs and titles (handling invalid slugs gracefully).
- Index tags and categories (full taxonomy metadata).
- Extract clean headings and useful excerpts.
- Exclude `render: false` pages.
- Ensure minified, safely escaped, deterministic `search.json` output.
- Enhance `assets/js/search.js` UI rendering to consume every supported signal (`t`, `u`, `g`, `s`, `h`).

## Implementation Steps
1. **Search Index Core (`internal/search/search.go`)**:
   - Add `omitempty` to `Tags` and `Snippet` in `search.Item` for cleaner JSON minification when empty.
   - Verify snippet and heading extraction logic.
2. **Generator Wiring (`internal/generator/generator.go`)**:
   - Validate frontmatter `slug` prior to calculating `search.Item` URL to match `relOut`.
   - Merge `meta.Tags` and `meta.Categories` into `search.Item` taxonomy terms.
   - Sort `searchIndex` deterministically by `URL` then `Title`.
   - Ensure `render: false` pages are skipped.
3. **UI Wiring (`assets/js/search.js`)**:
   - Update result template rendering in `search.js` to render tags (`item.g`) as badge pills and matching headings (`item.h`) as sub-headers.
4. **Documentation (`content/docs/search.md`)**:
   - Document taxonomy metadata indexing, heading signals, and UI signal consumption.
5. **Testing & Verification**:
   - Add unit/integration tests in `search_test.go` and `generator_test.go`.
   - Run `gofmt`, `go test -count=1 ./...`, `go test -race ./...`, `go vet ./...`.
