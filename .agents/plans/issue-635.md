# Plan: #635 — symlinked content_dir publishes an empty site

## Bug
`content.GatherMetadata` walks with `filepath.WalkDir`, which lstats its root:
a symlinked `content_dir` is reported as a symlink on the first callback, hits
the "Skipping symlink in content" branch, and yields an empty file map. The
empty result is also cached, so later edits do not fix it. `build` exits 0
with an empty `public/`.

## Fix
`internal/content/metadata.go`: resolve the walk root once up front (Lstat +
EvalSymlinks, mirroring `generator.walkRootFor` and `asset.resolveDir`).
A dangling/unresolvable content_dir symlink returns a clear error instead of
silently publishing nothing. Symlinks *inside* the tree keep the existing
skip-and-warn behavior. `relPath` keys are computed against the resolved root
so map keys are unchanged.

Scope note: another agent is fixing symlink-following writes in cache tmp
files; this change is deliberately limited to metadata gathering/discovery.

## Tests
- `internal/content`: GatherMetadata on a symlinked content dir returns the
  same rel-path keys and metadata as the real directory; an inner symlinked
  file is still skipped.

## Breaking changes
- None. Sites that silently built empty output now build the real pages.
