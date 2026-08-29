# Task Plan: fix-wolf-review-tier1

## Issues addressed (umbrella #500 — WOLF review backlog)

Self-contained code-review findings from the v0.1.0-prealpha pass, batched into
one branch/PR off `origin/master`:

- **#543** `internal/asset/copy.go` — `CopyFile` opens the destination with
  `os.O_RDWR|os.O_CREATE|os.O_TRUNC`. A copy only ever writes; switch to
  `os.O_CREATE|os.O_WRONLY|os.O_TRUNC`.
- **#545** `internal/generator/generator.go` — `replaceOutputDirectory` double
  failure: when both the live swap and the restore-from-backup fail, the
  previous site is stranded in the backup dir but the error never names it.
  Extract `restoreFailedError` so the backup path is part of the returned error.
- **#546** `cmd/la-famille/main.go` — `logger.Setup` errors were discarded in
  both call sites. Log a `slog.Warn` on failure and retain/close the returned
  `*os.File` via `closeLogFile()`.
- **#549** `internal/generator/generator.go` — `convertMu` was a `sync.RWMutex`
  but the read path (getConvertMarkdown) always took a full lock anyway; use a
  plain `sync.Mutex`.
- **#544** `format_check.sh` — the check only ran `go fmt`. Add `go vet`,
  `go mod tidy -diff`, and optional `golangci-lint run`.

## Tests added (same-package)
- `internal/asset/copy_test.go`: `TestCopyFileCreatesFreshDestination` —
  fresh-file creation (the O_CREATE|O_WRONLY path) and O_TRUNC overwrite.
- `internal/generator/build_audit_test.go`: `TestRestoreFailedErrorNamesBackupDir` —
  the composed error names the backup dir, keeps both causes, and preserves
  `errors.Is` on the primary cause. (The exact double rename is not reachable
  through a real filesystem — the move-aside must succeed first — so the new
  helper that owns the message is pinned directly.)

## Incidental fix (required by #544)
`internal/pathutil/pathutil.go` and `_test.go` were missing a trailing newline
on master (#541) and failed `gofmt -l`; added the EOF newline so the new format
check passes.

## Static asset pipeline impact
None. No `assets/`, `templates/`, or generated-output behavior changes.

## Verification
- `go test ./...` — all pass
- `go vet ./...` — clean
- `gofmt -l cmd internal` — clean
- `bash format_check.sh` — passes once changes are committed (it compares the
  staged-vs-unstaged tree; local uncommitted edits legitimately show as a diff)

## Handoff
Changes are uncommitted on `fix/wolf-review-tier1` (off `origin/master`). Roll
the six files into one commit, open a PR from the branch, and reference
#543 #545 #546 #549 #544 (+ #500 backlog).