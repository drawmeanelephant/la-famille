# Issue #649: NFC/NFD taxonomy terms collide on disk

## Scope and ownership

- `internal/content/metadata.go`: `NormalizeTaxonomyValue` keeps combining marks
  byte-verbatim, so canonically equivalent terms (NFC `café` vs NFD `café`)
  produce two byte-distinct path components that resolve to one file on
  normalization-insensitive filesystems (APFS).
- `internal/generator/generator.go`: `outputClaims.key` folds case only, so the
  collision registry cannot see normalization-only duplicates either.
- Fix: compose normalized terms to NFC at collection time
  (`NormalizeTaxonomyValue`, the single funnel for tag/category identity), and
  fold NFC into `outputClaims.key` so residual normalization-only collisions
  between any writers (content filenames, stubs, assets) are caught like
  case-only collisions.
- `golang.org/x/text` (already an indirect dependency at v0.42.0) supplies
  `unicode/norm`; no new external packages. `go mod tidy` moves it to a direct
  requirement.

## Static-output and compatibility impact

- An NFD term now publishes one canonical NFC archive (`tags/café/`) instead of
  a second byte-distinct archive that collided on disk; both spellings map to
  the same listing and one sitemap/search URL. The rewrite is counted as a
  normalization warning, matching existing punctuation-stripping behavior.
- NFC terms, ASCII terms, and all URLs are unchanged. On case-sensitive but
  normalization-sensitive filesystems, two outputs differing only in Unicode
  normalization are now admitted with a warning instead of silently passing.

## Tests

- `internal/content`: NFD input composes to NFC; a page listing both forms of
  `café` keeps exactly one tag.
- `internal/generator`: a build with an NFC-tagged page and an NFD-tagged page
  publishes one `tags/café/` archive listing both pages and one search/sitemap
  URL — filesystem-agnostic assertions (search.json + archive contents, not
  directory listing).
- `internal/generator`: `outputClaims` treats an NFC/NFD path pair like a
  case-only pair (refused on insensitive filesystems, warned on sensitive).

## Status

- [x] Failing regression tests written.
- [x] NFC composition in `NormalizeTaxonomyValue` and `outputClaims.key`.
- [x] `go test ./...`, `go vet ./...`, `format_check.sh` green.
