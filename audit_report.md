# Component Mapping & Micro-Improvement Audit: La Famille

## Part 1: Component Identification
The `internal/` directory contains the core business logic of La Famille, divided into focused components:

*   **`internal/generator`**: Orchestrates the overall site build process. It manages concurrency, dependencies, error collection, and build caching to output the final static site.
*   **`internal/render`**: Handles the rendering of HTML templates.
*   **`internal/transform`**: Manages transformations, such as adjusting markdown AST links and applying Emoji Kitchen features.
*   **`internal/asset`**: Copies static assets from the source to the output directory, respecting `.gitignore` patterns without invoking external `git` subprocesses.
*   **`internal/search`**: Responsible for generating a minified JSON search index from the content.
*   **`internal/taxonomy`**: Aggregates and renders taxonomy terms (like tags and categories) across the site.
*   **`internal/graph`**: Manages the adjacency list of backlinks and relationships between content nodes.
*   **`internal/ragexport`**: Exports the corpus into markdown bundles formatted for Retrieval-Augmented Generation (RAG) consumption.
*   **`internal/config`**: Provides configuration validation, defaults, and struct definitions for the site and engine.
*   **`internal/content`**: Extracts and structures metadata and YAML frontmatter from content files.
*   **`internal/stub`**: Provides logic for resolving missing or stubbed placeholders within the content.
*   **`internal/page`**: Defines the data models used when rendering templates.
*   **`internal/sitedata`**: Handles the writing of structural site metadata (like `meta.json` and sitemaps).
*   **`internal/markdown`**: Configures and instantiates the Goldmark markdown rendering engine with custom extensions.
*   **`internal/git`**: Provides utilities for querying local git repository status.
*   **`internal/github`**: Handles external GitHub API interactions, such as checking Pull Request policies and syncing data.
*   **`internal/watcher`**: Implements file system watching, debouncing, and SSE-based live-reloading for the development server.
*   **`internal/ask`**: HTTP orchestrator tying the retrieval corpus and LLM provider together for answering queries.
*   **`internal/retrieval`**: Handles corpus and artifact loading for RAG (Retrieval-Augmented Generation).

## Part 2: Micro-Improvements
Based on a focused architectural review, here are 4 high-ROI micro-improvements focusing on localized technical debt, memory optimization, and performance:

1.  **Struct Field Alignment Optimization (`internal/ask/server.go`, `internal/generator/cache.go`, etc.):**
    *   Several core structures are sub-optimally aligned in memory. For example, in `internal/generator/cache.go`, reordering `buildCache` to group pointers and word-sized fields together saves 32 bytes per struct (reducing from 176 to 144 bytes).
    *   Similar improvements can be made in `internal/ask/server.go` (`AnswerResponse` from 168 to 152 bytes) and `internal/generator/generator.go` (`BuildResult` from 136 to 104 bytes).

2.  **Pre-allocate Slices in Hot Loops (`internal/taxonomy/taxonomy.go`, `internal/generator/generator.go`):**
    *   In `internal/taxonomy/taxonomy.go` (line 163), when gathering deduplicated pages, the `pages` slice is instantiated dynamically. Pre-allocating it with `make([]string, 0, len(rawPages))` will avoid intermediate reallocations.
    *   In `internal/generator/generator.go` (line 513), `joinErrs` is appended to without a known length, but its length will always exactly match `len(errs)`. We can pre-allocate it with `make([]error, 0, len(errs))`.

3.  **Optimize Map-based Slice Deduplication (`internal/generator/generator.go`):**
    *   In `internal/generator/generator.go` (line 342), a map `taxonomySeen` and slice `taxonomyTerms` are created per-file to deduplicate tags. Since we know the maximum capacity is `len(meta.Tags)`, pre-allocating `taxonomyTerms` and potentially reusing a map across the loop could save allocations in the hot path.

4.  **Error Wrapping Enhancements (`internal/generator/generator.go`):**
    *   In `internal/generator/generator.go` (line 517), `errors.Join(joinErrs...)` is used nicely to aggregate errors. However, there are places where native errors are returned without contextual wrapping. Ensuring all errors propagated up from components like `internal/sitedata` are wrapped with `fmt.Errorf("...: %w", err)` improves observability when debugging build failures.
