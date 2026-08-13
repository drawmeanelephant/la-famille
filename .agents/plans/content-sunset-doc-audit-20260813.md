# Task: Content Sunset & Documentation Coherence Audit

Task ID: `content-sunset-doc-audit-20260813`

## Scope

Remove retired/legacy content from the live site and reconcile documentation so
the repository reflects reality. No code changes.

## Content sunset (removed)

- `content/soundtrack/` (albums, routine-tasks, index) — soundtrack feature sunset
- `content/catfacts/` — cat-facts routine retired
- `content/the_godfather_of_farts.md` — one-off joke page
- `content/genesis_retrospective.md` — superseded retrospective

## Duplicate/legacy documents (consolidated or removed)

- Root `architecture_report.md`, `architecture_review.md`, `audit_report.md`,
  `report.md` — four near-duplicate component-mapping audits. Consolidated into
  one canonical `content/docs/architecture.md`; the other three removed.
- `content/meta/changelog.md` merged into `content/docs/changelog.md`
  (single canonical changelog).
- `content/jules/cat-facts-routine.md` removed (dead routine).

## Reference cleanup (reality must match docs)

- `content/index.md` — drop Catfacts and Soundtrack sections
- `content/help.md` — rewrite to link real docs (no more stale stub links)
- `content/meta/aspirations.md` — mark shipped/sunset status per item
- `content/meta/mascot-raouls.md` — fix soundtrack reference + wrong 2024 date
- `content/meta/index.md` — link mascot and changelog
- `content/docs/index.md` — link architecture doc
- `content/docs/changelog.md` — merge meta entries, add 2026-08-13 sunset entry
- `content/jules/running-routines.md` — drop soundtrack completion step
- `content/jules/index.md` — drop cat-facts routine link
- `AGENTS.md` / `GEMINI.md` — drop nonexistent `pkg/` directory from structure

## Breaking changes to static asset pipeline

None. All removed content was authored markdown; no code, templates, assets, or
fixtures reference `content/soundtrack/` or `content/catfacts/`. The
`soundtrack_theme` frontmatter field (multimedia devlog feature) is untouched.

## Verification

- `grep` for dangling references to removed paths
- `go test ./...`, `go vet ./...`
- `la-famille build` and confirm removed pages no longer appear in `public/`
