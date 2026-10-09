# Issue #643 — `rag_dir: "."` produces 0-byte bundles with exit 0

## Task ID
`issue-643` (branch `t3/fix-rag-hardening`)

## Bug
With `rag_dir: "."` the RAG archive dir resolves to the project root.
`isWithinDir` (`internal/ragexport/export.go`) then treats every walked file
as inside the output directory and excludes it: `rag-system.md`,
`rag-config.md`, and `rag-content.md` are written 0 bytes while "Created ..."
is logged and the command exits 0. The same empty-archive result occurs for
any RagDir that contains the project root (e.g. `--output ..`), not only the
exact-root case.

## Fix
Two layers:

1. `internal/config/config.go` `validateOutputIsolation`: reject a `RagDir`
   that IS the project root or that contains it (`rag_dir: "."` reaches this
   via `Validate`/`ValidateResolved`, which `la-famille rag` already calls).
   Message states that the archive would exclude every source file.
2. `internal/ragexport/export.go` `RunExport`: guard before any writes —
   `isWithinDir(ProjectRoot, outDir)` (project root inside or equal to the
   output dir) returns a clear error instead of emitting empty bundles.
   This covers library callers (TUI, pack) that bypass config validation.

Rejected alternative: "include the project files anyway". The walk's
self-exclusion is what keeps stale archives out of bundles; a RagDir that
contains the root cannot distinguish archive output from sources, and the
upcoming atomic-swap fix (#645) would rename the entire project root aside.
Rejection is the honest outcome and one of the issue's accepted greens.

## Tests (written first, failing on master)
- `TestRunExport_RagDirIsProjectRoot` — RunExport errors, no rag-*.md files
  left behind; also covers RagDir = parent of project root.
- `TestValidateRejectsRagDirContainingProjectRoot` — `rag_dir: "."` and a
  RagDir parent both fail validation with a clear message.

## Breaking-change note
Configs with `rag_dir` equal to (or containing) `project_root` now fail
validation for every command, not just `rag`. Such a config always produced
empty archives, so this surfaces a latent misconfiguration rather than
breaking a working layout.
