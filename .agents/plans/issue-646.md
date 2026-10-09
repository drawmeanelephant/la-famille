# Issue #646 — writes follow pre-existing symlinks in persistent dirs [SECURITY]

## Task ID
`issue-646` (branch `t3/fix-rag-hardening`)

## Bug
A cloned repo can plant gitignore-able symlinks so `rag`/`build` clobber
files outside the project:

- `internal/ragexport/export.go` — `os.Create` / `O_APPEND` on
  `rag-archive/rag-*.md` writes through a symlinked destination.
- `internal/generator/cache.go` — `os.WriteFile(path + ".tmp")` writes
  through a symlinked `.la-famille-cache.json.tmp`.

## Fix

### RAG bundles
With the #645 staging change, bundle bytes are written inside a fresh
`MkdirTemp` staging dir — a planted destination symlink is never opened.
To keep the attack visible (the issue asks for a clear failure, not silent
success), `RunExport` verifies destinations before staging:

- `RagDir`, when it exists, must be a real directory (`os.Lstat`, rejects
  symlinks — same rule as `createStagingOutput` in generator.go).
- Each bundle path (`rag-system.md`, `rag-config.md`, `rag-content.md`),
  when it exists, must be a regular file. Any symlink/special file →
  error, so the command fails with a clear message and the victim file
  is untouched. `os.Rename` during the swap never follows a leaf link, so
  there is no check-to-act write-through race.

### Build cache tmp file
Write the temp side via `os.Root`: `root.OpenFile(".la-famille-cache.json.tmp",
O_WRONLY|O_CREATE|O_TRUNC, 0600)`. Root refuses a symlinked leaf (verified:
`openat link: path escapes from parent`), so a planted `.tmp` symlink makes
`writeBuildCache` — and the build — fail with a clear error. A stale regular
`.tmp` still truncates as before; the final `os.Rename(tmp, path)` already
replaces (never follows) a symlinked cache path itself.

No new dependencies: `os.Root` is stdlib and already used by
`internal/pack` (`feed.go`, `watch.go`).

## Scope note
`jsonutil.WriteJSON` is explicitly out of scope per the issue (operator-gated
via `diff --report-dir`). `runtimeassets.InstallMissing` and `pack/build.go`
already use O_EXCL.

## Tests (written first, failing on master)
- `TestRunExport_RefusesSymlinkedDestination` — planted
  `rag-archive/rag-system.md -> victim`; RunExport errors, victim bytes
  unchanged.
- `TestWriteBuildCache_RefusesSymlinkedTmp` — planted
  `.la-famille-cache.json.tmp -> victim`; writeBuildCache errors, victim
  unchanged.

## Breaking-change note
A RagDir/bundle path or cache `.tmp` that is a symlink now fails instead of
being written through. No legitimate layout is affected; pipelines that
relied on writing through a symlinked archive path must replace the link
with a real directory.
