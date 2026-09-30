# Issue #570: RAG export from a relative subdirectory

## Scope and ownership

- Fix the CLI's repeated resolution of an explicit relative `--project-root`
  for `rag` (and the same root-resolution path in `build`). The bootstrapper
  already resolves the flag against the invocation directory.
- Preserve the documented rule that relative RAG input and output overrides
  are relative to the project root. In particular, use
  `--output public/rag-archive` from the parent directory, not
  `--output sites/zai/public/rag-archive`.
- Add CLI package regression tests using a parent-directory invocation and
  a nested site with populated content, source, assets, and templates.
- Clarify the parent-directory invocation in the RAG guide.

## Dependencies

- Fast-forward to the latest `origin/master` before editing. No new packages.

## Static-output and compatibility impact

- No intentional change to archive formats or published URLs. Previously
  empty or misplaced exports from a relative subdirectory now read the
  selected site's files and write to its selected archive directory.
- Relative `--output` remains project-root-relative; CWD-relative paths that
  include the project-root prefix are not reinterpreted.

## Verification and status

- [x] Add failing CLI regressions for relative project root and output forms.
- [x] Fix root handling in `rag` and `build` without changing output semantics.
- [x] Run focused tests, `go test ./...`, and `go vet ./...`.
- [ ] Review diff, commit, and open PR against `master`.
