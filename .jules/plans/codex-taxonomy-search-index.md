# Codex Plan: Taxonomy Search Index Integration

## Goal
Fix the integration gap where generated taxonomy pages exist in the site and sitemap but do not appear in `search.json`.

## Scope & Target Files
- `internal/taxonomy/taxonomy.go`: Return output paths and search metadata items (`search.Item`) for taxonomy group index pages and individual taxonomy term pages.
- `internal/taxonomy/taxonomy_test.go`: Update unit test assertions for new return values.
- `internal/generator/generator.go`: Receive taxonomy search items and append them to `searchIndex` before writing `search.json`.
- `internal/generator/generator_test.go`: Integration tests verifying taxonomy search entries in `search.json`.

## Requirements Enforced
- Deterministic ordering of `search.json` items.
- Clean URLs matching actual output paths (`/tags/index.html`, `/tags/<tag>/index.html`, `/categories/index.html`, `/categories/<category>/index.html`).
- Titles ("Tags", "Categories", "Tag: <tag>", "Category: <category>") and taxonomy terms in `Tags` field are searchable.
- Empty taxonomy groups produce no search entries.
- `render: false` pages do not produce invalid taxonomy search entries.
- Content page search behavior remains completely unchanged.
