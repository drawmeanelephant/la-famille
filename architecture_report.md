# Component Mapping & Micro-Improvement Audit: La Famille

## Part 1: Component Identification

Here is the mapping of all major components currently residing in the `internal/` directories and a brief summary of their responsibilities:

- **internal/generator**: Handles build orchestration, managing the multi-threaded rendering process and build passes.
- **internal/render**: Manages HTML template parsing, caching, and execution using Go's `html/template`.
- **internal/transform**: Performs Markdown AST manipulation, utilizing Goldmark to rewrite local Markdown links into web URLs.
- **internal/asset**: Manages copying static assets to the output directory while natively evaluating and respecting `.gitignore` rules in Go.
- **internal/search**: Generates a minified JSON search index by stripping Markdown and HTML tags from content to produce clean text snippets.
- **internal/taxonomy**: Groups pages by tags and categories, and handles the rendering of taxonomy index pages.
- **internal/graph**: Manages the graph data structure (nodes and edges) to keep track of backlinks and missing files.
- **internal/ragexport**: Exports site content into a consolidated, specially formatted Markdown file optimized for Retrieval-Augmented Generation (RAG) ingestion.
- **internal/config**: Defines the configuration structures, validates user configurations, and sets default values.
- **internal/content**: Traverses the content directory, parsing YAML frontmatter and the remaining Markdown content for each file.
- **internal/stub**: Automatically generates placeholder stub pages for internal links that point to missing or uncreated files.
- **internal/page**: Defines the data model structures (`Page`) that are passed to HTML templates during rendering.
- **internal/sitedata**: Generates site-wide metadata files, such as sitemaps.
- **internal/markdown**: Configures and provides the Goldmark parser along with its required extensions.
- **internal/git**: Provides wrappers around local `git` shell commands to check repository status and retrieve remote URLs.
- **internal/github**: Interacts with the GitHub API to synchronize uncommitted content by creating and managing Pull Requests.
- **internal/watcher**: Provides file system watching (via `fsnotify`) and Server-Sent Events (SSE) functionality to enable live-reloading during local development.

## Part 2: Micro-Improvements

Here are 4 high-ROI micro-improvements focused on localized enhancements:

1. **Struct Field Alignment for Memory Packing**: Optimize several structs identified by the `fieldalignment` tool to reduce pointer bytes and overall size. For example, reordering fields in `internal/page.Page`, `internal/content.FileMeta`, and `internal/render.Renderer` can significantly improve memory packing and cache locality.
2. **Pre-allocating Slices**: In several slice append operations across the codebase, the final capacity is known or can be estimated. Pre-allocating slices using `make([]Type, 0, capacity)` in functions within `internal/taxonomy` and `internal/checker` will minimize slice reallocation overhead during generation.
3. **Error Wrapping and Logging Context**: Enhance error wrapping in `internal/watcher` and `internal/github`. By attaching more specific contextual metadata (e.g., exact file paths in the watcher, or API response bodies/codes in the GitHub client), diagnosis of live-reload edge cases and PR sync failures will be much faster.
4. **String Builder Pre-allocation for HTML Injection**: When modifying HTML output for WatchMode live-reload injection in `internal/render`, utilize a `strings.Builder` pre-allocated with `sb.Grow()` (based on the original HTML length plus the script block) to append the script block with minimal allocations, rather than performing heavy `strings.Replace` operations.
