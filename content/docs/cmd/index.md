---
title: "Command Architecture"
author: "La Famille maintainers"
date: "2026-07-26"
---

# Command Architecture

The `cmd/la-famille` package is the composition layer for La Famille’s CLI and TUI. It translates user-facing commands and flags into calls to the `internal/` packages, owns process lifecycle concerns such as logging and local servers, and keeps integration contracts close to the binary entrypoint.

These pages describe behavior from the current source and tests. The implementation and its tests remain authoritative when a detail changes.

## Architecture

- [CLI architecture](cli-architecture.md): root configuration, command registration, `check`, `new`, and `pr` orchestration.
- [TUI architecture](tui-architecture.md): Bubble Tea state, menu commands, serving/watch lifecycle, diagnostics, and mascot animation.
- [CLI contract tests](cli-contract-tests.md): how command-level tests protect flags, filesystem behavior, validation, and policy documentation.
- [Release and fixture tests](release-fixture-tests.md): end-to-end fixture builds and generated-output contracts.

## Verification

Run the command package tests directly while changing these surfaces:

```bash
go test ./cmd/la-famille
```

Before publishing a change, run the repository checks:

```bash
go test ./...
go vet ./...
```
