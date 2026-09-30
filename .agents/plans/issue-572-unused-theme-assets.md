# Issue #572 — Do not publish unused bundled theme assets

Task ID: `issue-572`
Branch: `droid/fix-la-famille-issue-572`

## Scope

Build currently stages every embedded theme stylesheet and image, even for a
site whose rendered pages use a completely different layout. `init` also
installs the entire theme packet into `assets/`, which the copier then
publishes unconditionally. Keep search and graph bundles as they are, but
publish the bundled theme CSS/images only when generated pages reference them.

## Approach

1. Identify references to the known bundled-theme assets in generated HTML
   after rendering, including per-page layouts, taxonomy/stub pages, content
   images, OG metadata, and deployments under a URL base path.
2. During asset copying, skip unreferenced embedded theme fallbacks and
   byte-identical copies installed by `init`. Keep all other project assets
   authoritative, including modified files at the same paths, and preserve
   the graph/search behavior.
3. Cover custom sites, theme switching, content/metadata references, source
   overrides, and optional graph assets in package tests. Update the frozen
   release-smoke manifest and the publishing documentation.

## Potential breaking changes to the static asset generation pipeline

Unreferenced bundled CSS/images will no longer appear in output, even when
their source files were installed unchanged by `init`. Sites that fetch these
files only at runtime, without a reference in generated HTML or site CSS/JS,
can set `include_unused_theme_assets: true` or keep their own modified copies
in `asset_dir`. The site asset directory is never deleted or modified;
staging and cache fingerprints continue to include it.

## Verification

Focused asset/generator/release-smoke tests passed, as did `go test ./...`
and `go vet ./...`. The frozen release-smoke manifest now omits five
unreferenced bundled theme files while retaining the active theme, search
and graph bundles.
