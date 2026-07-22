# Codex Content Scaffold Command Plan (`la-famille new`)

## Objective
Implement a developer-facing `la-famille new` command in `cmd/la-famille` that scaffolds a Markdown content file using project configuration and frontmatter conventions.

## Proposed Changes

### 1. `cmd/la-famille/new.go` [NEW]
Implement `setupNewCmd(cfg config.Config) *cobra.Command`:
- Accepts positional argument for `slug` or `output filename`.
- Supports flags:
  - `--title` / `-t` (string): Title of the post. Defaults to title-cased filename if empty.
  - `--tags` (string slice): Tags for the post.
  - `--layout` (string): Custom layout template.
  - `--date` (string): Publication date (YYYY-MM-DD). Defaults to current local date (`YYYY-MM-DD`).
  - `--force` / `-f` (bool): Force overwrite existing file.
  - `--content` / `-c` (string): Content directory override (defaults to `cfg.ContentDir`).
- Ensures target file path stays safely inside content directory using `pathutil.IsSafePath`.
- Creates parent directories as needed via `os.MkdirAll`.
- Refuses overwrite if file exists unless `--force` is true.
- Generates valid YAML frontmatter compatible with `internal/content` and `internal/checker`.
- Writes file and prints path and next-step guidance to `cmd.OutOrStdout()`.

### 2. `cmd/la-famille/main.go` [MODIFY]
Register `setupNewCmd(cfg)` in `setupRootCmd(cfg)`.

### 3. Documentation [MODIFY]
Update `content/docs/cli.md` to document the new `la-famille new` command and flags.

### 4. Tests [NEW]
Add `cmd/la-famille/new_test.go` with CLI tests for:
- Default scaffolding (date today, auto title from slug).
- Custom flags (`--title`, `--tags`, `--layout`, `--date`).
- Nested directory creation (`blog/nested/post.md`).
- Refusal to overwrite existing file without `--force`.
- Overwriting existing file with `--force`.
- Path traversal protection (attempting to write outside content directory).

## Verification Plan
1. Run `gofmt -w cmd/la-famille/`
2. Run `go test -count=1 ./...`
3. Run `go test -race ./...`
4. Run `go vet ./...`
