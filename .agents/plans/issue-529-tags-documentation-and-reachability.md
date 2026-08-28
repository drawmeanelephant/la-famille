# Issue #529: tags frontmatter undocumented; /tags/ archives unreachable (WOLF-B)

## Problem

Wolf-pack review of the v0.1.0-prealpha drop (#500), WOLF-B angle:

1. **Undocumented `tags:` frontmatter key.** RELEASE-QUICKSTART.md and the
   shipped README.md never mention tags; only `content/docs/frontmatter.md`
   does, and its `tags:` bullet is misplaced mid-example. A binary-only author
   has no way to discover tags exist.
2. **Generated `/tags/` archives unreachable.** No bundled layout renders a
   page's own tags, and nothing links `/tags/` itself — sitemap.xml is the only
   way in. The `/tags/` index links each tag page, but nothing links the index.

## Already in place (prior work)

- Scaffolded `content/index.md` carries `tags: [welcome]` (commit 9f3d6e3), so
  a fresh `init` + `build` already produces `/tags/`.
- `la-famille new --tags a,b --categories x` writes both keys.

## Changes

### Code — make archives reachable

- `internal/taxonomy/taxonomy.go`:
  - `ArchiveHref(prefix, name)`: root-relative URL of an archive page, derived
    exactly the way `generateTaxonomyGroup` computes output paths.
  - `PageTagLinks(tags, pageOut, policy) []byte`: sanitized, linked tag list
    appended to an article's rendered content. Hrefs are **relative to the
    page's output directory** (same convention as taxonomy page links and
    markdown link rewriting), so they work at any depth and under subpath
    deploys without the base-path rewrite.
  - `NavLinks(links, fileMap)`: appends `Tags → /tags/` and
    `Categories → /categories/` site links when those groups produce archive
    pages, skipping any the operator already configured (label or URL match).
    Mirrors `generateTaxonomyGroup`'s render:false / blank-term filtering.
- `internal/generator/generator.go`:
  - Before rendering pages, `siteCfg.SiteLinks = taxonomy.NavLinks(...)` so
    every page (taxonomy pages included) carries the nav link.
  - Append `taxonomy.PageTagLinks(...)` to each article's sanitized content.

### Tests

- `internal/taxonomy/taxonomy_test.go`: `NavLinks` (add when archives exist,
  skip when none, no duplicates), `PageTagLinks` (relative hrefs at root and
  nested depth, dedupe, escaping, empty → nil).
- `internal/generator/generator_test.go`: end-to-end build — article HTML
  contains a `tag-link` anchor to its archive; nav (via a template that renders
  `.Site.SiteLinks`) gains Tags/Categories links only when archives exist.

### Docs

- `RELEASE-QUICKSTART.md`: new "Authoring content" section documenting
  `tags:`/`categories:` with an example and the resulting `/tags/` archive.
- `README.md`: brief mention of taxonomy frontmatter → archives.
- `content/docs/frontmatter.md`: fix misplaced `tags:` bullet, add `categories:`,
  note archive pages and nav links.

### Follow-up: tag badges in the search modal link to archives

- `search.Item` gains `TagURLs` (`gu`), parallel to `g`, because `g` mixes
  tags and categories (deduped), so a badge cannot infer its archive path.
- Generator and taxonomy search items populate `gu` via
  `PublicPathForOutput`, so URLs carry the siteurl base path like `u` does.
- `assets/js/search.js` renders badges as `<a>` elements **sibling** to the
  result link (never nested), href from `gu` with a `/tags/<name>/` fallback
  for pre-`gu` indexes.
- `assets/css/search.css`: badge anchors get `text-decoration: none`, hover
  state, and the last-result border rule now targets the `li`.
- Verified live in a preview harness: badges render as links, `#blog` →
  `/categories/blog/`, `#meta` → `/tags/meta/`, no JS errors.

## Breaking-change notes (static asset pipeline)

- Build output changes only for sites that already use `tags:`/`categories:`:
  article pages gain a linked tag footer, and the nav gains Tags/Categories
  links. Sites with neither key produce byte-identical output (no nav link,
  no footer).
- The build cache fingerprint includes the binary hash, so the change
  invalidates stale caches automatically.
