---
date: "2026-08-13"
title: "Changelog"
author: "Jules"
---

# Changelog

This page is the curated, site-published changelog. **GitHub Releases are the source of truth** — each tag at `https://github.com/drawmeanelephant/la-famille/releases` is published with auto-generated notes from `release.yml:165` (`gh release create --verify-tag --generate-notes` over the verified tag commit; see `RELEASE-QUICKSTART.md` and `.github/scripts/release/tag.sh`). This page copies the highlights at release time and adds human context; it is updated once per release, not per PR.

> **Convention (hybrid, decided 2026-08-20 for #466):** No per-PR changelog gate. Keep this file tidy and de-duplicated; link to the Release for the full PR list. See `plan.md:15` roadmap and milestone `1. Release Readiness` (`content/meta/roadmap.md:20`).

## 2026-08-13 — Sunset + ask/build fixes

- **Content:** Sunset the soundtrack albums, cat facts, the genesis retrospective, and the "Godfather of Farts" page. Removed from the live site to keep it aligned with current project reality (history preserved in git).
- **Docs:** Consolidated four duplicate root-level component-mapping reports into a single [Architecture & Component Map](architecture.md).
- **Docs:** Rewrote the [Help Menu](../help.md) to link real documentation pages instead of stub links.
- **Docs:** Merged the meta changelog into this page; removed the retired cat-facts routine.
- **Ask:** The `ask` server now serves the generated site from the output directory, so "Open source" citation links resolve to the pages they name instead of reopening the assistant UI (`PR #463`, `issues-420-422.md`).
- **Build:** Case-only output collisions now follow the output filesystem. They still fail the build on case-insensitive hosts (macOS, Windows); on case-sensitive hosts (Linux) both outputs are published with a warning instead (`PR #463`).

## Recent Updates (pre-convention archive)

*   **Merge PR #277**: Refactor serve subcommand for graceful shutdown using signal context.
*   **Merge PR #275**: Add ReadTimeout and WriteTimeout to http.Server initializations in cmd/la-famille/main.go and cmd/la-famille/tui.go.
*   **Merge PR #274**: Optimize asset copy logic to skip git check-ignore if binary is missing.
*   **Merge PR #273**: Nightly maintenance routine, including normalizing yaml frontmatter keys to lowercase.
*   **Merge PR #272**: Docs: zhuzh tui.md guide.
*   **Merge PR #271**: Feat: Add and run Nightly Documentation Zhuzh Pass routine.
*   **Merge PR #270**: Docs: add component mapping and micro-improvement audit report.
*   **Merge PR #268 / PR #269**: Fix SiteLinks config and build before serving. Add docs-reality-pass routine.
*   **Merge PR #266**: Feat(stub): improve missing page stub content.
*   **Merge PR #265 / PR #264**: Refactor markdown engine to internal/markdown and execute cat facts routine.
*   **Merge PR #263**: ci: add golangci-lint pipeline and fix errors.
*   **Merge PR #261**: refactor: apply cyberpunk template updates.

## 2026-06-19
- **Templates:** Added a centered minimalist layout template.
- **Templates:** Added a Cyberpunk sidebar layout template.
- **Export:** Implemented RAG (Retrieval-Augmented Generation) export logic.

## 2026-06-18
- **Core Engine:** Implemented YAML frontmatter parsing.
- **Core Engine:** Implemented Markdown link resolution and render control.
- **Accessibility:** Added "skip to content" link and semantic navigation (Palette persona).
- **Core Engine:** Simplified extension removal logic and added unit tests for `relPathFromTo`.
- **Infrastructure:** Established automated release strategy and soundtrack integration (soundtrack since sunset, 2026-08-13).
- **Content:** Added soundtrack entries and lore.
- **CLI/Configuration:** Migrated to `spf13/cobra` with a `build` subcommand for better CLI configuration and flags.

[Go back to Index](index.md)