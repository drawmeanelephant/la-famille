---
date: "2026-07-23"
title: "Publishing Output Contract"
author: "Antigravity"
---

# Publishing Output Contract

This document specifies the contract, generation conditions, URL structures, and exclusion rules for all publishing artifacts produced by La Famille during static site generation.

---

## Overview of Generated Artifacts

When `la-famille build` executes, it processes Markdown source files in `contentDir` and writes output files to `outputDir`. The generator produces rendered HTML pages, search and discovery files, taxonomy listings, link graphs, and structural metadata.

| Artifact | Output Path | Generation Condition | Primary Purpose |
| :--- | :--- | :--- | :--- |
| **HTML Pages** | `<slug>/index.html` or `<path>/index.html` | Rendered for every `.md` source file with `render != false` | Public site web pages |
| **RSS Feed** | `feed.xml` | Generated when at least one rendered page has a `date` field | Syndication feed for web readers |
| **Sitemap** | `sitemap.xml` | Generated on every build | Search engine discovery |
| **Robots Rules** | `robots.txt` | Generated on every build | Search engine crawler rules & sitemap location |
| **Search Index** | `search.json` | Generated on every build | Client-side search index |
| **Taxonomies** | `tags/`, `categories/` | Generated when pages specify `tags` or `categories` frontmatter | Rendered tag and category index & detail pages |
| **Link Graph** | `graph.json` | Generated on every build | Page node and wikilink edge dataset |
| **Backlinks** | `backlinks.json` | Generated on every build | Map of target page IDs to referencing parent page IDs |
| **Site Metadata** | `meta.json` | Generated on every build | Page metadata dictionary |
| **Knowledge Graph Explorer** | `graph/index.html` | Generated on every build when `graph_explorer: true` (default) | Self-contained interactive visualization of the site's wikilink relationships |
| **Explorer Payload** | `graph/data.json` | Generated alongside `graph/index.html` | Resolved node list the explorer page renders: link direction, classification, titles, and public URLs |

---

## Detailed Artifact Contracts

### 1. HTML Pages & Metadata Tags

- **Generation:** Each `.md` file where `render` is `true` (default) is compiled through the template layout.
- **Canonical & OpenGraph Metadata:** Rendered inside `<head>`:
  - `<link rel="canonical" href="...">`: Contains the absolute URL if `siteurl` is configured (e.g. `https://example.com/about/`), or root-relative URL (e.g. `/about/`) if `siteurl` is unconfigured.
  - `<meta property="og:url" content="...">`: Set to the canonical URL.
  - `<meta property="og:title" content="...">`: Page title (or filename fallback).
  - `<meta property="og:description" content="...">`: Frontmatter `description`, or `default_description` from `config.yaml`.
  - `<meta property="og:image" content="...">`: Frontmatter `image`, or `default_og_image` from `config.yaml`.

### 2. RSS Feed (`feed.xml`)

- **Generation:** Written to `outputDir/feed.xml` whenever there is at least one rendered page containing a non-empty `date` frontmatter field (formatted as `YYYY-MM-DD`).
- **Clean-up:** If a build contains zero dated rendered pages, any existing `feed.xml` in `outputDir` is deleted to prevent serving stale feeds.
- **Content:** Contains RSS 2.0 `<item>` elements sorted by date (newest first). Each item includes `<title>`, `<link>`, `<guid>`, `<pubDate>` (RFC1123Z format), and `<description>` (extracted text snippet).

### 3. Sitemap (`sitemap.xml`)

- **Generation:** Written on every build using standard XML sitemap format (`http://www.sitemaps.org/schemas/sitemap/0.9`).
- **Included URLs:** Unique output locations for all rendered Markdown pages and generated taxonomy pages (`tags/index.html`, `tags/<tag>/index.html`, `categories/index.html`, `categories/<cat>/index.html`).
- **Exclusions:** Pages with `render: false` and unrendered raw files are excluded.

### 4. Robots Rules (`robots.txt`)

- **Generation:** Written on every build containing:
  ```txt
  User-agent: *
  Allow: /
  ```
- **Sitemap Directive:** If `siteurl` is configured, appends `Sitemap: <siteurl>/sitemap.xml`. If `siteurl` is omitted/empty, the `Sitemap:` line is omitted.

