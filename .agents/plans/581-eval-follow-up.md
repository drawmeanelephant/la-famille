# #581 — Evaluation follow-up to #587

## Context

Basic Phase 1 merged in #587 while the original local checkout was stale.
This worktree starts at master d3f2515 and preserves the original checkout and
its uncommitted work. User approved adapting the useful additions and opening
a draft PR. Do not replace the frozen phase-1 JSON or break Run(ctx, Options).

## Plan

- [x] Extend existing version-1 nested dataset/API additively: precision gates,
  measurement-only zero-recall probes, classes and graph-path diagnostics.
- [x] Build content-only fixtures in isolated temporary projects with real
  generator/exporter, confined reads, no symlinks or source-dir collisions.
- [x] Retain original tests and golden dataset unchanged; add two twelve-page
  fixtures and a hard set in the existing nested schema. Record both baselines.
- [x] Add focused regression tests and update docs without changing production
  ranking, citation verifier or existing ask host/content-dir behavior.
- [ ] Validate tests, vet, race, formatting and quality hooks; commit owned
  files, push dedicated branch and open draft referencing #581 and #587.

## Static pipeline risks

No production build configuration/fingerprint, generator or assets changes.
Evaluation writes outputs, cache and archives only into disposable projects.
Content copies go into canonical content/ regardless of source directory name.
Page labels remain source-root-relative. Root confinement prevents external
symlink reads; fixture symlinks are rejected. Fixture-specific config/layout/
assets are not copied, so this is retrieval evaluation rather than visual QA.

## Scope honesty

Original dataset remains the regression contract. Hard probes are synthetic
keywords, not held-out human reader questions. Strict abstention requires zero
retrieval plus canonical fallback; no model call is attempted for nonempty
near-miss retrieval (which already fails this gate). Path coverage is not
answer provenance. Embeddings, fusion and graph expansion/UI remain deferred.

## Results

Full `go test ./...`, `go vet ./...`, `go mod tidy -diff`, targeted package/CLI
race checks, and changed-file `gofmt -d` passed. Original tests pass unchanged.
The frozen #587 set measures recall 1.0000, precision 0.4375 (9/9 gates).
Hard set measures recall 0.5833333333, precision 0.3333333333 (7/8 gates), with
warranty near-miss abstention intentionally failing. Both fixtures have exactly
12 published pages; path coverage is 2/3 with no lexical bridge token leakage.

Added tests for schema/size/threshold validation, deduplication, aggregate and
question metric gates, cancellation, K overrides, CLI error visibility, fixture
root escape, file/directory symlinks, source-name collisions, graph identity,
real citation URLs, stale labels and invented graph hops. No dependency drift.

The checked-in hook passed formatting, vet, module hygiene and technical-debt
checks, but local golangci-lint cannot load the Go 1.26 project: its binary was
built with Go 1.24. User explicitly approved a one-command hook bypass and
requiring compatible CI lint; no git configuration is changed.

Opening a draft review is the remaining delivery step; no closure of #581.
