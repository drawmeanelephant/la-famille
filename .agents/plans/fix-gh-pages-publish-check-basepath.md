# Task Plan: fix-gh-pages-publish-check-basepath

## Problem

Every `Deploy La Famille Site` workflow run fails at the `Validate publish
artifact` step (publish-check). Reproduced locally:

- `la-famille build --site-url https://drawmeanelephant.github.io/la-famille/`
  renders the site as a GitHub Pages **project site**: every root-relative link
  carries the base path (`/la-famille/assets/...`, `/la-famille/`,
  `/la-famille/tags/`), and the artifact is uploaded as the Pages root.
- `publish-check` resolved those links as literal on-disk paths, so it looked
  for `public/la-famille/...` and reported every page as "references missing
  local file" — a hard failure, so nothing ever deployed.

## Fix

1. `internal/publisher/manifest.go`: `Check(outputDir, basePath string)` —
   new `stripBasePath` removes the configured base path from root-relative
   references before `resolveReference` (handles `/repo/` and `/repo`,
   rejects lookalikes like `/repo-other/`); problem messages still report the
   link the page actually emits.
2. `cmd/la-famille/publish_check.go`: new `--site-url` flag (defaults to
   `cfg.SiteURL`); base path derived via `config.Config{SiteURL: ...}.BasePath()`
   and passed to `publisher.Check`.
3. `.github/workflows/deploy.yml`: the Validate step now passes the same
   `SITE_URL` (`inputs.site_url || vars.SITE_URL || steps.pages.outputs.base_url`)
   to publish-check, matching the Build step.

## Tests

- `internal/publisher/manifest_test.go`: `TestCheckResolvesBasePathReferences`
  (project-site artifact passes with the base path, fails without it, and a
  genuinely missing `/repo/not-there/` still fails) + `TestStripBasePath` unit
  cases. All existing `Check` call sites updated to the new signature.
- Full suite: `go test ./...`, `go vet ./...`, `gofmt -l` clean.
- Local workflow repro: build + rag + publish-check with `--site-url` exit 0.

## Static asset pipeline impact
None — no templates or generated output change; publish-check only.

## Handoff
Uncommitted on `fix/gh-pages-publish-check-basepath`. After merge, the next
`push` to master should deploy; verify the run + `https://drawmeanelephant.github.io/la-famille/`.