# SEO & OpenGraph Implementation Plan

## Goal
Implement SEO and OpenGraph metadata tags across the static site generator's configuration, metadata parser, and templates.

## Steps Taken
1. Updated `internal/config/config.go` with `DefaultDescription` and `DefaultOGImage` in the `Config` struct and the `WriteDefault` setup.
2. Updated `internal/content/metadata.go` to add `Description` and `Image` to `FileMeta` and parsed them from frontmatter YAML fields.
3. Updated `internal/page/page.go` to expose `Description` and `Image` in the `Page` object accessible by templates.
4. Updated `internal/generator/generator.go` to correctly map file metadata to the `Page` struct, using the config defaults as fallbacks.
5. Injected `<meta>` tags into `templates/layout.html` and `templates/layout-documentation.html`.
6. Created `internal/generator/generator_test.go` to explicitly test that metadata tags are correctly injected when parsing frontmatter.
7. Regenerated test fixtures inside `assets/testdata/sites/` so their `expected/` HTML files reflect the new template layout with the SEO tags.
8. Successfully ran `go test ./...` and `go vet ./...`.

## Potential Breaking Changes
- `assets/testdata/sites/*/expected` output HTML files have changed structure. Tests will fail if the fixtures are not correctly updated.

---

# CI/CD Pipeline Upgrade Plan

## Goal
Level up the repository CI/CD pipeline by removing third-party dependencies for PR auto-merge, adding a continuous deployment workflow for GitHub Pages, and implementing a nightly job for autonomous PR synchronization.

## Steps
1. Replaced the `pascalgn/automerge-action` with native GitHub CLI in `.github/workflows/jules-ci.yml`.
2. Created `.github/workflows/deploy.yml` for compiling and deploying the `public/` directory via GitHub Pages.
3. Created `.github/workflows/cron-sync.yml` to automatically run `la-famille pr sync` each night.
4. Verified that the syntax of the workflows aligns with the required tool versions.
