---
title: "La Famille SSG: Roadmap & Backlog"
author: "Jules"
date: "2026-06-19"
render: true
---

# Roadmap & Backlog

This document outlines the active roadmap, development milestones, and ready-to-hand tasks for La Famille.

## Milestones

- **Milestone 1: Foundation cleanup** — docs reconciliation, package refactor, test hardening, and output contracts for `graph.json`, `backlinks.json`, and `meta.json`.
- **Milestone 2: Creator workflow** — asset pipeline, multi-template architecture, serve/watch flow, and better stub-page UX.
- **Milestone 3: Publish and discover** — CI/CD deploy, search UI powered by existing metadata, and taxonomy/tag generation from frontmatter.

## Backlog

- Extract generator, graph, and stub logic into `internal/` packages with tests kept green.
- Extract RAG export logic out of `cmd/la-famille/main.go` and into `internal/` or `pkg/`.
- Add static asset sync from `assets/` to `public/`, which matters now that we already have logo and image material in `assets/img/favorites/`.
- Add multi-template support, partials, and layout selection, since the repo already has several templates beyond `layout.html` but the generator still parses a single selected template file.
- Add local dev server plus file watching for markdown and template changes.
- Add GitHub Actions build/test/generate/deploy workflow and later layer on client-side search using `meta.json` plus frontmatter-based taxonomy pages.

## Ready-to-hand TODO

*These are punchy tasks ready to be picked up in the next development cycle:*

- [ ] Refactor generation and export logic out of `cmd/la-famille/main.go` into `internal/` packages without changing behavior.
- [ ] Implement asset copying from `assets/` into build output.
- [ ] Support template partials, multiple page layouts, and layout selection.
- [ ] Add `serve` plus file watching for local authoring.
- [ ] Add CI/CD deploy workflow, then client-side search and taxonomy generation.
