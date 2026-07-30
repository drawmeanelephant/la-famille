---
title: "La Famille Component Mapping and Micro-Improvement Audit"
date: "2024-05-24"
author: "Jules"
layout: "page"
---
# Component Mapping

- **internal/generator**: Orchestrates the static site build process, manages caching, and handles output generation and concurrency.
- **internal/render**: Responsible for HTML template rendering, executing standard templates, and ensuring correct output structures.
- **internal/transform**: Applies transformations to markdown AST, handles link rewriting, and emoji processing.
- **internal/asset**: Manages copying of static assets and respects `.gitignore` rules during asset transfer.
- **internal/search**: Defines search index data structures, builds the minified JSON index, and strips HTML markup from snippets.
- **internal/taxonomy**: Processes tags/categories and builds metadata mapping for group pages.
- **internal/graph**: Constructs the inter-document link graph, managing adjacency and backlinks.
- **internal/ragexport**: Handles the export of site content into a unified markdown format tailored for RAG pipelines.
- **internal/config**: Loads, validates, and provides default configuration settings for the site.
- **internal/content**: Parses markdown frontmatter and extracts page metadata.
- **internal/stub**: Manages placeholders for missing content or unresolved links, generating stub pages.
- **internal/page**: Defines core page template data models like `Page`.
- **internal/sitedata**: Writes global metadata and sitemaps (e.g. `meta.json`).
- **internal/markdown**: Configures and executes the Goldmark markdown parser.
- **internal/git**: Interfaces with local git status (`git status`).
- **internal/github**: Handles GitHub API interactions, such as syncing and policy checks.
- **internal/watcher**: Implements a file system watcher for live-reloading.

# Micro-Improvements
1. **Optimize Struct Memory Alignment in `BuildResult` (`internal/generator/generator.go`)**: Reorder fields to eliminate padding and reduce memory size from 136 bytes to 104 bytes.
2. **Optimize Struct Memory Alignment in `LoadResult` (`internal/retrieval/loader.go`)**: Reorder fields to eliminate padding and reduce memory size from 96 bytes to 80 bytes.
3. **Optimize Struct Memory Alignment in `buildCache` (`internal/generator/cache.go`)**: Reorder fields to eliminate padding and reduce memory size from 176 bytes to 144 bytes.
