# La Famille binary quickstart

Verify the archive with its matching `SHA256SUMS` entry, then run the binary
without a Go toolchain or source checkout:

```bash
# macOS / Linux (sha256sum on most Linux distros, shasum on macOS)
shasum -a 256 --check SHA256SUMS

# No shasum on PATH? Python 3 works everywhere macOS ships:
python3 -c "import hashlib,sys; print(hashlib.sha256(open(sys.argv[1],'rb').read()).hexdigest())" \
  la-famille_0.1.0-prealpha_darwin_arm64.tar.gz
```

Compare the printed digest against the matching line in `SHA256SUMS`. Compare the
*digest*, not the filename: the archive you downloaded may be named
`la-famille.tar.gz` (a renamed drop) while the checksum line names the versioned
asset `la-famille_<version>_<os>_<arch>.tar.gz`. What must match is the
hex digest, on your platform's line.

Then:

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
is the complete static artifact. Incremental cache state lives inside the
project root as `.la-famille-cache.json`, and a failed build swap can briefly
leave a transient `.public.previous-<random>` copy of the previous site beside
`public/`; delete it manually if it is ever stranded (a build that cannot
remove it reports it as a warning).

## Deploying `public/` to GitHub Pages (binary-only install)

`public/` is the only directory you upload. Two settings make or break a
project pages site (`<user>.github.io/<repo>`):

1. **Set `siteurl` in `config.yaml` to the full site URL including the repo
   path**, e.g. `https://<user>.github.io/<repo>`. Without the `/repo` suffix,
   asset and link paths resolve at the domain root and every subresource 404s.
2. Upload the *contents* of `public/`, not the directory itself, so
   `index.html` sits at the site root.

Leave `siteurl` unset only for local-only builds. Without it the generated
`sitemap.xml` has root-relative `<loc>` entries, which the sitemaps.org
protocol requires to be absolute — so a built-and-uploaded site ships an
invalid sitemap on day one (run `la-famille check` and set `siteurl` before
you deploy anywhere public).

Minimal Actions snippet for a binary-only operator:

```yaml
- name: Deploy public/ to GitHub Pages
  uses: actions/upload-pages-artifact@v3
  with:
    path: public
# then a deploy step (actions/deploy-pages@v4) — or upload public/ manually
# under Settings → Pages when you have no CI at all.
```

Run `publish-check --output public --strict` before uploading; it fails on
broken internal links and generated "Missing Page" stubs so a typo never ships.

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

`serve` starts on port 8080 by default. If that port is busy (common on macOS
with Docker or OrbStack running), recover with `serve -p <port>` or by
setting `port:` in `config.yaml` — the bind error now says exactly that.

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

## More commands

The quickstart above walks the core workflow. The binary ships everything
below — `--help` lists them all, and none needs a source checkout:

- **`check`** — validate frontmatter, dates, tags, slugs, internal links, and
  (with `--asset-health`) asset references before you publish. `new` and
  `serve --watch` point at it; run it after writing content.
- **`rag`** — export the project into RAG-friendly markdown bundles
  (`rag-archive/` by default).
- **`ask`** — local citation-grounded Q&A over the RAG archive.
- **`tui`** — a semi-graphical full-screen interface.
- **`pr`** — manage GitHub PRs (`pr sync`).
- **`completion`** — shell autocompletion.

### Tags

Add a `tags:` list in a page's frontmatter and `build` renders the tag archive
under `/tags/` (plus a per-tag listing and a `/tags/` index):

```yaml
---
title: "A post"
date: "2026-08-27"
tags:
  - writing
  - meta
---
```

Tag names are lowercased and reduced to `[a-z0-9-]` for the URL; a value that
cannot survive that (`café ☕` → `caf`, a purely-non-ASCII tag) is reported as
a warning, and one that normalizes to nothing is dropped with a warning. The
bundled themes link a page's own articles but do not yet add `/tags/` to the
nav, so visitors reach archives through the pages that tag them or a custom
nav link.

## Release & changelog convention (decided 2026-08-20, #466)

Hybrid: **GitHub Releases are the source of truth**.

- `release.yml:165` publishes each tag via `gh release create --verify-tag --generate-notes` after proving the tag commit equals the built commit (`.github/scripts/release/tag.sh:resolve_and_checkout`). Notes are auto-generated from merged PR titles between tags.
- `content/docs/changelog.md` is a **curated snapshot** updated once per release (not per PR). Copy the highlights from the generated Release notes, de-dupe, and add human context. Keep the header's link to `https://github.com/drawmeanelephant/la-famille/releases`.
- No per-PR changelog lint gate; `jules` nightly maintenance and small fixes do not need a changelog entry. If you want a check, watch for drift between `content/docs/changelog.md` and the latest Release during release.
- This keeps `plan.md:15` and `content/meta/roadmap.md:20` honest: one convention, automated where it matters (provenance + notes generation), curated where humans add value.
