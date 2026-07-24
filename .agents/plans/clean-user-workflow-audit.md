# Clean User Workflow Audit & Fix Plan

## Task ID
`clean-user-workflow-audit`

## Objective
Audit La Famille from a clean temporary directory as a new user, document workflow pain points, and implement high-value CLI/docs/workflow fixes so that a clean user can follow one documented path from `init` -> `new` -> `build` -> `check` -> `rag` -> `serve` cleanly.

## Reproducible Pain Points Identified
1. **`la-famille init` Incomplete Environment Setup**:
   `la-famille init` currently only creates `config.yaml`. Running `la-famille new page` followed by `la-famille build` fails because `templates/layout.html` does not exist.
   *Fix*: Update `init` command to scaffold a default `templates/layout.html` template when initializing a workspace, ensuring `build` works immediately after `init`.

2. **`la-famille new` Path Duplication Bug**:
   If a user runs `la-famille new content/posts/my-post.md`, `new` prepends `content/` without checking, producing `content/content/posts/my-post.md`.
   *Fix*: Normalize/trim prefix if target filepath already starts with the target content directory.

3. **`la-famille serve` Continues on Initial Build Failure**:
   When `serve` encounters an initial build error (e.g. missing layout or malformed content), it logs the error but continues to start the HTTP server.
   *Fix*: Fail early and exit cleanly if the initial build fails in `serve`.

4. **Documentation Alignment**:
   Ensure `README.md` and `content/docs/setup.md` accurately document the single clear path from `init` to generated site.

## Proposed Code Changes
- `cmd/la-famille/init.go`: Ensure default `templates/layout.html` is scaffolded during `init`.
- `cmd/la-famille/new.go`: Strip redundant content directory prefix if provided in target file path.
- `cmd/la-famille/serve.go`: Exit with error if initial build fails.
- `cmd/la-famille/init_test.go`, `cmd/la-famille/new_test.go`, `cmd/la-famille/serve_test.go`: Add tests covering new behaviors.
- `content/docs/setup.md` / `README.md`: Verify and update setup walkthrough if needed.

## Verification Plan
- Run unit tests: `go test ./...`
- Run static checks: `go vet ./...`
- Verify clean-directory end-to-end workflow in `.tmp_clean_user`:
  1. `la-famille init`
  2. `la-famille new test-post`
  3. `la-famille build`
  4. `la-famille check`
  5. `la-famille rag`
  6. Verify all exit 0 and produce valid outputs.
