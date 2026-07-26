---
title: "CLI Contract Tests"
author: "La Famille maintainers"
date: "2026-07-26"
---

# CLI Contract Tests

The command-level tests in `cmd/la-famille` protect boundaries that package unit tests cannot see: Cobra flags, process-level configuration, repository fixtures, generated output, workflow policy, and user-facing errors.

## `main_test.go`

This suite exercises root configuration and CLI behavior, including configuration overrides, initialization, serve/build failure paths, path safety, cache status, and repository workflow assumptions. Several tests use temporary projects or subprocesses so they can validate behavior at the same boundary an operator invokes.

## `check_test.go`

These tests verify the command adapter’s three important outcomes: valid content succeeds, invalid content returns a failure, and warning-only asset findings are reported without incorrectly failing the command. Validation rules themselves live in `internal/checker`.

## `new_test.go`

The `new` tests cover default and custom content locations, nested paths, title/date/frontmatter generation, overwrite refusal and explicit force behavior, and unsafe or symlinked paths. They use temporary directories so a test cannot mutate the repository’s real content tree.

## `pr_test.go`

The PR tests treat command flags, workflow files, and documentation as one policy surface. They protect dry-run defaults, required token and label gates, nightly merge authority, Jules CI’s verify-only posture, and the documented default-branch behavior. Reusable PR policy remains tested in `internal/github`.

## Test design rules

- Assert behavior at the boundary being changed; do not duplicate internal implementation tests in `cmd/`.
- Keep filesystem writes inside `t.TempDir()`.
- Treat stable user-facing error strings as contracts only when the CLI or its docs relies on them.
- When a workflow or documentation string is intentionally policy-bearing, update its contract test in the same change.
- Record gaps honestly instead of claiming coverage from a fixture that does not exercise the path.