### 5. Search Index (`search.json`)

- **Generation:** Minified JSON array written to `outputDir/search.json`.
- **Item Fields:**
  - `t` (Title): Page title or taxonomy heading.
  - `u` (URL): Root-relative URL path (e.g., `/posts/first-post/`).
  - `g` (Tags/Categories): Combined slice of tags and categories.
  - `s` (Snippet): Up to 160 characters of clean text extracted from page body (stripping Markdown codeblocks, HTML tags, and formatting).
  - `h` (Headings): Extracted ATX heading titles (`#` through `######`).
- **Included Entries:** Includes all rendered content pages and all generated taxonomy pages.

### 6. Taxonomy Pages (`tags/` and `categories/`)

- **Generation:** When content files declare `tags` or `categories` arrays:
  - Main index pages are generated at `tags/index.html` and `categories/index.html`.
  - Term detail pages are generated at `tags/<tag-name>/index.html` and `categories/<category-name>/index.html`.
- **Content:** Lists titles and relative links of associated rendered pages.
- **Exclusions:** Pages with `render: false` are excluded from tag/category aggregation.

### 7. Graph & Metadata Files (`graph.json`, `backlinks.json`, `meta.json`)

- **`graph.json`**: Contains `nodes` map (keyed by page ID) and `edges` list of directed wikilink pairs `[source, target]`. Nodes include `"type": "page"` and `"render": true|false`.
- **`backlinks.json`**: Map of target page ID -> sorted array of referencing parent page IDs.
- **`meta.json`**: Map of page ID -> metadata object (`title`, `author`, `date`, `tags`, `word_count`, `render`, `categories`). Unknown fields are forward-compatible additions.

### 8. Knowledge Graph Explorer (`graph/index.html`)

- **Generation:** Written to `<output_dir>/graph/index.html` when the `graph_explorer` config option is true (default). When disabled, neither the page nor its companion assets directory are emitted and no nav link is injected into layouts.
- **Companion assets:** `assets/graph/explorer.js` and `assets/graph/explorer.css` are copied into `<output_dir>/assets/graph/` by the established asset pipeline. Both files ship under the user-owned `assets/` directory and reach the explorer page via root-relative URLs.
- **Runtime semantics:** The page is fully static — it loads `../graph.json`, `../meta.json`, and `../backlinks.json` via relative fetches and never contacts a remote host. A `<link rel="canonical">` is emitted only when `siteurl` is configured; without it the page works the same when opened directly via `file://`.
- **No change to existing contracts:** The explorer's writer does NOT extend `graph.json`, `meta.json`, or `backlinks.json`. It emits its own `graph/data.json` payload, assembled in Go from the same graph the build already computed. Public URLs there follow the path each page was actually written to, so front-matter `slug:` overrides and sub-path deployments (for example a GitHub Pages project site) both produce correct links.

### Manual Navigation Snippet

Adding a global nav link to every bundled template would break custom-template compatibility, so the explorer does not auto-inject one. Drop this anywhere inside a layout that you control:

```html
<a href="/graph/" rel="nofollow">Knowledge Graph</a>
```

The link is root-relative so it works with or without `siteurl`, mirroring the explorer's own URL construction.

### Explorer Orphan Rule

The explorer flags a page as **orphan** when it has zero inbound links unless the page id is `index` (the rendered homepage). Rendered pages named `about`, `posts/2026/welcome`, etc. follow the standard zero-inbound rule. Raw `render: false` pages never appear as orphan candidates because their IDs carry the `.md` suffix and the rule is not applied to them. Sites that use `render: false` for their homepage must provide inbound links from somewhere else.

### Large-Site Threshold

Sites with **≥ 500 nodes** default to a search-first view; the graph visualization is suppressed until the user clicks a search suggestion. The threshold is exposed in the page footer so future tuning can be done without code changes.

---

## Configuration Impact: `siteurl` / `SiteURL`

The `siteurl` configuration option defines the public canonical base URL (e.g. `https://example.com`).

