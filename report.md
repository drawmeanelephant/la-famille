# Component Mapping & Micro-Improvement Audit: La Famille

## Part 1: Component Identification

*   `internal/generator`: Orchestrates the static site generation process (parsing, building graphs, rendering HTML) and manages the build cache.
*   `internal/render`: Handles parsing and execution of HTML templates, utilizing a double-checked locking mechanism for safe concurrent template compilation.
*   `internal/transform`: Parses and modifies Markdown ASTs (using Goldmark) to rewrite relative file links to final HTML URLs and builds the backlink/dependency graph.
*   `internal/asset`: Manages copying static assets to the output directory, securely enforcing file scope and honoring `.gitignore` rules locally.
*   `internal/search`: Generates clean text snippets, heading arrays, and minified JSON representations for client-side search indexing.
*   `internal/taxonomy`: Processes taxonomy structures (tags and categories) from frontmatter, generating localized taxonomy pages and search indices.
*   `internal/graph`: Provides a lightweight data structure to store nodes and edges, modeling the internal link network of the content.
*   `internal/ragexport`: Packages the repository markdown contents into an optimized text format suitable for Large Language Model Retrieval-Augmented Generation (RAG).
*   `internal/config`: Loads, parses, and validates the global site configuration from YAML or JSON.
*   `internal/content`: Discovers and parses Markdown files, extracting both raw frontmatter data and body content, normalizing taxonomy lists.
*   `internal/stub`: Generates placeholder markdown files for links that point to non-existent internal targets based on the backlink graph.
*   `internal/page`: Defines structural data models representing content views passed into HTML templates for rendering.
*   `internal/sitedata`: Aggregates dynamic JSON files and generates `sitemap.xml` files for web crawlers.
*   `internal/markdown`: Configures and initializes the customized Goldmark parser (with various syntax extensions) used across the project.
*   `internal/git`: Provides basic filesystem-level Git utilities (e.g. checking local branch names or finding remotes) without executing the git binary.
*   `internal/github`: Integrates with the GitHub API to list PRs, fetch commit check runs, and fetch remote repository states for sync operations.
*   `internal/watcher`: Implements live-reloading logic by monitoring the filesystem for changes and broadcasting Server-Sent Events to connected browsers.

## Part 2: Micro-Improvements

Here are 5 high-ROI micro-improvements that provide localized enhancements focused on memory efficiency, deterministic execution, and improved logging context.

1.  **Avoid Unnecessary Slice Reallocations by Pre-allocating Capacity:**
    In `internal/checker/checker.go` and `internal/generator/generator.go`, the loops that extract map keys to sort deterministically use `append` starting with a zero-length slice. Since the map size is known (`len(fileMap)`), allocating the exact capacity (`keys := make([]string, len(fileMap))`) and using index assignment avoids reallocation overhead.

2.  **Struct Field Alignment for Better Memory Packing:**
    In `internal/content/metadata.go`, the `FileMeta` struct could be optimized for 64-bit alignment by reordering fields. Grouping slices (e.g., `Tags`, `Categories`, `Warnings`), pointers (`Render`), strings, and other types sequentially can reduce memory padding and footprint, especially when caching many files.

3.  **Refactor Linear Slice Scans to Map Lookups:**
    In `internal/transform/link_transformer.go`, checking if a file is already a recorded parent for missing links involves a linear slice scan (`for _, p := range parents`). Switching the internal structure of `MissingFiles` to `map[string]map[string]struct{}` would convert this to an O(1) map lookup.

4.  **Improve Error Wrapping Context with `%w`:**
    Several error return paths (such as reading the cache in `internal/generator/cache.go`) return raw errors or strings using `fmt.Errorf` without `%w`. Wrapping the root errors with `%w` provides a better error trace for debugging build or initialization failures.

5.  **Expand `strings.Builder` Pre-allocation Strategy:**
    In `internal/search/search.go`, `ExtractSnippet` utilizes `sb.Grow(len(s))` nicely to avoid intermediate string allocations. This pattern should be extended to other string-heavy functions like XML generation in `internal/discovery/write.go` and link processing in `internal/transform/link_transformer.go`.
