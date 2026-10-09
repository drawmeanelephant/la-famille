# Issue #645 — failed export leaves a mixed-generation archive

## Task ID
`issue-645` (branch `t3/fix-rag-hardening`)

## Bug
`RunExport` writes the three bundles sequentially into the persistent
`rag-archive/` directory. When a later stage fails (e.g.
`unpublishedContentExcludes` hits a permission-denied content file), the
directory keeps fresh `rag-system.md`/`rag-config.md` beside a stale
`rag-content.md` — an archive that looks complete but mixes generations.

## Fix
Mirror the build's existing convention (`createStagingOutput` /
`replaceOutputDirectory` in `internal/generator/generator.go`):

1. Create a staging sibling `.<base>.staging-*` via `os.MkdirTemp` in the
   RagDir parent (same filesystem → atomic rename possible).
2. Write all three bundles into staging only. Walks exclude BOTH the real
   RagDir and the staging dir, since staging inside the project would
   otherwise leak into the bundles it is producing.
3. On success, swap: `os.Rename(outDir -> .previous-*)`,
   `os.Rename(staging -> outDir)`, remove backup, restoring on failure —
   the same dance as `replaceOutputDirectory`. On any earlier failure the
   deferred `RemoveAll(staging)` cleans up and the prior archive is intact.

A `defer os.RemoveAll(staging)` guarantees no `*.staging-*` litter.
"Created rag-*.md" logs move after the swap so they only report committed
output (also required by #643's "no empty bundle reported as Created").

## Tests (written first, failing on master)
- `TestRunExport_FailureLeavesArchiveIntact` — successful export, then a
  chmod-000 source makes the next export fail; all three bundle files must
  be byte-identical to the first generation and no staging dirs remain.

## Breaking-change note
The archive directory is now installed by rename, so the previous archive
dir is replaced wholesale on success and left untouched on failure. No
pipeline-visible behavior changes on the happy path.
