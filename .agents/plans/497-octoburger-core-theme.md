# 497: Ship a curated theme set inside the release binary

Task ID: `497-octoburger-core-theme`
Issue: drawmeanelephant/la-famille#497 (milestone v0.1.0-prealpha)

## Scope

Make a curated set of themes usable from a binary-only install, with the new
octoburger theme at the core of the packet.

Operator direction (2026-08-25): the octoburger style — the 🍔 OCTOBURGER MENU
identity from the TUI (`cmd/la-famille/tui.go:801`), Raoul(s) the octopus
mascot, pink-205/212 highlights, yellow-228 headers, blue-39 accents — becomes
the soul of the release. It needs a site layout of its own.

## Design decisions

- **New flagship layout `templates/layout-octoburger.html`.** Translates the
  TUI palette to CSS (pink #f472b6 selection/focus, bun-yellow #ffe08a,
  Raoul-blue accents, charcoal panels), burger-stack card styling, octopus-wave
  flourishes, pure-CSS checkbox drawer menu (label-based toggle, no JS, no
  unresolvable anchors). Must satisfy every rule in
  `internal/render/template_contract_harness_test.go`, which auto-discovers new
  layouts.
- **Curated packet = 3 themes:** `layout` (default), `layout-octoburger`
  (flagship), `layout-terminal` (existing, self-contained, zero design work).
- **Selection mechanism (minimal path):** everything stays file-based.
  - `templates/embed.go` embeds the curated layouts beside `layout.html`.
  - `runtimeassets.CuratedLayouts()` exposes them to the runtime.
  - `init` installs ALL bundled layouts into the project `templates/` dir
    (missing-only, site files stay authoritative), so frontmatter
    `layout:` switching works across the packet immediately.
  - New `init --theme <name>` validates against the bundled set and writes
    that layout as the configured default (`config.WriteDefault` gains a
    template-path parameter).
- No render/generator changes; no breaking change to existing projects.

## Static asset generation impact

- Fresh `init` output tree now includes the bundled layouts beyond
  `layout.html` (+ unchanged search partial). No effect on generated `public/`
  manifests; release-smoke fixture builds bypass init.
- New showcase page `content/showcase/layout-octoburger.md` renders on the
  deployed site.

## Tests

- Contract harness covers `layout-octoburger` automatically once the file
  exists; must pass all ten rule groups.
- `internal/runtimeassets`: curated set contents, byte-level sanity.
- `cmd/la-famille`: `init --theme` writes theme file + config default;
  unknown theme errors naming valid choices; plain `init` unchanged behavior;
  existing files never overwritten.
- Local validation: `go test ./... && go vet ./...`.

## Non-goals

- More themes beyond the three (follow-ups under #497 comments later).
- Docs/quickstart rewrite (issue #498), changelog/tag (#499).
