---
title: "CLI Architecture"
author: "La Famille maintainers"
date: "2026-07-26"
---

# CLI Architecture

The command package builds the Cobra tree and adapts operator input to the internal configuration, generator, checker, GitHub, and RAG packages. It should remain thin: policy and reusable behavior belong in `internal/`, while command files own flags, user-facing errors, and process orchestration.

## Root configuration flow

`cmd/la-famille/main.go` establishes the root Cobra command, loads `config.yaml`, validates configuration when a site-dependent command needs it, and registers the subcommands. Configuration-independent commands such as initialization must remain usable when the existing site configuration is missing or malformed; the root command therefore gates site-dependent execution instead of blindly passing an unusable config downstream.

The command package also owns the default logger setup and the `serve` process boundary. The local server binds to the configured port on loopback, serves the configured output directory, and uses bounded HTTP timeouts. Build and serve errors are returned to the CLI rather than silently publishing stale output.

## `check`

`cmd/la-famille/check.go` is an adapter around `internal/checker.Validate`. It:

- creates the check command and exposes its flags;
- loads the effective configuration;
- prints findings with their severity and source;
- returns a failing command result for validation errors;
- allows warning-only findings to be reported without treating them as fatal.

The checker owns validation rules. The CLI owns exit behavior and presentation, so changes to validation semantics should be tested in `internal/checker` while changes to command output or status should be tested in `cmd/la-famille/check_test.go`.

## `new`

`cmd/la-famille/new.go` scaffolds a Markdown content file. The safety boundary is deliberate:

1. derive a safe target from the requested path;
2. reject traversal and symlinked path components;
3. refuse to overwrite an existing file unless `--force` is explicit;
4. write the generated frontmatter and content with restrictive file permissions.

The command uses the configured content directory and supports title, author, date, slug, and category-related input according to its flags. Tests in `new_test.go` cover default output, custom content paths, overwrite behavior, and path safety.

## `pr`

`cmd/la-famille/pr.go` is intentionally a thin entrypoint for the PR synchronization workflow. It gathers flags and environment configuration, enforces the required token/label gates, then delegates synchronization and policy decisions to `internal/github`.

The safe default is dry-run. Merge authority remains in the explicitly configured automation workflow; Jules verification is not merge authority. The command-level tests also audit workflow and documentation strings because those files are part of the operator-facing policy contract.

## Change guidance

When changing a command:

- put reusable rules in the relevant `internal/` package;
- keep Cobra callbacks focused on input, orchestration, and presentation;
- update the same-package contract tests for flags and error text;
- run `go test ./cmd/la-famille` before the full repository checks.
