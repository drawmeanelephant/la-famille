# Task Plan: Boutique Example Site (`artisanal-ceramics`)

## Task ID
`boutique-example-site`

## Objective
Create a small, polished boutique example site for "Kintsugi & Co. Artisanal Ceramics Studio" placed within `assets/testdata/sites/artisanal-ceramics/` following existing fixture/example conventions, build it through the generator workflow, inspect output artifacts, write fixture tests, and report findings.

## Site Structure & Content Specifications
- `assets/testdata/sites/artisanal-ceramics/content/`:
  - `index.md`: Studio homepage with introduction, links to collection, care guide, and journal.
  - `collection/wheel-thrown-vessels.md`: Collection showcase page with tags (`ceramics`, `wheel-thrown`, `collection`), categories (`crafts`), image reference, and internal links.
  - `care-guide.md`: Ceramic care guide page with tags (`ceramics`, `care`, `guide`), categories (`guides`), and internal links.
  - `journal/2026-07-15-glazing-techniques.md`: Dated journal entry (`date: "2026-07-15"`) suitable for RSS feed inclusion, with tags (`glazing`, `craft`, `journal`), categories (`journal`), asset reference, and internal links.
  - `notes/unrendered-formulas.md`: Unrendered page (`render: false`) demonstrating the documented raw/unrendered behavior.
- `assets/testdata/sites/artisanal-ceramics/assets/`:
  - `ceramic-vase.png`: Sample static asset file.

## Test & Inspection Plan
- Add `cmd/la-famille/boutique_example_test.go` to test building `artisanal-ceramics` and verify:
  1. HTML Output: Rendered pages generated for `index.html`, `collection/wheel-thrown-vessels/index.html`, `care-guide/index.html`, `journal/2026-07-15-glazing-techniques/index.html`.
  2. Unrendered behavior: Confirms `notes/unrendered-formulas/index.html` is NOT rendered to HTML.
  3. Search Index: `search.json` contains entries for all pages.
  4. Taxonomy: Tag (`tags/`) and Category (`categories/`) listing/detail pages generated.
  5. RSS Feed: `feed.xml` contains item for `2026-07-15-glazing-techniques`.
  6. Sitemap & Robots: `sitemap.xml` and `robots.txt` present.
  7. Graph / Backlinks / Meta: `graph.json`, `backlinks.json`, `meta.json` generated.
  8. Static Assets: `assets/ceramic-vase.png` copied to output directory.

## Code Verification
- Run `go test ./...` and `go vet ./...`.
