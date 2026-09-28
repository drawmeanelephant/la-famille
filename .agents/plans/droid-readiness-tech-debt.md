# Agent Readiness: Technical Debt Tracking

## Scope

Add an issue-linked technical-debt marker policy and enforce it with a
repository scanner in CI and the local pre-commit quality hook.

## Ownership and dependencies

- Owner: repository contributors and autonomous coding agents.
- Dependency: Git's tracked-file search and the existing `format_check.sh`.
- Marker syntax: `TODO(#123):`, `FIXME(PROJ-123):`, `HACK(#123):`, or
  `XXX(PROJ-123):`.

## Potential static-output impact

None. The scanner only inspects tracked source and configuration files.

## Tests and validation

- Run the scanner against the current repository.
- Validate shell syntax and the pre-commit quality path.
- Run `go test ./...` and `go vet ./...`.

## Status

Complete. The issue-linked marker scanner runs in CI and through the local
quality hook, and it covers tracked source and configuration files without
flagging historical documentation examples. Valid and invalid marker cases
were tested from a nested repository directory. The full Go test suite and vet
checks pass.
