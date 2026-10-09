# Issue #648 — check: false negative for root-escaping .md links

Branch: `t3/fix-check-links`

## Finding

`[escape](../../outside.md)` and the percent-encoded form
`[enc](%2e%2e/outside.md)` are skipped by `checker.go:393-395`
(`!filepath.IsLocal || strings.Contains(dest, "%2E%2E")`) and by
`link_transformer.go:74-76`, so the verbatim href ships and 404s, with no
finding.

## Plan

- Checker source-ref path: a decoded target that is not `filepath.IsLocal`
  produces an ERROR finding naming file, destination, and resolved target.
  The raw `%2E%2E` string check is dropped — `url.Parse` already decodes
  `%2e`/`%2E` into `u.Path`, so the decoded-path locality test covers every
  encoded form (also fixes uppercase/lowercase asymmetry).
- Output-ref path (`internal/checker` + `references.go`): resolve relative to
  the linking page's *output* directory (`path.Dir(GetOutputURL(relPath, slug,
  render))`) instead of its source directory — a rendered `blog/post.md` lives
  at `/blog/post/`, so `../x` resolves to `/blog/x`, not `/x` (latent
  off-by-one fixed on the way). `..` segments that climb above the output root
  are clamped at root, matching browser URL resolution (RFC 3986); the clamped
  candidate is then checked against expected outputs, so `../../nope` is now
  reported as a broken link instead of skipped, while `../../real-page`
  correctly resolves.
- `LinkTransformer`: a `.md` link whose decoded target escapes the content
  root is refused — unwrapped to plain text via the existing `excludedLinks`
  mechanism (same handling as publish:false targets) instead of shipping a
  guaranteed-broken href.
- `manifestLinkReference` records escaping links as unresolved links with
  their non-local target so `check --manifest` reports them too.

## Breaking changes to the pipeline

- Rendered HTML: root-escaping `.md` links now render as their anchor text
  without `href` (previously shipped verbatim). No valid output is removed —
  those hrefs always 404'd.
- `check` output: new ERROR findings for root-escaping source links and for
  output links that only resolve after root clamping misses.
