# PR 389 — Publishing output contract

## Scope

- Document the generated publishing artifacts in `content/docs/publishing.md`.
- Extend the release smoke fixture to cover `render: false`, configured and empty `SiteURL`, canonical/OG metadata, and deterministic publishing outputs.

## Dependencies

- PRs 384–388 are already merged into `master` and are included in this branch after rebasing.

## Potential static-output impact

None intended. This task documents and tests existing output behavior.

## Verification

- `go test ./...`
- `go vet ./...`
- `git diff --check`

## Status

Ready for review after the rebase and validation pass.
