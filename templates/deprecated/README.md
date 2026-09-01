# Deprecated themes

These layouts depend on CDN Tailwind/DaisyUI (`cdn.tailwindcss.com`,
`cdn.jsdelivr.net`) and some reference hardcoded assets that no longer exist
(`/assets/vid/...`, `mascot-electric-blue.jpeg`). That violates La Famille's
local-first guarantee: pages must render offline from `file://` with no
network requests.

They were moved out of `templates/` on 2026-08-31 so they no longer ship in
the release theme packet (`templates/embed.go` only embeds the curated five).
The curated, framework-free themes are:

- `layout.html` — default, token-driven (`assets/css/theme.css`)
- `layout-octoburger.html` — flagship, fully self-contained
- `layout-terminal.html` — synthwave console
- `layout-editorial.html` — serif gazette
- `layout-midnight.html` — restrained dark

If you want to revive one of these, port it to local CSS (system font stacks,
no CDN) and move it back next to the curated layouts.
