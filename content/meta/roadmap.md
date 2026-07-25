---
title: "La Famille SSG: Roadmap"
author: "Jules"
date: "2026-07-23"
---

# Roadmap

This is the short, current list of work worth doing next. Completed projects are recorded below so the backlog stays honest.

## Current priority

### 1. Release readiness

Make the project easier to consume and safer to ship:

- Add a clear version/build-info command and expose the version in generated diagnostics.
- Add a small end-to-end smoke test that builds a representative site and checks the key generated artifacts.
- Document the supported build, serve, watch, check, new, and RAG workflows in one concise getting-started path.
- Decide on a release/changelog convention and keep it automated where practical.

### 2. TUI workflow polish

Improve the useful parts of the TUI without expanding its visual theme system:

- Make build, serve, watch, and failure states easier to understand at a glance.
- Improve keyboard help and command discoverability.
- Make diagnostics and content-health findings link clearly to the next CLI action.
- Keep the current mascot and visual language stable.

### 3. Build correctness and performance

Turn the existing transactional build and cache work into a dependable production path:

- Add regression coverage for cache invalidation when content, templates, assets, configuration, or deleted files change.
- Measure cold versus incremental builds on representative sites.
- Audit generated-output cleanup, concurrent builds, and watcher-triggered rebuilds.

### 4. Content quality and publishing

Use the existing checks and generated metadata to help authors before deployment:

- Add a concise validation summary for broken links, missing metadata, asset health, and orphaned pages.
- Verify RSS, sitemap, robots, canonical URLs, search data, and taxonomy pages together in an integration fixture.
- Document the output contract for these generated publishing artifacts.

## Explicitly deferred

- Jules mascot themes, alternate animations, and broader visual customization. The current TUI is good enough; revisit this only after the workflow and release work above is complete.
- Large template/theme redesigns. Prefer targeted accessibility, layout, and interaction fixes.

## Shipped

- Generator, graph, and stub logic extracted into tested `internal/` packages.
- Static asset copying, template partials, multiple layouts, and shared style foundations.
- Local `serve` and watch workflows with lifecycle and debounce coverage.
- GitHub Actions build, test, generate, and deploy workflow.
- Client-side search, taxonomy/tag/category generation, RSS, sitemap, robots, and canonical publishing metadata.
- Transactional output and incremental build cache with observable TUI status.
- `la-famille new` content scaffolding and `la-famille check` asset-health diagnostics.
- Template contract/accessibility regression coverage and TUI content-health metrics.
