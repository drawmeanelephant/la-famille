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
- **`meta.json`**: Map of page ID -> metadata object (`title`, `author`, `date`, `tags`, `word_count`, `render`).

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

### Do Not Publish / Internal Only
- `.buildcache.json`: Stored inside `outputDir` for incremental build change detection. Safe to host if published, but intended strictly as generator build cache state.
- `.staging-*` directories: Temporary build staging folders created during atomic build execution. Cleaned up automatically upon build completion.
