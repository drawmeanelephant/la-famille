---
title: "Routine Report: Sitemap Generation and Docs Sync"
date: "2026-07-01"
---

# Routine Report: Sitemap Generation and Docs Sync

**Date:** 2026-07-01
**Routine Name:** Sitemap Generation and Docs Sync
**Success Status:** Success

## Details

*   Updated `internal/sitedata/write.go` to generate `sitemap.xml` properly using `urlset`, `url`, and `loc` tags.
*   Added safeguard logic against traversal bugs when determining the sitemap output path.
*   Sorted the metadata keys alphabetically before writing to ensure deterministic `sitemap.xml` output.
*   Updated the tests in `internal/sitedata/write_test.go` to verify standard XML structure and `<loc>` links.
*   Updated `content/docs/generator.md` to document the `EmojiKitchenParser` custom inline parser for Goldmark.
*   Created `content/docs/emoji.md` documenting the exact shorthand syntax (`!ek[emoji+emoji]`) for rendering mutant CDN emoji kitchen stickers.

## Suggestions

*   Currently the sitemap only contains the page URL (`loc`). Future iterations could parse the frontmatter to include standard sitemap fields like `<lastmod>`, `<changefreq>`, or `<priority>` to provide better crawl hints to search engines.
