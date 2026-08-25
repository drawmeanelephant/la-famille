# Milestone: v0.1.0-prealpha

Task ID: `milestone-v0-1-0-prealpha`
Type: roadmap-maintenance (tracker only; no source changes in this task)

## Goal

Create the GitHub milestone that gets La Famille to its first consumable,
pre-alpha release: one static binary that builds sites fully offline, packaged
with a curated theme set so a person (or a review agent) can download a single
archive and work in it without a Go toolchain or source checkout.

## Current state (verified 2026-08-25)

- Release pipeline exists and is hardened: `.github/workflows/release.yml`
  builds 6-platform archives with SHA256SUMS, exact-tag provenance checks, and
  auto-generated notes. It has never run: no tags exist, 0 releases.
- Binary embeds only `templates/layout.html` + search partial + runtime assets
  (`internal/runtimeassets/assets.go`). The other ~19 layouts in `templates/`
  are unreachable from a released archive.
- Version command (`--version`, `--json`) and full offline CLI smoke test exist.
- Curated changelog convention decided (#466): GitHub Releases are source of
  truth; `content/docs/changelog.md` is a per-release curated snapshot. Never written.
- Tracker state: 0 open issues; milestone #4 (Content Quality) complete.

## Deliverables (as tracker items)

1. Curated theme set embedded in the release binary, selectable by binary-only
   users (no breaking change to existing projects).
2. First-run/docs experience: quickstart shows theme switching; scaffolded site
   exercises the chosen theme.
3. Cut `v0.1.0-prealpha`: push tag, verify pipeline output, write curated
   changelog snapshot.
4. Black-box review pass: hand archives to fresh agent sessions with no repo
   access; triage findings into issues that feed the next milestone.

## Static asset generation impact

None in this task. Issues 1–2 will touch the embedded-template pipeline
(`internal/runtimeassets`, `internal/render` contract harness); those PRs must
carry their own plans and tests per AGENTS.md.

## Handoff

- Milestone created on GitHub with the four issues above attached.
- Implementation tasks branch off individually; each gets its own
  `.agents/plans/<task-id>.md`.
