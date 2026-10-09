# Task Plan: Issue #654 — Same-title remove+add mis-paired as rename

## Task ID
`issue-654` (GitHub Issue #654)
Branch: `t3/fix-graph-edges` (from `origin/master` at `eb834ca`)

## Objective
`matchPages` pairs leftover removed/added pages on a unique shared title
alone. Removing `x.md` and adding an unrelated `y.md` titled the same reports
`~ x → y (renamed)` with 0 added / 0 removed, hiding a full page replacement.

## Context
- `internal/diff/diff.go:352-387`: leftover pairing is title-only.
- Manifest v2 pages carry `ContentHash` (sha256 of the Markdown body). An
  unchanged body under a new identity is the corroborating signal a title
  cannot provide; a different body means the pair is a removal plus an
  addition, not a rename.
- Gate: skip title pairing when *both* manifests carry a `ContentHash` and
  they differ. When either hash is missing (v1 snapshots, partial data) the
  title remains the only available signal and keeps its legacy meaning —
  `testdata/rename-{before,after}.json` has no hashes and still pairs.
- Consequence: a genuine rename whose body was also edited reports as
  remove+add. At manifest level that is honest — nothing distinguishes
  "moved and rewrote" from "deleted + coincidentally same title".

## Scope
- `internal/diff/diff.go`: veto unique-title pairing on hash mismatch.
- New test in `internal/diff/diff_test.go`: same-title pages with different
  `ContentHash` → 1 removed + 1 added, no rename; same hash → still a rename.

## Static-Output Impact
None — `la-famille diff` output only. Reports become strictly more accurate
(v2 snapshots); v1 snapshot behavior unchanged.
