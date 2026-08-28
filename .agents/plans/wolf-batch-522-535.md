# Work through the WOLF review issues backlog (#522–#535)

## Task ID
`wolftiger-batch-522-535`

## Objective
Resolve the issues filed from the black-box review pass (#500): the open
`bug` and `documentation` issues #522–#535 on current `master` (the published
`v0.1.0-prealpha` archive they repro against is older than HEAD).

## Issues already fixed on master (verified, not re-worked)
- **#522** — `publish-check --strict` exists.
- **#527 item 3** — `init --theme` logs the applied theme.

## Bugs fixed (with reproductions + tests)
- **#534** — errors printed 2–3× in inconsistent formats. Root causes: cobra
  wasn't silenced, the config-error path logged a third time before the logger
  was set up, and cobra arg-validation errors fired before `PersistentPreRunE`
  set up slog. Fix: `SilenceErrors: true`, dedupe the config-error log, and set
  up the logger at the top of `main()`. Verified: exactly one
  `time=... level=ERROR msg="Application error"` line per failure.
  Tests: existing CLI tests still pass; `TestServeBindHint`.
- **#530** — unterminated frontmatter silently rendered as body with no warning.
  `adrg/frontmatter` reports it as "no frontmatter"; detect a file that begins
  with an opener yet had nothing parsed. Now a counted warning.
  Tests: `TestGatherMetadataUnterminatedFrontmatterWarns`,
  `TestGatherMetadataClosedFrontmatterDoesNotWarnUnterminated`.
- **#532** — non-ASCII tag normalization (`café �Ľ → caf`) and empty-normalized
  tags (`�Ľ`) dropped while `warnings=` stayed 0. Lossy/dropped taxonomy values
  are now counted warnings naming the file.
  Tests: `TestGatherMetadataCountsLossyTaxonomyWarnings`.
- **#533** — `chmod 555 public/` build exited 0 and stranded
  `.public.previous-*`. `replaceOutputDirectory` now returns warnings; a
  stranded backup is retried then reported as a counted warning naming the
  stale copy.
  Tests: `TestReplaceOutputDirectoryReportsStrandedBackup`.
- **#531** — sitemap/feed/search/canonical emitted raw spaces/unicode in slugs.
  Percent-escape path segments at emission time in the config public-path
  builder (carefully: `URLForOutputPath` feeds `url.URL` whose `.String()`
  re-escapes, so escaping lives only on the string-returning consumers) plus
  the no-siteurl fallbacks in discovery and feed. Verified no double-encoding.
  Tests: config `TestPublicPathForOutputEscapesSegments` + spaced-slug cases,
  discovery `TestWriteEncodesSpacedSlug`, feed `TestLocalURL` cases.
- **#528** — siteurl subpath rebased canonical/og but not asset/link URLs.
  Added `applyBasePath` in the renderer (rebase root-relative `href/src/poster/
  action/srcset` under `BasePath()`, inject `<meta name="la-famille-base-path">`
  for client scripts), make `serve` mount under `BasePath()` with a `/` →
  base redirect, and made client-side fetches (`search.js`,
  watch-mode livereload) base-aware. No-op when base is empty, so root deploys
  and the existing test corpus are byte-for-byte unchanged.
  Tests: `TestApplyBasePath`.
- **#535** (code side) — empty siteurl now produces a site-wide `check` WARN
  (root-relative sitemap `<loc>` is protocol-invalid), cleared once siteurl is
  set. Two checker/CLI "clean pass" tests updated to set a SiteURL.
  Tests: `TestValidate_WarnsOnEmptySiteURL`.
- **#525** (code side) — `serve` bind error now suggests `serve -p <port>` /
  `port:` via `serveBindHint`.
  Tests: `TestServeBindHint`.
- **#527.1** (code side) — `--project-root` help text clarifies the default is
  the current directory.

## Documentation
- RELEASE-QUICKSTART.md: checksum verification calls out digest-vs-filename
  (#523); cache + `.public.previous-*` transient naming incl. #533 doc bit and
  #527.2 cache-location wording; empty-siteurl warning for local-only (#535);
  `serve -p` recovery (#525); "More commands" section (check/rag/ask/tui/pr/
  completion, #526); Tags frontmatter note with `/tags/` reachability caveat
  (#529); bundled themes/`base_path` deploy note (#528 doc side).

## Potential Static-Output Impact
- Sites built with a **subpath `siteurl`** now have their rendered HTML's
  root-relative URLs prefixed with the base path and a
  `la-famille-base-path` meta injected (#528). Domain-root and empty-siteurl
  builds are byte-for-byte unchanged.
- Machine-readable consumers (sitemap/feed/search/canonical) now percent-escape
  non-ASCII/spaced slugs when present; `[a-z0-9-]` paths unchanged (#531).
- `check` output gains a site-wide warning when `siteurl` is unset (#535).

## Verification
- `go test ./...` passes across every package.
- `go vet ./...` passes.
- Real reproductions verified for #534/#530/#532/#533/#531/#528/#535.

## Handoff
Changes are local (working tree on `master`); nothing committed. Suggest
opening one PR per logical group (bugs vs docs) or a single feature branch.
#529 (tags reachability in themes) and project-pages subpath polish beyond the
base-path rebase are candidates for follow-up milestones.