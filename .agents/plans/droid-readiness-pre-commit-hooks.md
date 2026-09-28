# Agent Readiness: Pre-commit Hooks

## Scope

Add a checked-in Git pre-commit hook that runs the repository's existing
formatting, vet, module-hygiene, and optional golangci-lint checks. Document
how contributors activate the hook after cloning.

## Ownership and dependencies

- Owner: repository contributors and autonomous coding agents.
- Dependency: the existing `format_check.sh` validation script.
- No source-package or generated-site changes are expected.

## Potential static-output impact

None. The hook only validates the working tree and does not build or modify
static site output.

## Tests and validation

- Validate the hook shell syntax.
- Run the hook from the repository root.
- Confirm the working tree remains clean after validation.

## Status

Complete. The checked-in hook and activation documentation are in place.
Shell syntax, formatting, vet, module hygiene, the full Go test suite, and
`go vet ./...` passed. The local golangci-lint binary is incompatible with
the module's Go version, so its hook stage was validated as an environment
limitation rather than treated as a repository failure.
