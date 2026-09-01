# Quick wins: theme-name mismatch + midnight font stack

Follow-up to `theme-quality-pass.md` / the 2026-08-31 theme audit.

## Problem 1 — `theme: dark` silently falls back to retro
- `config.DefaultConfig.Theme = "retro"` and `theme.css` defines palettes for
  `retro, ink, sepia, slate, moss` on `[data-theme=...]`.
- Tests and real configs use `theme: dark` (e.g. `cfg.Theme = "dark"` in
  generator tests). `data-theme="dark"` matches no palette, so the page
  renders with the `:root` retro palette while the user thinks they picked a
  dark theme.

### Fix
Add `[data-theme="dark"]` as an alias of the `ink` palette in
`assets/css/theme.css` (same selector group as `ink`). No Go changes.

## Problem 2 — midnight lists an unloaded font
- `assets/css/layout-midnight.css` `--mn-sans` starts with `"Inter"`, but the
  theme never loads Inter (framework-free by design), so it silently falls to
  Segoe UI/system.

### Fix
Remove `"Inter"` from the stack so the declared stack matches reality.

## Breaking changes to static asset pipeline
None: additive selector in an existing embedded CSS file; embedded asset
bytes change but no file names, paths, or template contracts change.

## Verification
- `go test ./...`
- `go vet ./...`
- Optional: build a page with `theme: dark` and confirm the ink palette
  applies (data-theme="dark" now matches).
