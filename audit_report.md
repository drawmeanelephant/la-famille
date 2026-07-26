# Component Mapping & Micro-Improvement Audit: La Famille

## Part 1: Component Identification
* `internal/generator`: Orchestrates the site build process, manages the output directory lifecycle (staging, replacement), and handles output path reservations to prevent collisions.
* `internal/render`: Handles HTML template parsing, caching, layout discovery, partials loading, and rendering pages to HTML files. Injects live-reload scripts during WatchMode.
* `internal/transform`: Provides URL transformation utilities, slug validation, and abstract syntax tree manipulations (e.g., link transformation, emoji kitchen).
* `internal/asset`: Manages copying static assets to the output directory and natively evaluates `.gitignore` rules to filter files without using external git subshells.
* `internal/search`: Generates clean text snippets, extracts headings from Markdown, strips HTML/Markdown noise, and writes a minified JSON search index.
* `internal/taxonomy`: Manages tags, categories, and other metadata aggregations across pages to build taxonomy listings.
* `internal/graph`: Parses backlink data, builds adjacency lists, and exports a knowledge graph JSON bundle for the explorer UI.
* `internal/ragexport`: Formats and exports site content into a clean Markdown structure suitable for Retrieval-Augmented Generation (RAG) consumption.
* `internal/config`: Defines application configuration structures (`SiteLink`, `Config`), validation logic, and default values.
* `internal/content`: Parses Markdown source files, decodes YAML frontmatter into typed structs, normalizes frontmatter keys, and gathers file metadata.
* `internal/stub`: Manages missing link placeholders by claiming output paths and generating stub pages for dangling links.
* `internal/page`: Defines the core data models and template context models used during HTML rendering.
* `internal/sitedata`: Generates aggregate site metadata outputs such as writing `meta.json`.
* `internal/markdown`: Configures and extends the Goldmark Markdown parser with custom renderers and extensions for La Famille (GFM, Typographer).
* `internal/git`: Provides native git operations and local working tree status checks.
* `internal/github`: Integrates with the GitHub API to fetch PRs, evaluate policy for automatic merging/closing (litterbox policy engine), and sync local changes.
* `internal/watcher`: Implements WatchMode using `fsnotify`, monitoring the filesystem for changes, triggering rebuilds, and serving SSE live-reload events.

## Part 2: Micro-Improvements
1. **Struct Field Alignment (`internal/search/search.go` -> `Item`)**: Reorder the fields of the `Item` struct by size (placing slices `Tags` and `Headings` before strings `Title`, `URL`, `Snippet`) to minimize padding and improve memory packing.
2. **Linear Scan to Map Lookup (`internal/github/policy.go` -> `authorAllowed`)**: `authorAllowed` and `hasRequiredLabel` perform linear slice scans and lowercasing inside loops for every PR evaluation. Converting `PolicyConfig.BotAuthors` and labels to maps (`map[string]bool`) during policy initialization would turn these `O(N)` scans into `O(1)` map lookups.
3. **Better Error Context (`internal/asset/copy.go`)**: When `os.MkdirAll` or `os.Chtimes` fails during asset directory creation or timestamp syncing, they return raw standard library errors. Wrapping them with `fmt.Errorf("failed to create asset dir %q: %w", outDirClean, err)` would provide much better context during debugging.
4. **Optimize Live-Reload Injection (`internal/render/render.go`)**: In `writeWithLiveReload`, the code writes directly to an `io.Writer` in three separate byte chunks. Utilizing `strings.LastIndex` to pinpoint `</body>` and assembling the final payload with a `strings.Builder` (pre-allocated with `sb.Grow()`) before performing a single `w.Write()` would reduce overhead and allocations.
