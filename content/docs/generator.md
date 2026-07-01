---
title: "How the Generator Works"
author: "Jules"
---

# How the Generator Works

La Famille is a static site generator written in Go that transforms Markdown content into fully-fledged HTML pages with associated metadata. Below is an overview of the internal pipeline and the packages that drive it.

## The Multi-Pass Pipeline

The build process is executed in several distinct passes to ensure accurate linking, rendering, and output generation:

1.  **Content Walk:** The `content/` directory is recursively scanned for all `.md` files.
2.  **Frontmatter Parse:** The frontmatter of each file is parsed to extract metadata such as title, layout, render flag, and description.
3.  **AST Transform:** The Markdown content is parsed into an Abstract Syntax Tree (AST). A custom transformer walks the AST, converting relative `.md` links into their `.html` output equivalents, and tracking missing internal links to build the site's graph.
4.  **HTML Render:** The processed Markdown is converted to safe, sanitized HTML and injected into the appropriate layout templates.
5.  **Stub Generation:** For any internal links pointing to non-existent pages, simple HTML "stubs" are automatically generated. This ensures there are no broken links on the site and provides clear entry points for future content.
6.  **Asset Copy:** Static assets like images and CSS are copied verbatim into the output directory, respecting any ignore patterns.
7.  **JSON Output:** Finally, site metadata and graph structures are exported as JSON for advanced client-side functionality.

## Key Internal Packages

The generator's functionality is cleanly separated into several internal packages:

*   **`internal/content`**: Responsible for walking the file system and parsing the YAML frontmatter from Markdown files into structured `FileMeta` objects. It also handles basic validation (e.g., date formats, tag normalization).
*   **`internal/transform`**: Houses the `LinkTransformer`, which visits AST nodes during Markdown parsing to rewrite links, enforce clean URLs (using the `url.go` utilities), and record missing files for stub generation. It also implements custom inline Goldmark parser structures, specifically the `EmojiKitchenParser` for mutating inline CDN emoji stickers.
*   **`internal/render`**: Manages the loading, parsing, and execution of HTML templates and partials. It caches templates for performance and injects the live-reload script when running in watch mode.
*   **`internal/stub`**: Handles the creation of stub pages for missing files detected during the transform phase. It lists the parent pages that linked to the missing content.
*   **`internal/asset`**: Safely copies static files from the asset directory to the output folder while skipping Go files, `testdata`, and paths matched by `.gitignore`.
*   **`internal/jsonutil`**: A simple utility package providing `WriteJSON` to easily write Go structures out as nicely indented JSON files.

## Metadata and Graphs

As part of the JSON Output step, the generator produces a few specialized files that provide deep insights into the structure of your site:

*   **`graph.json`**: Describes the entire site structure as a directed graph. It lists every page (node) and the links between them (edges), which is very useful for visualizing the structure or feeding into a knowledge graph system.
*   **`backlinks.json`**: A mapping of pages to the list of pages that link *to* them. This makes it easy to build "Mentioned In" features at the bottom of articles.
*   **`meta.json`**: Provides global site metadata mapped to page IDs, such as page titles, word counts, and tags.
*   **`search.json`**: Contains a minified array of `SearchItem` structs, providing a compressed plaintext snippet of each page's content for client-side search discovery.
