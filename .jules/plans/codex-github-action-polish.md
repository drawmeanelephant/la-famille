# Codex GitHub Action Developer Experience Polish Plan

## Objective
Improve La Famille’s reusable GitHub Action developer experience by auditing inputs/defaults, adding configurable CLI parameters, passing inputs safely into the build flow, documenting usage in `README.md`, and adding validation & unit tests while preserving existing defaults and deployment behavior.

## Proposed Changes

1. **CLI / Build Flags (`cmd/la-famille/main.go`)**:
   - Add `--site-url` (and `--siteurl` alias) flags to `buildCmd`.
   - Update `buildCmd.RunE` to override `cfg.SiteURL` when provided and execute `cfg.Validate()`.

2. **GitHub Action (`action.yml`)**:
   - Audit and declare configurable inputs with sensible defaults:
     - `content-dir` (default: `'content'`)
     - `output-dir` (default: `'public'`)
     - `template` (default: `'templates/layout.html'`)
     - `site-url` (default: `''`)
   - Safely pass inputs via step environment variables into `go run ... build` array arguments.

3. **Documentation (`README.md`)**:
   - Update the GitHub Action section with a minimal and configurable usage example.

4. **Testing & Verification (`cmd/la-famille/main_test.go`)**:
   - Update `TestCommandFlags` to verify `"site-url"` flag presence.
   - Add unit tests for `--site-url` flag input mapping, configuration override, and validation.
   - Run `gofmt`, `go test -count=1 ./...`, `go test -race ./...`, `go vet ./...`.
