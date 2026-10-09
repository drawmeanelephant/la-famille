# Issue #644 — unreadable matched files silently omitted from bundles

## Task ID
`issue-644` (branch `t3/fix-rag-hardening`)

## Bug
`writeBundle` (`internal/ragexport/export.go`) does
`content, err := os.ReadFile(path); if err != nil { continue }`. A matched
file that fails to read (e.g. mode 000) is dropped from the bundle with no
warning; the export exits 0 and logs success over an incomplete archive.

## Fix
Return the read failure from `writeBundle`, naming the project-relative
path: `read <rel>: %w`. `RunExport` already wraps per-bundle errors
("failed to write system/content bundle"), so the command exits non-zero
with a message that identifies the skipped file. Combined with the #645
staging change, a failed run also leaves the previous archive untouched.

Chosen behavior is fail-the-run rather than warn-and-skip: the bundle then
always matches the matched file set exactly, which the acceptance criteria
requires; a warning would ship a silently incomplete archive.

## Tests (written first, failing on master)
- `TestRunExport_UnreadableMatchedFile` — chmod 000 on a matched `.go`
  source; RunExport must return an error naming the file (skipped when
  running as root, matching the `os.Getuid() == 0` convention used by
  `internal/config/config_test.go`).

## Breaking-change note
`rag` now exits non-zero where it previously succeeded with an incomplete
archive. The affected input was already broken; nothing else changes.
