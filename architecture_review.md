# Component Mapping & Micro-Improvement Audit: La Famille

## Part 1: Component Identification

Here is the architectural map of the major components located in the `internal/` directory, detailing their primary responsibilities:

1. **`internal/generator`**: Orchestrates the core site build process. It manages output path claims (preventing collisions), builds sites in a temporary staging directory, and performs atomic directory swaps.
2. **`internal/render`**: Handles HTML template parsing and execution. Utilizes `sync.RWMutex` for safe concurrent caching and loading of layouts.
3. **`internal/transform`**: Walks the Markdown AST (via Goldmark) to process links, rewrite local relative `.md` paths to final URLs, and track site-wide backlinks.
4. **`internal/asset`**: Manages static asset syncing. Safely copies assets to the output directory while respecting `.gitignore` rules and preventing path traversal bounds violations.
5. **`internal/search`**: Builds minified JSON search indices by stripping Markdown/HTML noise and extracting clean text snippets and headings.
6. **`internal/taxonomy`**: Processes tags and categories from page frontmatter to generate corresponding taxonomy index pages and grouped listings.
7. **`internal/graph`**: Compiles structural site data to construct the Knowledge Graph explorer payload (JSON), mapping page relationships and backlinks.
8. **`internal/ragexport`**: Generates RAG-optimized export bundles, assembling the site's content into clean Markdown for consumption by LLMs.
9. **`internal/config`**: Responsible for configuration parsing (YAML), structural defaults, and strict validation (e.g., verifying `OutputDir` safely isolates from source inputs).
10. **`internal/content`**: Reads local files to parse YAML frontmatter and separate it from the raw Markdown body, resolving canonical key spellings and coercing data types.
11. **`internal/stub`**: Provides fallback generation mechanisms, dynamically creating stub placeholder pages for internal links targeting missing files.
12. **`internal/page`**: Defines the data models (e.g., `Page` struct) and view variables passed into HTML templates during the render phase.
13. **`internal/sitedata`**: Generates aggregate site-wide metadata artifacts (like `meta.json` or sitemaps) into the output directory.
14. **`internal/markdown`**: Configures the Goldmark Markdown engine, seamlessly registering extensions like GFM and Typographer for standardized rendering.
15. **`internal/git`**: Executes local git subprocesses to analyze branch status and check for uncommitted working tree changes.
16. **`internal/github`**: Manages GitHub API interactions, allowing the system to list open Pull Requests and execute automated sync/merge policies.
17. **`internal/watcher`**: Provides local development capabilities by binding `fsnotify` for debounced file watching and running an SSE server for live-reload updates.

## Part 2: Micro-Improvements

Based on the audit, here are 5 high-ROI localized micro-improvements to enhance memory efficiency, safety, and performance:

1. **Struct Field Alignment (`internal/config/config.go`)**:
   In the `Config` struct, boolean fields (like `WatchMode` and `CheckAssetHealth`) are interspersed with larger fields (like `Port`). This alignment creates unnecessary padding holes. By reordering the struct to group the larger fields before the 1-byte booleans, you can pack them tightly and save bytes per struct instance.
2. **Slice Pre-allocation (`internal/asset/copy.go`)**:
   In `ParseIgnoreRules`, `strings.Split(contents, "\n")` produces a statically sized slice of lines. However, the `rules` slice is dynamically appended to without pre-allocation. Initializing it with `rules := make([]IgnoreRule, 0, len(lines))` avoids unnecessary slice capacity expansions and memory copying.
3. **Enhanced Error Wrapping (`internal/asset/copy.go`)**:
   Within the asset copying routine, failing operations like `os.Chtimes` simply return `err`. Wrapping these calls with `fmt.Errorf("failed to sync asset: %w", err)` provides crucial context, ensuring logs clearly indicate exactly which file caused the sync to halt.
4. **Optimize HTML Live-reload Injection (`internal/render/render.go`)**:
   When dynamically injecting the SSE livereload script into rendered pages during WatchMode, use `strings.LastIndex` to locate the `</body>` tag. Then, append the script block utilizing a pre-allocated `strings.Builder` (with `sb.Grow()`), which prevents the heavy allocation footprint of standard `strings.Replace` or byte slice operations on large HTML documents.
5. **Slice Pre-allocation (`internal/search/search.go`)**:
   In `ExtractHeadings`, `strings.Split(string(rest), "\n")` produces a known count of lines. The `headings` slice is appended to dynamically. Since the maximum number of headings cannot exceed the number of lines, initialising it with `make([]string, 0, len(lines)/4)` or similar capacity estimation could save multiple reallocations during extraction.
