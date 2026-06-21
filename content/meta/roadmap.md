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
- Add static asset sync from `assets/` to `public/`, which matters now that we already have logo and image material in `assets/img/favorites/`.
- Add template partials support, since the repo already has layout selection via frontmatter.
- Add local dev server plus file watching for markdown and template changes.
- Add GitHub Actions build/test/generate/deploy workflow and later layer on client-side search using `meta.json` plus frontmatter-based taxonomy pages.

## Ready-to-hand TODO

*These are punchy tasks ready to be picked up in the next development cycle:*

- [ ] Refactor generation logic out of `cmd/la-famille/main.go` into `internal/` packages without changing behavior.
- [ ] Implement asset copying from `assets/` into build output.
- [ ] Support template partials.
- [ ] Add `serve` plus file watching for local authoring.
- [ ] Add CI/CD deploy workflow, then client-side search and taxonomy generation.

## TUI Improvements
- Track build time stats and show them in the Stats screen.
- Track RAG sizes and represent them in terms of LLM context windows in the Stats screen.
- Better graphics and more options for the mascot Jules (e.g. Jules themes, different animations).
