# Task Plan: 466 Release/Changelog Convention

## Task ID
`466-release-convention` (GitHub Issue #466)

## Objective
Decide and automate the release/changelog convention per `content/meta/roadmap.md:20` ("Decide on a release/changelog convention and keep it automated where practical") to close the last open item in milestone 1. Release Readiness.

Closes: https://github.com/drawmeanelephant/la-famille/issues/466

## Context
- `content/docs/changelog.md:1` is canonical changelog but hand-edited with duplicate `2026-08-13` headings and stale `PR #277` Recent Updates. No automation.
- `.github/workflows/release.yml:165` hardened in `release-447.md` — uses `gh release create --verify-tag --generate-notes` with full provenance check (`resolve_and_checkout` from `.github/scripts/release/tag.sh`). `--generate-notes` auto-generates GitHub Release notes from PRs/commits.
- `RELEASE-QUICKSTART.md:1` documents binary usage but not changelog policy.
- Milestone 1 otherwise complete: #465 smoke test shipped (`release_smoke_test.go:17`), #467 CLI polish shipped (`main.go:140`, `new.go:55`).

## Decision (Hybrid)
GitHub Releases are source of truth; `content/docs/changelog.md` is curated snapshot.

- Primary: `gh release create --generate-notes` generates notes from merged PR titles (conventional-ish but not strictly enforced).
- Curated: `content/docs/changelog.md` is updated at release time (not per-PR) by copying/highlighting the generated notes and adding human context. Site publishes this page.
- No per-PR lint gate blocking merges on missing changelog entry (avoids friction for `jules` nightly maintenance).
- Document this in `RELEASE-QUICKSTART.md` and `content/docs/changelog.md` header.

Rejected: strict per-PR changelog lint (too noisy for automated Jules PRs). Rejected: fully generated root `CHANGELOG.md` (duplicates GH Releases, diverges from `content/docs/changelog.md` contract).

## Scope

In scope:
- `content/docs/changelog.md` — de-dupe duplicate 2026-08-13 section, consolidate sunset + ask/collision entries into one dated block, add header note linking to GitHub Releases and explaining hybrid convention, keep Recent Updates but sort newest first, remove stale duplication.
- `RELEASE-QUICKSTART.md` — add "Release & Changelog Convention" subsection.
- `content/docs/publishing.md` — add one-line reference to changelog convention if needed (optional).
- No code/template/asset changes.

Out of scope:
- Changing `release.yml` tag provenance logic.
- Adding new external dependencies.
- Per-PR `golangci-lint` or `ci.yml` changelog gate (deferred; note as future optional).

## Static-Output Impact
None beyond `content/docs/changelog.md` rendering. No change to `public/` artifact generation other than cleaner changelog page. No breaking changes to `internal/generator` pipeline.

## Ownership
Agent + human review. No parallel agent dependencies.

## Tests & Verification
- `go test ./...` (existing `release_smoke_test.go:17` must still pass)
- `go vet ./...`
- Manual: `go run ./cmd/la-famille build` — verify `public/docs/changelog/index.html` renders correctly, links to Releases.

## Steps
1. Update `content/docs/changelog.md` header + body per decision.
2. Update `RELEASE-QUICKSTART.md` with convention subsection.
3. Run validation.
4. Close #466 or leave for human approval.

## Status
- [x] Milestone created (1. Release Readiness)
- [x] #465/#467 closed as shipped
- [ ] Changelog docs updated
- [ ] Validation passing
