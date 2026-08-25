# La Famille binary quickstart

Verify the archive with its matching `SHA256SUMS` entry, then run the binary
without a Go toolchain or source checkout:

```bash
./la-famille --version --json
./la-famille --project-root /path/to/site init
./la-famille --project-root /path/to/site new hello --title Hello
./la-famille --project-root /path/to/site build
./la-famille --project-root /path/to/site publish-check --output public
```

`init` scaffolds a starter homepage (`content/index.md`) plus a theming demo,
so `build` produces a real site immediately.

Relative paths come from `--project-root`. The generated `public/` directory
is the complete static artifact; incremental cache state stays beside the
project root.

## Bundled themes

The binary ships with a small packet of layouts. List them any time, no
project or config required:

```bash
./la-famille themes
```

Pick one as the site default while initializing:

```bash
./la-famille --project-root /path/to/site init --theme layout-octoburger
./la-famille --project-root /path/to/site build
./la-famille --project-root /path/to/site serve
```

Switch later without a source checkout:

- **Site default:** edit the `template:` line in `config.yaml`
  (`templates/layout.html`, `templates/layout-octoburger.html`,
  `templates/layout-terminal.html`), or re-run
  `init --force --theme <name>`; the previous config is kept as
  `config.yaml.bak`.
- **Single page:** set `layout:` in that page's frontmatter — the scaffolded
  `content/theming.md` is a working example.

Every bundled layout is installed into the project's `templates/` directory by
`init`, so switching is always a one-line change.

## Release & changelog convention (decided 2026-08-20, #466)

Hybrid: **GitHub Releases are the source of truth**.

- `release.yml:165` publishes each tag via `gh release create --verify-tag --generate-notes` after proving the tag commit equals the built commit (`.github/scripts/release/tag.sh:resolve_and_checkout`). Notes are auto-generated from merged PR titles between tags.
- `content/docs/changelog.md` is a **curated snapshot** updated once per release (not per PR). Copy the highlights from the generated Release notes, de-dupe, and add human context. Keep the header's link to `https://github.com/drawmeanelephant/la-famille/releases`.
- No per-PR changelog lint gate; `jules` nightly maintenance and small fixes do not need a changelog entry. If you want a check, watch for drift between `content/docs/changelog.md` and the latest Release during release.
- This keeps `plan.md:15` and `content/meta/roadmap.md:20` honest: one convention, automated where it matters (provenance + notes generation), curated where humans add value.
