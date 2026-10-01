# La Famille flagship site

Task: `droid/update-git-and-review-flagship-site`

## Scope

Build a useful, distinctive product and documentation site in this repository.
Use `https://la-famille.filed.fyi` as its canonical origin and the existing
Cloudflare Pages project `la-famille-go` as the publishing target. Keep the
octopus/burger identity, existing documentation URLs, and generator defaults.

1. Add a site-specific configuration and accessible, responsive layout with
   shared navigation, working search, and an intentional homepage.
2. Improve the documentation entry point and binary-first getting-started
   path. Add Escapement and Z.ai as real showcases without removing the
   existing template gallery. Clearly label experimental/source-only features.
3. Add Cloudflare publishing with the existing secret names, validated static
   artifacts, and canonical URLs. Preserve the GitHub Pages publishing path.
4. Add package-local rendering, generated-output, and workflow regression tests.
5. Run formatting, full tests, race/stability tests, vet, artifact validation,
   and desktop-browser checks where available.

## Static-output and compatibility impact

Homepage and documentation presentation change; existing content routes remain
available. New showcase/detail and site-publishing documentation routes are
additive. The flagship configuration must not leak its hostname or theme into
new user projects through generated default configuration. Reuse existing
graph, search, sitemap, and asset contracts without schema or dependency changes.
Publish only content RAG on the Cloudflare site, never source/config bundles or
private caches. Preserve the separate documented GitHub Pages output contract.

## Safety and remote state

The worktree was clean. The repository has `CLOUDFLARE_ACCOUNT_ID` and
`CLOUDFLARE_API_TOKEN` secret names; no secret values were read. Both supplied
Cloudflare URLs returned HTTP 522 during the initial read-only check. This is
not evidence of a local generator error. No pushes, deployments, DNS changes,
Cloudflare account edits, or secret changes were authorized during the initial
implementation. The publication approval below supersedes the push/deploy
restriction, but not the DNS/account/secret restrictions.

## Approved publication, 2026-10-01

The user explicitly selected **Open PR, merge, and deploy**. Publish all
flagship changes in a PR against `master`, wait for CI, merge normally, and
verify the Cloudflare production upload and both public addresses. The user
confirmed the Pages project has no deployments and expects repository-driven
direct uploads, never a manual file handoff.

Update the workflow so pushes to `master` publish the validated artifact.
PR builds remain secret-free and do not publish. Manual dispatch remains an
optional explicit retry. Read the existing project's production branch through
a GET-only Cloudflare API request inside the authorized deploy job and pass it
to Wrangler; do not change that setting or guess `main`/`master`. Add tests for
branch selection, safe credential handling, and event guards. This changes
publishing behavior, not static schemas or starter defaults.

No DNS/domain/secret/account edits, force pushes, or protection bypasses are
authorized. If publication is blocked by repository policy or token scope,
report the exact blocker rather than weakening protections.

## Validation and handoff

- Implemented the separate website configuration, shared navigation/footer,
  responsive local-asset layouts, homepage, docs/quickstart, real-site showcase,
  and gallery that distinguishes active themes from deprecated historical notes.
- Added same-package rendering tests, existing-Markdown-title compatibility,
  isolated full-site output tests, and guarded workflow contract tests. Initial
  missing-layout/config tests failed as expected; all final tests pass.
- Passed `go test ./...`, `go vet ./...`, `go test -race ./...`, and
  `go test -shuffle=on -count=2 -parallel=4 ./...` on the final template changes.
- Passed `format_check.sh` with a disposable golangci-lint v2.14.0 binary.
  The preinstalled v1.64.5 is too old for the repository's Go version; no
  repository linter settings were weakened. Also passed AGENTS/log-scrubbing
  guards, actionlint, JavaScript syntax, and patch whitespace checks.
- The real flagship build produces 201 pages and a valid 243-file artifact
  after adding content-only RAG. Verified no system/config RAG or private
  cache in its published tree. The GitHub Pages subpath fallback also passes
  artifact validation.
- Existing deprecated-layout/report fallback warnings remain. The intentional
  `docs/missing/index.html` example remains the sole published stub. Neither
  is presented as a newly introduced error or a strict stub-free audit.
- A dependency-free Node harness exercised the existing search controller
  against the generated corpus: showcase/quickstart hits, empty/no-match queries,
  Escape, and `/` focus all pass.
- Desktop CDP loaded the local homepage, all referenced assets, the intended
  canonical URL, and accessible navigation. After viewport configuration the
  page reported width/scrollWidth 1314 with no horizontal overflow. Screenshot
  capture timed out. The target then disappeared; creating a recovery tab was
  denied by the desktop bridge. Mobile, graph interaction, and a full visual
  review are **not claimed**. The temporary loopback preview server was stopped.
- Local build artifacts are retained under ignored
  `SUPPORT/flagship-site.yO0i1N/` for inspection. No generated output is committed.
- The initial implementation made no commits, pushes, deployment dispatches,
  DNS changes, account edits, or secret changes. The subsequent user-approved
  publication uses automatic direct uploads on `master` and detects the existing
  Pages production branch; DNS/account/secret changes remain out of scope.

### Publication validation

All final Go tests, vet, race/stability suites, formatting/lint, and actionlint
pass. Five dependency-free Python tests pass with warnings treated as errors:
branch selection, invalid responses, GET-only requests, credential-safe errors,
and configuration rejection. HTTP error responses are explicitly closed.

A fresh build under ignored `SUPPORT/flagship-publish.0xB3lJ/` passes artifact
validation (201 pages, 243 files, the same single intentional example stub).
The externally modified earlier artifact was not overwritten. Git identity
was already configured; repository rules require a PR and the Go Test Suite,
with no required approving review count. Publication status will be recorded
in the PR and final handoff rather than claiming a deployment before it runs.
