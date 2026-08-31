# init scaffold cleanup — the fresh site reads like a site, not a help file

Task ID: `init-scaffold-cleanup`
Branch: master

## Problem

Fresh `init` scaffolded demo content that read janky:

1. The homepage was a wall of tool-usage instructions ("This homepage was
   scaffolded by la-famille init…", two CLI command code-blocks, an essay
   about the "layout packet") instead of site content.
2. `content/theming.md` claimed "This page does not use the site default
   template" but pinned `layout: layout-octoburger` — which *is* the default
   since octoburger became the global default. The alternative picker
   normalized empty `--theme` to the hardcoded name `layout` and silently
   chose the real default; the test encoded the same stale assumption.
3. The theming demo pinned the plain `layout` theme, so on a two-page starter
   page 2 looked broken rather than like a designed showcase.
4. `internal/runtimeassets.DefaultTemplatePath` still said `layout.html`.

## Changes

- `cmd/la-famille/scaffold.go`
  - `demoContentFiles` resolves the effective default from
    `config.DefaultLayoutPath` when `--theme` is empty.
  - New 3-page starter: `index.md` (clean welcome prose, no commands, keeps
    the `welcome` tag per #529), `about.md` (placeholder intro), and
    `theming.md` (short, intentional switch demo).
  - `alternativeBundledTheme` prefers the flagship octoburger as the showcase
    for any non-octoburger default; for an octoburger default it pins
    `layout-editorial` (designed contrast) instead of the plain look.
- Tests: `scaffold_test.go` default resolution mirrors production; new
  `TestDemoContentReadsLikeASite` regression guard bans CLI-chips/scaffold
  meta-talk from index/about; `main_test.go` init expectations updated.
- `internal/runtimeassets/assets.go`: `DefaultTemplatePath` corrected to
  octoburger (was unused but stale).

## Editorial search/rule overlap fix

The theming demo pins `layout-editorial`, and on fresh sites (no `site_links`
configured) the search box was rendered with `position: absolute` inside the
nav. It was taller than the nav row, so it hung down through the masthead's
4px double rule — the rule crossed straight through the input. `assets/css/
layout-editorial.css` now keeps the search in-flow as a flex child (matching
midnight/terminal/default), so the nav grows to contain it and the double rule
stays a clean divider below. Verified visually with CSS inlined.

## Verification

- Fresh `init` + `check`: 0 errors, 0 orphaned pages, 0 missing descriptions
  (only the siteurl deploy hint).
- Built theming page loads `css/layout-editorial.css`; homepage is octoburger.
- `go test ./...` all green, `go vet` clean, `gofmt` clean.
- Binaries rebuilt across all six targets; native binary init smoke-tested.

No breaking changes to the static asset generation pipeline.