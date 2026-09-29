# Task Plan: Moonshot Ideation (`docs/MOONSHOTS.md`)

## Task ID
`moonshot-ideation` (branch: `moonshot-ideation`)

## Objective

Produce `docs/MOONSHOTS.md`: a strategic, moonshot-level proposal set for
la-famille grounded in the actual codebase. Deliverable is **documentation
only — no code changes**. The PR adds the doc; it does not implement any
moonshot.

## Scope

In scope:
- Read the repo (README, `content/docs/`, `content/meta/`, `internal/`, the
  GitHub issue tracker) before proposing anything.
- Explore 8–12 candidate moonshots, then rank a top 5 by
  `(impact × identity fit) / cost`.
- Each moonshot: WHAT (one paragraph), why la-famille specifically, 3+ phases
  with per-phase observable done-criteria, honest cost in agent-days, biggest
  risk.
- Blue-sky proposals explicitly labeled.
- At least one moonshot ladders directly toward the honest anchor: shipping a
  real site with la-famille.
- State what is cut and why.

Out of scope:
- Implementing any moonshot.
- Touching the generator, TUI, CLI, or any Go source.
- Editing the root `plan.md` (per AGENTS.md planning policy).

## Grounding research performed

- Read `README.md`, `plan.md`, `content/meta/roadmap.md`,
  `content/meta/aspirations.md`, `content/docs/{architecture,ask,rag,frontmatter,search}.md`.
- Surveyed all 30 `internal/` packages (~35.4k LOC Go, 85 test files).
- Read `.agents/plans/` (58 task plans) — including the prior
  `ask-moonshot.md` to avoid re-proposing already-shipped work.
- Enumerated every GitHub issue and open PR: **0 open issues, 0 open PRs**
  (all issue history is CLOSED). Nothing is currently queued, so a moonshot
  doc fills a real vacuum rather than duplicating a backlog.

## Key findings that shaped the proposals

1. **The identity is "site as a machine-readable knowledge corpus."** la-famille
   is the only SSG that simultaneously emits a link graph
   (`graph.json`/`backlinks.json`), an LLM-readable archive
   (`rag-system.md`/`rag-content.md`/`rag-config.md`), a local
   citation-grounded assistant (`ask` + Ollama + BM25-lite), a static
   interactive graph explorer (`/graph/index.html`), a Bubbletea TUI site
   manager, and GitHub PR automation (`pr sync`). No generic "add a feature"
   proposal should compete with keeping that bundle coherent.
2. **The content model is a fixed 17-field struct** (`internal/content.FileMeta`:
   Title, Author, Date, Tags, Categories, Slug, Layout, Description, Image,
   Render, VideoScript, AnimationCues, SoundtrackTheme, ComplianceModal,
   RelPath, Content, Rest, Warnings). There is no custom/extra frontmatter
   passthrough, no data files, no pagination, no redirects/aliases, no
   series, no i18n. These are the concrete gaps a real (non-self-referential)
   site will hit first.
3. **`internal/stub` already implements the Obsidian "unresolved link
   creates a note" behavior** (auto-generates stub pages for links to missing
   files). This is an accidental, load-bearing head start for a knowledge-vault
   moonshot.
4. **The RAG format is a designed, reversible, escaped line-oriented envelope**
   (`internal/ragfmt`: `<file path><content>` with marker escaping so the
   archive can bundle the project's own Go sources and the format's own docs
   without self-corrupting). This is a real interchange seam, not a hack.
5. **The honest anchor holds:** the current `content/` tree is the project's
   own docs/showcase/devlog site. No real destination site has been shipped.
   The generator has never built a site about something other than itself.

## Static-output impact

None. This PR adds one Markdown file under `docs/` (repository
documentation) and one plan file under `.agents/plans/`. It does not modify
`content/`, `templates/`, `assets/`, or any Go source, so the generated
`public/` tree and all build artifacts are byte-for-byte unchanged. The doc
is intentionally placed in root `docs/` (repo docs) rather than `content/docs/`
(site content) so it is not rendered into the published site and does not
create a new page in the site graph or search index.

## Verification plan

1. `gofmt -l .` — no Go files touched, expect empty.
2. `go vet ./...` — expect clean.
3. `go test ./...` — expect pass (unchanged; no source diff).
4. `bash format_check.sh` — expect "format check passed" (note: requires
   `go mod tidy` to be clean; docs-only change cannot affect it).
5. Confirm the only changed files are `docs/MOONSHOTS.md` and this plan via
   `git status`/`git diff --stat` (no `git add -A`).

## Status

- [x] Read repo + docs to ground proposals
- [x] Survey `internal/` packages
- [x] Check issue tracker (0 open issues / 0 open PRs)
- [x] Create branch `moonshot-ideation` off `master`
- [x] Write this task plan
- [x] Explore 8–12 candidate moonshots
- [x] Rank top 5 by (impact × fit) / cost; document cuts
- [x] Write `docs/MOONSHOTS.md`
- [x] Final validation + open PR
