# Issue #653: search index loses a category whose term equals a tag

## Scope and ownership

- `internal/generator/generator.go` (`processJob`): one `taxonomySeen` set is
  shared across the tags loop and the categories loop, so when a page carries
  the same term as both tag and category (`tags: [go]`, `categories: [go]`),
  the category's archive URL is never appended to `gu`. `/categories/go/` is
  generated on disk but unreferenced from search, breaking the #529 contract
  that every taxonomy badge links to a real archive.
- Fix: track `seen` per kind so a term listed under both taxonomies contributes
  both archive URLs (`g: ["go","go"]`, `gu: ["/tags/go/","/categories/go/"]`,
  preserving the positional `g`/`gu` contract in `internal/search/search.go`).

## Static-output and compatibility impact

- Pages tagged and categorized with the same term gain the missing
  `/categories/<term>/` entry in `search.json` `gu`. `g` lists the term once
  per kind so `g` and `gu` stay index-aligned. No other output changes.

## Tests

- `internal/generator`: build a page with `tags: [go]` and `categories: [go]`,
  parse `search.json`, assert the page item's `gu` references both
  `/tags/go/` and `/categories/go/`.

## Status

- [x] Failing regression test written.
- [x] Per-kind dedupe in `processJob`.
- [x] `go test ./...`, `go vet ./...`, `format_check.sh` green.
