# #582 Vault Mode shipped-contract acceptance

## Scope and ownership

- Branch: `droid/finish-vault-mode-582`, based on current `origin/master`
  (`08f9a8e` at the start of this pass).
- Accept the shipped wiki links, tags, unresolved-note navigation, target
  search, Ask source cards, and per-note `publish: false` selection.
- Directory exclusion patterns are deferred and are not a closure requirement.
- This is not full Obsidian compatibility, editor integration, or a guarantee
  that arbitrary private vaults or attachments are safe to publish.
- Do not change Corpus Packs code, dependencies, themes, translations,
  attachments, import tools, or retrieval behavior.

## Plan

1. Inspect and reuse the existing tests from #589–#594. Map all seven fixed
   acceptance rows to evidence.
2. Add only missing regression assertions or narrowly necessary fixes. Use
   synthetic content with a unique unpublished-body canary.
3. Compile the current CLI and demonstrate `build`, `check`, `rag`, and
   `publish-check` on a small synthetic vault. Verify initial stubs and a
   rebuild after adding the missing note.
4. Reuse the deterministic two-note RAG/Ask integration check for correctly
   titled and linked source cards, without claiming real-model quality.
5. Run focused tests, `./format_check.sh`, `go test ./...`, `go vet ./...`,
   race tests for filesystem changes, and shuffled repeated tests for test
   changes.
6. Write a concise acceptance record with commands, results, output paths,
   row-by-row evidence, and limitations. Open one PR against `master`.
   Close #582 only after review and merge against the documented scope.

## Dependencies and static-output impact

- Reuse the existing generator, checker, search, RAG, and Ask contracts.
- No new dependencies, release, deployment, or installed model.
- No static-output format or asset-pipeline breaking changes are intended.
  Any discovered publication leak is a blocker requiring a narrow fix and
  regression coverage before acceptance.
- Generated demonstration artifacts stay outside tracked source.

## Status

- Existing tests establish rows 1–6; focused baseline tests passed.
- Expanded row-7 checks cover rendered and raw excluded notes, private-only
  taxonomy, metadata/manifest, feed/sitemap, and generated-output body canary.
- The new check exposed early AST-walk termination when an excluded wiki link
  is removed, leaving subsequent Markdown links into excluded notes intact.
  Fixed by deferring excluded-link unwrapping until the walk finishes; a
  regression verifies public links after exclusions still transform.
- Current master fetched and fast-forwarded on the separate task branch.
- Existing hook already points to `.githooks`.
- Synthetic demonstration fixture added under `assets/testdata/vault-acceptance`.
  Compiled CLI builds before/after filling the stub, exports RAG, and passes
  final source/manifest checks and strict publish-check.
- Initial source check correctly reports the intentionally unresolved link.
  Final source checks have zero errors, warnings, and orphans.
- Row-5 browser check passes through the shipped search input handler; row-6
  existing two-note fake-provider integration passes.
- Body-canary scan passes across all generated files and both content bundles.
- Formatting/vet/module/debt/lint, full tests, race tests, and repeated
  shuffled tests pass. The old installed linter was incompatible; rebuilt
  task-local golangci-lint 2.14.0 with Go 1.27.1, without changing dependencies.
- Acceptance evidence: `docs/acceptance/vault-mode-582.md`.
- Implementation and acceptance complete. PR targets `master`; issue closure
  waits for review and merge.
