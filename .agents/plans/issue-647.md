# Issue #647 — check: false positive "broken internal link" for links to stub pages

Branch: `t3/fix-check-links`

## Finding

`internal/checker/checker.go` flags every `.md` link whose target does not
exist as `[ERROR] broken internal link`, but a build turns those same links
into generated "Missing Page" stubs (`internal/stub/stub.go`,
`internal/generator/generator.go:697-708`): the transformer rewrites the href
to the stub's output dir and `stub.GenerateStubs` writes the page.
`publish-check` reports stubs as warnings (`--strict` to fail), so `check`
contradicts the publish contract.

Note: the issue's literal repro uses `[missing](/missing)` — an extensionless
link, which does NOT generate a stub (the transformer only stubs `.md`
destinations); that link genuinely 404s and correctly stays an error. The real
false positive is the `.md`/wiki form (`[missing](missing.md)`, `[[Note]]`).

## Plan

- `buildExpectedOutputs` gains the set of stub outputs: collect every missing
  link target a build would stub (`.md` links with `HasSuffix ".md"` semantics
  matching `LinkTransformer`, plus unresolved wiki targets via
  `UnresolvedWikiTargetPath`) across published pages, and add
  `GetOutputURL(target, "", true)` per target. This also lets output-style
  links (`/missing`) resolve when a stub will exist.
- Also add the always-written `unresolved-notes/index.html` to expected
  outputs (same omission class).
- A source `.md`/wiki link whose target is missing now yields WARN
  ("resolves to a generated Missing Page stub") instead of ERROR — unless the
  output path is already owned by a real expected output (claim-denied case),
  which resolves to real content and stays silent.
- Manifest mode (`check --manifest`): unresolved link with a local `.md`
  Target → stub WARN; unresolved output Target → broken ERROR as before.

## Breaking changes to the pipeline

None to generated output; `check` findings for missing `.md`/wiki targets
downgrade ERROR → WARN, so `check` stops failing on links a build satisfies
with stubs.
