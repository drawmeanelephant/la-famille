# Issue #632: panic on Unicode whitespace before a standalone image

## Scope and ownership

- Own `internal/transform/figure.go` only: the standalone-image detection and
  the `ImageFigure` renderer.
- Root cause: `isStandaloneImage` tolerates whitespace-only `*ast.Text`
  siblings (`bytes.TrimSpace` trims Unicode whitespace such as U+00A0, U+2009,
  U+3000), but `promoteFigures` then asserts `para.FirstChild().(*ast.Image)`.
  When a whitespace-only text node precedes the image, the assertion panics
  (exit 2, no `public/` output). Same unchecked assertion exists in
  `renderImageFigure` for defense in depth.
- Reachable from every path that parses Markdown with the shared engine:
  `build`, `check`, `collectWikiHeadingTargets`, search wiki-link extraction.

## Dependencies

- Branch `t3/fix-markdown` based on `origin/master`. No new external packages;
  goldmark stdlib-style helpers only.

## Static-output and compatibility impact

- Pages that previously crashed the entire build now render; a standalone
  image surrounded by Unicode-whitespace text nodes is promoted to
  `<figure>` the same way an ASCII-whitespace sibling already was.
- No change for paragraphs with meaningful inline content, multiple images,
  links, or Emoji Kitchen glyphs.

## Verification and status

- [x] Failing regression tests in `internal/transform/figure_test.go` covering
  whitespace-only text siblings (incl. U+00A0) before and after a standalone
  image.
- [x] Fix: return the located `*ast.Image` from the standalone check instead
  of re-asserting `FirstChild()`; harden the renderer's assertion.
- [x] `go test ./...`, `go vet ./...`, `format_check.sh`.
- [x] Manual repro: NBSP line + standalone image builds with exit 0 and emits
  `<figure>`.
