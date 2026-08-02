---
title: "Component Mapping & Micro-Improvement Audit"
date: "2026-07-29"
author: "Jules"
layout: "report"
---

# Component Mapping & Micro-Improvement Audit: La Famille

## Part 1: Component Identification

*   **internal/generator**: Orchestrates the site build process, pulling in content, resolving links, caching artifacts, generating files, and swapping the staging output directory with the final one.
*   **internal/render**: Handles rendering the Markdown AST into HTML templates, managing layouts, partials, and the Go `html/template` pipeline.
*   **internal/transform**: Modifies Goldmark's Markdown AST to rewrite internal links, resolve image paths, insert backlinks, and check for slug validity.
*   **internal/asset**: Copies static files (images, CSS, JS) from the asset directory to the output directory, respecting `.gitignore` rules via native Go pattern matching.
*   **internal/search**: Generates the client-side minified JSON search index used by the frontend for site-wide search functionality.
*   **internal/taxonomy**: Processes metadata to group content by categories and tags, and handles the generation of listing pages.
*   **internal/graph**: Constructs an in-memory directed graph of internal links between markdown files for computing backlinks and the knowledge graph.
*   **internal/ragexport**: Generates XML-tagged Markdown bundles (content, config, system) that act as the corpus for Retrieval-Augmented Generation (RAG).
*   **internal/config**: Loads, parses, and validates the `config.yaml` configuration, ensuring safe local path resolution and output isolation.
*   **internal/content**: Parses Markdown files to extract and normalize YAML frontmatter and content body, ensuring strict formatting.
*   **internal/stub**: Creates placeholder "stub" pages for internal links that point to missing or non-existent files.
*   **internal/page**: Defines the standard data model (`page.Data`) exposed to HTML templates, providing context for rendering.
*   **internal/sitedata**: Generates machine-readable metadata artifacts like `sitemap.xml`, `robots.txt`, and `meta.json`.
*   **internal/markdown**: Configures and initializes the Goldmark renderer and its extensions (GFM, typographer, math, etc.).
*   **internal/git**: Provides native git subprocess execution utilities for inspecting local state (status, branch, uncommitted changes) and operations (checkout, commit, push).
*   **internal/github**: Interacts with the GitHub API for syncing PRs, evaluating merge policies, and enforcing required checks/labels.
*   **internal/watcher**: Implements WatchMode, monitoring the file system for changes and triggering live-reloading via Server-Sent Events (SSE).

## Part 2: Micro-Improvements

1.  **Memory Packing Optimization via Struct Field Alignment**:
    *   **Context**: Using `golang.org/x/tools/go/analysis/passes/fieldalignment`, we can identify several core structs with sub-optimal field ordering, resulting in wasted memory due to padding bytes.
    *   **Action**: Reorder fields in highly utilized structs such as `config.Config` (in `internal/config/config.go`), `generator.BuildResult` (in `internal/generator/generator.go`), and `retrieval.LoadResult` (in `internal/retrieval/loader.go`) from largest (slices/strings/pointers) to smallest (bools/ints). This is a localized, zero-risk change with compounding benefits for memory efficiency.
2.  **Pre-allocating Slices with Known Lengths**:
    *   **Context**: In `internal/github/github.go` (e.g. `EvaluatePR`), `hasRequiredLabel` and `authorAllowed` rely on linear scans through strings, but in functions like `internal/generator/health.go` or `internal/retrieval/loader.go`, we append to slices without pre-allocating the underlying array when the final length is deterministic or tightly bounded.
    *   **Action**: Explicitly pre-allocate slices using `make([]T, 0, len(source))` where applicable (e.g., in `enrichCorpusWithSiteMeta` or when mapping keys in `internal/generator/generator.go`).
3.  **Refactoring Linear Slice Scans to Map Lookups**:
    *   **Context**: In `internal/github/policy.go`, functions like `authorAllowed`, `hasRequiredLabel`, and `headPrefixAllowed` perform case-insensitive linear scans (`for _, a := range allowlist { ... }`) on every PR evaluation.
    *   **Action**: Refactor the policy evaluation logic to construct lowercase maps/sets for bot authors and required labels once during initialization, changing $O(N)$ lookups into $O(1)$ constant-time checks.
4.  **Improving Error Context with `%w` Wrapping**:
    *   **Context**: Throughout the codebase (e.g., in `internal/asset/copy.go` and `internal/github/github.go`), there are instances where format strings append errors loosely (`fmt.Errorf("...: %s", err)`) instead of properly wrapping them with `%w`.
    *   **Action**: Standardize error wrapping using `%w` across the `internal/` packages. This preserves the error chain for `errors.Is` and `errors.As` checks downstream and improves debugging telemetry.
