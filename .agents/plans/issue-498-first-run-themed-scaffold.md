# First-Run Experience: Docs and Scaffold Exercise Bundled Themes

## Task ID
`issue-498-first-run-themed-scaffold`

## Objective
Close #498: someone holding only the release archive must discover and try the
bundled theme packet without reading source.

Current state (from #497): `init --theme <name>` selects a bundled layout as
the site default, plain `init` installs all three layouts missing-only, and
`--theme` help lists bare names. Missing: theme docs for binary-only installs,
demo content that exercises themes, one-line descriptions, tests.

## Proposed Changes
- `internal/runtimeassets/assets.go`: add `CuratedLayoutDescriptions` pairing
  each `CuratedLayoutNames` entry with a one-line description (source of truth
  lives next to the packet).
- `cmd/la-famille/main.go`: new `themes` command printing name + description
  from the packet; unknown-theme error reuses descriptions.
- `cmd/la-famille/main.go` (`init`): scaffold missing-only demo content into
  the content dir — an index page plus a second page with per-page
  `layout:` frontmatter switching to another bundled theme, so a fresh site
  demonstrates picking AND switching. Never overwrite existing files.
- `README.md`: quickstart section on picking/switching bundled themes for
  binary-only installs.
- `RELEASE-QUICKSTART.md`: same flow for archive-only operators.
- Tests:
  - `internal/runtimeassets/assets_test.go`: every curated name has a
    non-empty description.
  - `cmd/la-famille/main_test.go`: `themes` command output lists all names +
    descriptions; `init` writes demo content honoring chosen default theme;
    demo scaffold is missing-only (rerun does not clobber user edits).

## Potential Static-Output Impact
None for this repository's own site: scaffolding only runs inside a target
project root given to `init`. The repo site build is untouched.

## Verification
```bash
go test ./...
go vet ./...
```
Manual smoke: unpack-style run of `init --theme layout-octoburger`, `build`,
inspect `public/` renders the octoburger layout and per-page switch works.

## Handoff
- Branch: `issue/498-first-run-themed-scaffold`
- PR references #498; closes it when acceptance boxes pass.
- Status: implemented; `go test ./...` + `go vet ./...` green; binary smoke of
  themes/init/new/build/publish-check passed (octoburger default + per-page
  switch verified in rendered HTML).
- Note: quickstart flow updated — `new index` became `new hello` because init
  now scaffolds content/index.md as a real homepage.
