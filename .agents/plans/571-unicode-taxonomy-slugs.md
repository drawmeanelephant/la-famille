# Issue #571: preserve native-language taxonomy terms

## Scope and ownership

- Own taxonomy normalization in `internal/content`, diagnostics in `internal/checker`,
  and archive link generation in `internal/taxonomy`.
- Keep the existing ASCII normalization contract and filesystem byte limit.
  Allow Unicode letters, digits, and combining marks in a single path
  component; do not add a transliteration dependency or a mapping setting.
- Use one normalization rule for build and `check`. Invalid terms that cannot
  produce an archive should be actionable errors in `check`, not silent losses.
- Add package-level regression tests for CJK tags/categories, byte boundaries,
  encoded archive URLs/links, and rejected values.

## Dependencies

- Fast-forward this task branch to the latest `origin/master` before editing
  production code. No new external packages.

## Static-output and compatibility impact

- Non-ASCII taxonomy terms that previously vanished or lost characters now
  produce native-language archive directories (for example `tags/起始/`) and
  escaped URL paths (for example `/tags/%E8%B5%B7%E5%A7%8B/`). Existing
  `café`-like terms change archive URLs from their ASCII-stripped form.
- ASCII-only terms and their URLs should remain unchanged. `check` becomes
  stricter for genuinely unusable (empty or overlong) taxonomy values.
- Search entries, sitemap/canonical URLs, article links and archive index links
  must all agree on the same published URL.

## Verification and status

- [x] Write failing regressions in the affected packages.
- [x] Implement Unicode-preserving normalization and consistent diagnostics.
- [x] Run targeted package tests, `go test ./...`, and `go vet ./...`.
- [x] Run `format_check.sh` with the compatible Homebrew `golangci-lint`
  selected ahead of the older binary in `~/.local/bin`.
- [x] Review the staged diff; open the PR against `master` after committing.
