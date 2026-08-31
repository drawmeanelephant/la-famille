# Bundled theme quality pass

Task ID: `theme-quality-pass`
Branch: master

## Decision (asked via structured questions)

Single-shot selection:
- **Scope:** the 4 remaining embedded themes (`layout`, `layout-terminal`,
  `layout-editorial`, `layout-midnight`) — octoburger already polished as the
  flagship/global default.
- **Dependency policy:** local-first and framework-free — no CDN, no
  Tailwind/DaisyUI-from-network.

## Changes

- **`layout-terminal` rewritten local-first:**
  - New `assets/css/layout-terminal.css` replacing the CDN Tailwind/DaisyUI
    approach; synthwave console identity kept (mono stack, neon pink/cyan,
    terminal-window article, traffic-light titlebar, blinking caret disabled
    under `prefers-reduced-motion`).
  - Clean, dynamic copy: brand/site-name from config, `~/<SiteName>` path,
    footer "Built with La Famille — static sites in pure Go" (removed the
    "…and DaisyUI by Google" text and hardcoded chrome).
  - Added skip link, `main#main-content`, one `<title>`/`<h1>`, viewport,
    canon contact/og:url conditionals, nav landmark, on-site search, and a
    compliance `<dialog>` for parity with the other bundled layouts.
- **`layout-editorial` / `layout-midnight` polish:**
  - Added the on-site search partial (`search.css` + `search.js`), a
    site-name-aware footer, and a compliance `<dialog>`; editorial search sits
    right of the centered nav (collapsing to full-width on narrow screens),
    midnight search joins its flex nav.
  - `layout-midnight.css` / `layout-editorial.css`: new search sizing and
    dialog/button styles.
- **Default `layout.html`:** already local-first and consistent (tokens, search,
  compliance, object-fit avatar); no changes needed.
- **Embedding/wiring:**
  - `assets/embed.go` `//go:embed` now includes `css/layout-terminal.css`.
  - `internal/runtimeassets/assets.go` `DefaultAssetFiles` includes it.
  - `assets/testdata/sites/release-smoke/manifest.txt` adds the new asset.
  - Docs: `content/docs/templates.md` now lists all 5 embedded themes as
    local-first/offline and no longer lumps terminal into the CDN gallery.

## Verification

- `go test ./...` — all packages pass (incl. the layout contract harness,
  which auto-discovers and validates every bundled layout, and the release
  smoke manifest check).
- `go vet ./...` clean.
- Rebuilt site: terminal page has zero CDN references; search present on all
  three themes; new CSS serves 200 and resolves under a static server.