| Context | When `siteurl` is Configured | When `siteurl` is Empty / Unset |
| :--- | :--- | :--- |
| **Canonical Link Tag** | `<link rel="canonical" href="https://example.com/about/">` | `<link rel="canonical" href="/about/">` |
| **OpenGraph URL** | `<meta property="og:url" content="https://example.com/about/">` | `<meta property="og:url" content="/about/">` |
| **RSS Feed Item Link** | `<link>https://example.com/posts/p1/</link>` | `<link>/posts/p1/</link>` |
| **Sitemap Location** | `<loc>https://example.com/about/</loc>` | `<loc>/about/</loc>` |
| **Robots Sitemap** | Includes `Sitemap: https://example.com/sitemap.xml` | Omitted from `robots.txt` |

---

## Unrendered Pages (`render: false`)

Frontmatter allows specifying `render: false` to copy raw file contents directly to output without template layout rendering or sanitization.

### Exclusion vs Inclusion Rules for `render: false`

- **EXCLUDED from:**
  - HTML template layouts (no `<head>`, canonical tags, or HTML wrappers)
  - `feed.xml` (RSS feed items)
  - `sitemap.xml`
  - `search.json` (search index entries)
  - Taxonomy index and term detail pages (`tags/`, `categories/`)

- **INCLUDED in:**
  - `graph.json` (as a node with `"render": false`)
  - `meta.json` (with `"render": false`)
  - `backlinks.json` (tracked if referenced by or referencing other pages via wikilinks)

---

## Safe to Publish vs Internal Cache Files

When deploying the generated static site to production or static web hosts:

### Safe to Publish (Web Root)
All files and subdirectories created in `outputDir` are intended for public web serving, including:
- HTML pages and assets
- `feed.xml`, `sitemap.xml`, `robots.txt`
- `search.json`
- `graph.json`, `backlinks.json`, `meta.json`
- `graph/index.html` (Knowledge Graph Explorer) and `assets/graph/explorer.{js,css}`

### Do Not Publish / Internal Only
- `.buildcache.json`: Stored inside `outputDir` for incremental build change detection. Safe to host if published, but intended strictly as generator build cache state.
- `.staging-*` directories: Temporary build staging folders created during atomic build execution. Cleaned up automatically upon build completion.

---

## Knowledge Graph Explorer

| Aspect | Detail |
| :--- | :--- |
| **Page path** | `<output_dir>/graph/index.html` (controlled by `graph_explorer: true` config, **default: true**) |
| **Companion bundle** | `<output_dir>/assets/graph/explorer.{js,css}` copied from user-owned `assets/graph/` by the asset pipeline |
| **Data sources** | `../graph.json`, `../meta.json`, `../backlinks.json` (loaded at runtime via relative fetches) |
| **Runtime network calls** | None. The page is fully static; no remote scripts, fonts, or APIs are loaded. |
| **Canonical URL** | Emitted only when `siteurl` is configured. Otherwise the page works the same when opened directly via `file://`. |
| **Slug handling** | Public URLs are resolved by the generator from the output path each page was written to, so frontmatter `slug:` overrides link correctly. |
| **Sub-path deploys** | URLs include the base path from `siteurl`, so a project site such as `https://user.github.io/project` links to `/project/...` rather than `/...`. |
| **Link direction** | Inbound and outbound neighbours are computed once in `internal/graph.Adjacency` from the `[source, target]` edge list, deduplicated and sorted. The client renders them; it does not re-derive them. |
| **Disabled behavior** | With `graph_explorer: false`, neither `/graph/` nor `/assets/graph/*` are emitted and no nav link is auto-injected. |

### Threshold for large sites

Sites with **≥ 500 nodes** open in search-first mode; the graph visualization stays suppressed until the user clicks a search suggestion. The threshold is exposed in the page footer so future tuning can be done without code changes.

### Manual Navigation Anchor

```html
<a href="/graph/" rel="nofollow">Knowledge Graph</a>
```

Root-relative, so it works with or without `siteurl` — mirroring the explorer's own URL construction.

### Orphan Rule (Explorer)

A page is treated as **orphan** when its inbound list is empty, with one carve-out: the rendered homepage (page id `index`) is exempt so a freshly-seeded site that links out from `index.md` doesn't flag `index` as orphan. Rendered sub-pages, raw `render: false` pages (whose IDs carry the `.md` suffix), and sites that use `render: false` for the homepage follow the standard zero-inbound rule.
