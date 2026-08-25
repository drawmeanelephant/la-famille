# Themes: Local-First Overhaul

Task ID: `themes-local-first-overhaul` (branch basis; no PR yet)

## Goal
Make the theme family mature and fully offline-capable, fix the image embed
pipeline, and add variety beyond DaisyUI (user-approved scope: full overhaul +
2 new standalone layouts).

## Work Items
1. **Image pipeline** — goldmark AST transformer promoting standalone images to
   `<figure>` with optional `<figcaption>` (from title attr), plus
   `loading="lazy"` / `decoding="async"`. Emoji-kitchen inline images and mixed
   inline content stay untouched. Extend bluemonday policy in generator.
2. **Foundations CSS** — mature shared defaults for figure/figcaption so every
   layout inherits sane image presentation.
3. **Default layout rewrite** — remove Tailwind + DaisyUI CDN tags from
   `templates/layout.html`; self-contained CSS using design tokens. Preserve
   data contract (search_modal partial, `.ComplianceModal`, landmarks, skip
   link, single h1) so contract tests keep passing.
4. **Theme tokens** — new `assets/css/theme.css` defining `[data-theme]`
   palettes for the default layout: `retro` (default), `ink`, `sepia`, `slate`,
   `moss`. Embed in binary via assets/embed.go + runtimeassets list.
5. **New layouts** — `layout-editorial.html` (serif gazette) and
   `layout-midnight.html` (restrained dark tech blog). Zero framework CSS,
   local-first, must satisfy the template contract harness.
6. **Docs/showcase** — gallery index + demo pages for the two new layouts;
   update config comment wording (`theme` no longer a DaisyUI name for the
   default layout).

## Out of Scope
- Rewriting the 19 legacy DaisyUI alternates (stay CDN-based showcase demos).
- Copying content-local images into output (separate feature).

## Breaking-Change Risk Assessment (asset generation pipeline)
- Default layout no longer references CDNs → generated pages work offline;
  sites overriding layout.html keep their old bytes (override precedence
  unchanged).
- New required asset `css/theme.css`: installed by InstallMissing on existing
  projects (file absent → written). Release binaries embed it.
- Markdown images now emit `<figure>` markup: any site CSS targeting bare
  `p > img` may need updating (foundations provide neutral defaults).
- bluemonday policy grows figure/figcaption/loading/decoding allowances.

## Verification
- `go test ./... && go vet ./...`
- New unit tests: transformer behavior (figure promotion, caption, lazy attrs,
  emoji-kitchen untouched), sanitizer survival of figure markup.
- Build site locally and smoke-check rendered HTML for default + new layouts.
