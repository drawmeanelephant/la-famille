---
date: "2026-08-13"
title: "Changelog"
author: "Jules"
---

# Changelog

This page is the curated, site-published changelog. **GitHub Releases are the source of truth** — each tag at `https://github.com/drawmeanelephant/la-famille/releases` is published with auto-generated notes from `release.yml:165` (`gh release create --verify-tag --generate-notes` over the verified tag commit; see `RELEASE-QUICKSTART.md` and `.github/scripts/release/tag.sh`). This page copies the highlights at release time and adds human context; it is updated once per release, not per PR.

> **Convention (hybrid, decided 2026-08-20 for #466):** No per-PR changelog gate. Keep this file tidy and de-duplicated; link to the Release for the full PR list. See `plan.md:15` roadmap and milestone `1. Release Readiness` (`content/meta/roadmap.md:20`).

> **Versioning (settled at v0.2.0):** semver tags `vX.Y.Z` are the only version source — `--version` is stamped by linker flags from the tag, there is no version file. While pre-1.0: minor bumps (`v0.X.0`) mark feature drops, patch bumps (`v0.X.Y`) mark fix-only releases, and prerelease suffixes (`-prealpha`, `-rc.N`) mark preview cuts.

## 2026-10-11 — v0.2.0: packs, ledgers, and vaults

The first stable-line release ([full notes](https://github.com/drawmeanelephant/la-famille/releases/tag/v0.2.0)). Three feature epics landed since the pre-alpha cut, alongside a deep hardening pass; one supported surface was retired.

- **Corpus Packs:** `la-famille pack` builds, verifies, diffs, applies, publishes, pulls, and watches content-only site archives — deterministic output, member-level integrity checks, local and HTTPS feeds, and a durable `pack watch` subscriber that re-pulls when verification fails.
- **Change Ledger (#580):** every build emits a deterministic `site-manifest.json`; `la-famille diff` compares two manifests — output directories, manifest files, or Git revisions — and `diff.json`/`diff.txt` plus a `--gate` flag make a site diff reviewable, scriptable, and gate-able in CI.
- **Vault Mode (#589–594):** `[[wiki links]]`, body `#hashtags` in the taxonomy, an unresolved-notes index, and `publish: false` frontmatter to keep working notes out of the shipped site. Site search indexes wiki-link targets.
- **Themes:** octoburger becomes the flagship default and the bundled packet is fully self-contained — no CDN. `init` scaffolds a goldmark showcase page and a warm homestead starter set (AGENTS.md, deploy workflow, next-steps greeting).
- **TUI:** octoburger glow-up — progress, cook spinner, confetti; `serve` honors the base path, builds asynchronously, and reports watch errors.
- **Site & publishing:** flagship website plus a Cloudflare publishing path; `publish-check` honors the site base path.
- **Removed (breaking):** `la-famille ask` and the local-assistant/Ollama stack are retired. RAG export (`la-famille rag`), site search, and Corpus Packs carry that work forward.
- **Hardening:** the post-prealpha WOLF review (three tiers), dogfood triage (#506–#518), and a closing sweep across pack pulls, durable watch, cache, symlinks, taxonomy identity, graph edges, markdown panics, and TUI serve.

## 2026-08-25 — v0.1.0-prealpha: the first cut

The first tagged release ([full notes](https://github.com/drawmeanelephant/la-famille/releases/tag/v0.1.0-prealpha)). Intentionally pre-alpha: rough edges are expected, and black-box review findings (#500) drive the next iteration. Because it is the first tag, the generated notes list every merged PR in project history; the highlights below are what the release actually means.

- **Release pipeline:** provenance-verified tags — each archive is built only after CI proves the checked-out commit equals the tag commit (`tag.sh`); six platform targets (linux/darwin/windows × amd64/arm64), `SHA256SUMS`, and a native linux/amd64 offline smoke test run before anything publishes. Semver prerelease suffixes are first-class.
- **Bundled theme packet (#497):** the binary embeds its themes — default layout, octoburger flagship (Raoul(s) holding the burger while you write), and terminal. `init` installs the packet missing-only so frontmatter switching works without a source checkout.
- **First-run experience (#498):** `la-famille themes` lists bundled layouts with one-line descriptions, config-free. `init` scaffolds a starter homepage plus a theming demo page that switches layout per-page; README and RELEASE-QUICKSTART document picking/switching on binary-only installs.
- **Publishing validation (#474/#482/#483):** `publish-check` validates a generated `public/` tree against the output contract; `check` reports a concise summary of broken links, missing metadata, asset health, and orphans.
- **Site hygiene:** broken showcase hrefs repaired with clean-URL fallback (#488/#487), soundtrack/cat-facts content sunset (2026-08-13).

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