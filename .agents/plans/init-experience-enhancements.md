# Task Plan: Init Experience Enhancements for Users and Models

Task ID: `init-experience-enhancements`
Branch: master

## Problem Statement
When `la-famille init` is run in an empty directory, it creates `config.yaml`, `templates/`, `assets/`, and `content/`. However:
1. No `.gitignore` is created, leading to `public/` and `.la-famille-cache.json` polluting git status.
2. No root `README.md` is created to orient human developers or visitors.
3. No `AGENTS.md` is created to guide AI coding assistants (Claude, Cursor, Copilot, Antigravity) on frontmatter rules, link formats, and validation commands.
4. `init` exits silently without printing a helpful "Next steps:" block on stdout.
5. `about.md` does not link back to `index.md`, leaving backlinks and graph exploration one-way.
6. In `config.yaml`, the commented `default_og_image` references a non-existent file `/assets/default-og.png`.
7. `la-famille new` lacks a `--description` flag and does not scaffold `description:`, causing `la-famille check` to immediately emit a warning on new posts.

## Proposed Changes
- `cmd/la-famille/scaffold.go`:
  - Add templates for `.gitignore`, `README.md`, and `AGENTS.md`.
  - Add `scaffoldProjectRootFiles(dir string)` using missing-only semantics (never overwrite existing files).
  - Update `about.md` with link to `index.md`.
- `cmd/la-famille/main.go`:
  - Invoke `scaffoldProjectRootFiles(root)` during `init`.
  - Print clear confirmation and "Next steps:" output on `cmd.OutOrStdout()`.
- `cmd/la-famille/new.go`:
  - Add `--description` / `-d` flag.
  - Include `description:` in frontmatter scaffold.
- `config.yaml`:
  - Point `default_og_image` to `/assets/img/mascot-default.jpeg`.
  - Regenerate `internal/config/default_config_gen.go`.
- Tests:
  - Add unit tests in `cmd/la-famille/main_test.go` and `cmd/la-famille/scaffold_test.go`.

## Potential Static-Output Impact
None — static generator build output remains strictly backward-compatible.

## Current Status
- [x] Researched `init` and `new` scaffolding and identified developer/agent friction points.
- [x] Created `scaffoldProjectRootFiles` with missing-only semantics for:
  - `AGENTS.md` (detailed operating manual for AI models)
  - `.github/copilot-instructions.md` (official instructions for GitHub Copilot Agent)
  - `.github/workflows/deploy.yml` (automated GitHub Pages deploy workflow)
  - `.agents/plans/README.md` (agent task planning workspace)
  - `.gitignore` (ignores `public/`, `.la-famille-cache.json`, logs, backups)
  - `README.md` (clean project overview and quickstart)
- [x] Enhanced `cmd/la-famille/main.go` to invoke root scaffolding and print interactive "Next steps:" on `init` completion.
- [x] Enhanced `cmd/la-famille/new.go` with `--description` / `-d` and automatic title-based description derivation so newly scaffolded content passes `check` with 0 warnings.
- [x] Enriched `content/about.md` demo content with return links to `index.md`, creating a bidirectional knowledge graph.
- [x] Updated `config.yaml` `default_og_image` example and regenerated `internal/config/default_config_gen.go`.
- [x] Added unit tests in `cmd/la-famille/scaffold_test.go` and `cmd/la-famille/main_test.go`.
- [x] Ran `go test ./...` and `go vet ./...` (100% green).
- [x] Verified full end-to-end lifecycle in fresh scratch repository.
