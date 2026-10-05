# Issue #629: retire Ask This Site

## Scope and ownership

- Owner: Droid on `droid/debug-la-famille-issue-629`.
- Remove the Ask CLI, TUI screen/lifecycle, server/UI, Ollama/fake providers,
  evaluation harness, and exclusively Ask-owned retrieval/cache code and fixtures.
- Audit shared consumers before removal. Keep `internal/ragexport`,
  `internal/ragfmt`, site search, and Corpus Pack build/verify/diff/feed/watch work.
- Replace subscriber tests that depend on Ask with archive-level compatibility
  checks. Remove only the Ask-specific pack-to-retrieval adapter.
- Update current documentation, navigation, roadmap, and CI. Keep historical
  plans and reports, marking obsolete demo instructions as historical.
- Link the PR to #629 and the superseded #602 measurements follow-up.

## Breaking changes and static-output impact

- Intentional: `la-famille ask`, all Ask flags/API/UI, and the TUI Ask entry
  are removed. The command must fail as unknown rather than launch a service.
- No replacement provider, runtime, or dependency.
- No changes to generated search/metadata/graph assets, RAG archive encoding,
  or Corpus Pack schema and delivery semantics. Repository documentation and
  navigation change to reflect retirement.
- Never remove existing user caches, archives, content, or installed models.
  Preserve the publishing guard against accidentally exposing legacy caches.

## Implementation and validation

1. Audit Ask references and package/fixture dependencies on current master.
2. Add CLI/TUI retirement regressions and preserve RAG/search/pack coverage.
3. Remove owned implementation and update current docs, navigation, and CI.
4. Run focused tests, `./format_check.sh`, `go test ./...`,
   `go test -race ./...`, `go vet ./...`, and shuffled repeat tests.
5. Smoke-test build/check/serve/search/RAG/packs without any model runtime.
6. Review the diff, commit, push, and open a PR against `master`.

## Status

- Planning: issue read; branch fast-forwarded to current master.
- Audit: Ask-only pack corpus loader found; core pack operations are separate.
- Implementation: Ask CLI/TUI/packages/assets/eval fixtures removed. Shared
  pack subscriber fixture renamed; subscriber coverage now checks archive
  framing, updated text, and page/search/graph metadata directly.
- Documentation: retired guide kept as a permalink notice; current feature
  advertising and navigation replaced. Historical reports/proposals retained
  with retirement context. Legacy-cache publishing protection retained.
- Focused validation: new CLI/menu regressions failed before removal and now
  pass. All changed packages pass their suites, including RAG escaping,
  subscriber pull/watch, snapshot replacement, and flagship layout tests.
- Full quality checks: `go test ./...`, `go vet ./...`,
  `go test -race ./...`, and shuffled repeated tests passed. Coverage passed
  at 84.8%, including the final legacy-data regression.
- `./format_check.sh`: passed with temporary golangci-lint v2.14.0 built
  using the current toolchain (0 issues). The globally installed linter is
  built with Go 1.24 and cannot target this module's Go 1.26. The CI-pinned
  v2.13.1 source install also failed to resolve its own godot package; no
  global tool, module, or CI version was changed.
- One existing timing-sensitive `TestBuild_BenchColdVsWarm/n=100` failed
  during concurrent build activity (94.6 ms warm versus 90.2 ms cold).
  Rerunning the complete suite without competing smoke builds passed;
  generator code and this test were not changed.
- Smoke checks: build/check/publish-check/RAG/pack build+verify/serve/search
  passed with an empty executable PATH and no Ollama daemon. A listener on
  port 11434 detected zero attempted connections. Legacy cache bytes stayed
  untouched; the new regression also preserves user archives and model files.
- Baseline comparison: a representative fixture with a homepage produced
  19 static files and 3 RAG archives byte-identical to current master with
  equal cold-build baseline state. The initial fixture lacked a homepage;
  the smoke was adjusted, not the generator or publishing contract.
- Final checks: tracked file-size and AGENTS.md guards passed; staged diff
  passed `git diff --cached --check`. No active deleted-package, UI, or CI
  references remain outside historical plans/reports.
- Handoff: implementation and validation complete. Open the reviewed branch
  against `master` for #629; close the superseded #602 follow-up as not planned.
