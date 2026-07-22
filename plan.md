# Implementation Plan - Rendered Taxonomy (Tag & Category) Pages

## Summary
Implement and complete rendered tag and category pages using the existing `internal/taxonomy` package, wired directly into the site generator pipeline. Ensure deterministic ordering, correct relative output URLs, HTML label escaping, empty-tag/category filtering, `render: false` page exclusion, and full integration tests.

## Potential Breaking Changes
None. Static asset generation will now output rendered tag pages (under `tags/<tag>/index.html`), category pages (under `categories/<category>/index.html`), and taxonomy listing index pages (`tags/index.html`, `categories/index.html`).

## Proposed Changes

### 1. Content Metadata (`internal/content/metadata.go`)
- Add `Categories []string` to `FileMeta`.
- Support parsing `category` (string or slice) and `categories` (slice or string) in markdown frontmatter.
- Normalize and deduplicate tags and categories per file.

### 2. Taxonomy Package (`internal/taxonomy/taxonomy.go`, `internal/taxonomy/taxonomy_test.go`)
- Complete `GenerateTags` and implement `GenerateCategories` / `GenerateTaxonomies`.
- Filter empty tag/category strings (`""` or whitespace) and ignore tags/categories with 0 rendered pages.
- Exclude pages where `meta.Render != nil && !*meta.Render`.
- Ensure deterministic sorting of tags/categories and page relative paths (`sort.Strings`).
- Deduplicate page references per tag/category.
- Generate main taxonomy index pages (`tags/index.html` and `categories/index.html`) when tags/categories exist.
- Escaped labels (`html.EscapeString`) for titles, headings, and link attributes.
- Return generated relative output paths.
- Add comprehensive unit tests in `taxonomy_test.go`.

### 3. Generator Wiring (`internal/generator/generator.go`, `internal/generator/generator_test.go`)
- Wire taxonomy page generation into `build()`.
- Append taxonomy output paths to `renderedPaths` (included in `sitemap.xml`) and update `result.PageCount`.
- Maintain graph, search index, sitedata, and cache compatibility.
- Add integration tests for rendered tag/category pages in `generator_test.go`.

## Verification Plan

### Automated Tests & Quality Checks
- `gofmt -w internal/content internal/taxonomy internal/generator`
- `gofmt -d .`
- `go test ./...`
- `go vet ./...`
- `go run ./cmd/la-famille build`
