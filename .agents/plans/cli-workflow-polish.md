# CLI Workflow Polish Implementation Plan

## Task ID
`cli-workflow-polish`

## Objective
Implement remaining CLI workflow polish for La Famille:
1. Make `la-famille init` create expected `content/` and `assets/` directories alongside `config.yaml` and `templates/layout.html`.
2. Add `--categories` flag to `la-famille new`.
3. Ensure scaffolded frontmatter includes `categories` properly.
4. Make `new` follow-up hints use the configured/resolved content directory instead of assuming `content`.
5. Add focused CLI unit tests for fresh init, existing init, categories, custom content directories, and path safety.

## Guardrails
- Do not reopen output-overlap, symlink, or init-overwrite fixes.
- Do not touch themes or TUI design.
- Use `.agents/plans/<task-id>.md`, never root `plan.md`.

## Proposed Changes
- `cmd/la-famille/main.go`: Ensure `init` creates `content/` and `assets/` directories.
- `cmd/la-famille/new.go`: Add `--categories` flag, update `frontmatterData` struct, and adjust follow-up hint for non-default content directories.
- `cmd/la-famille/main_test.go`: Add/update tests for fresh init (creating `content/` and `assets/`) and existing init behavior.
- `cmd/la-famille/new_test.go`: Add tests for `--categories`, custom content directory hints, and path safety.

## Verification
- `go test ./cmd/...`
- `go test ./...`
- `go vet ./...`
