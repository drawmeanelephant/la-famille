# Issue #638: multi-line headings get no anchor id for wiki fragments

## Scope and ownership

- Own `LinkTransformer.addWikiHeadingIDs` in
  `internal/transform/link_transformer.go`.
- Root cause: the candidate heading id is generated from only the *last* line
  of `heading.Lines()`. A heading spanning multiple lines (Setext
  `Foo\nBar\n===`, the only multi-line heading form in CommonMark) renders the
  full text `Foo Bar`, and `WikiHeadingFragment("Foo Bar")` produces
  `foo-bar`, but the generated candidate is `bar`, so `wanted` never matches
  and the heading is emitted with no `id`.
- Fix: build the candidate text from every heading line — trim each line and
  join with a single space — so the id derivation matches the rendered text a
  wiki-link author types. `ids.Generate` already collapses ASCII whitespace
  and punctuation to `-` and drops markup characters, so `Foo *em*\nBar`
  still matches `[[page#Foo em Bar]]`.

## Dependencies

- Branch `t3/fix-markdown` based on `origin/master`. No new external packages.

## Static-output and compatibility impact

- Multi-line headings targeted by `[[page#Heading Text]]` wiki links gain the
  deterministic id the link fragment expects (e.g. `id="foo-bar"`).
- Single-line headings are unchanged: joining one trimmed line yields the same
  bytes the current code passes to `Generate`.
- Note on the issue repro: `## Foo` followed by `Bar` is an ATX heading plus a
  separate paragraph in CommonMark, not a multi-line heading; the fix targets
  real multi-line (Setext) headings. `[[target#Foo]]` now resolves there;
  `[[target#Foo Bar]]` remains unmatched because no such heading exists.

## Verification and status

- [x] Failing regression test in `internal/transform/link_transformer_test.go`:
  Setext multi-line heading + `WikiHeadingTargets` entry gets `id="foo-bar"`.
- [x] Implement all-lines id derivation.
- [x] `go test ./...`, `go vet ./...`, `format_check.sh`.
- [x] Manual repro: `[[target#Foo Bar]]` against a `Foo\nBar\n===` target
  scrolls to `id="foo-bar"`.
