# Issue #655 — pack watch never self-heals after subscriber pack deletion/corruption

## Problem
`pollWatch` (`internal/pack/watch.go`): once `current.json` names a pack whose
file is missing or fails verification, every poll errors on the unchanged fast
path (`verifiedStatePack`) or the base-capture path (`captureStatePack`). The
watch loops forever logging `Pack poll failed; current version preserved` while
the packs dir is empty — the "preserved" claim is false and no recovery is
attempted.

## Fix
- Treat a state pack that fails verification as unusable: remove the
  content-addressed entry (bytes that do not match `packs/<sha256>.tar` can
  never become valid again), recreate `packs/` if it is gone, and fall through
  to the normal pull path with no delta base. The feed's advertised full pack
  is reinstalled and the pointer re-published; the poll reports a real pull
  event instead of an endless error.
- Covers both the unchanged fast path (same SHA) and the changed path (missing
  base degrades to a full pull).
- Add `WatchEvent.Preserved`, set only when this poll actually verified the
  recorded current pack (successful `captureStatePack`). The CLI report prints
  `Pack poll failed; current version preserved` only when `Preserved` is set;
  otherwise it prints `Pack poll failed: <err>` so failures with no usable
  installed pack no longer claim preservation.

## Behavior changes / breaking notes
- `pack watch` output: failed polls without a verified intact current pack now
  print `Pack poll failed: ...` instead of `...; current version preserved`.
- Existing test `TestWatchRejectsBadStateAndConcurrentWriter` case
  "corrupt current" now self-heals when the feed is intact; the case is kept
  but the feed target is removed so it still asserts a loud failure when
  healing is impossible.
- No change to the static asset generation pipeline; subscriber durable-state
  layout (`current.json` + `packs/<sha>.tar`) unchanged.

## Tests
- New `TestWatchHealsDamagedInstalledPack`: after a successful install, delete
  or corrupt the installed pack (and temporarily break the feed to assert
  `Preserved` stays false and the poll error is reported); next poll must
  reinstall the pack byte-identically and succeed.
- Extend `TestWatchFailedSelectedDeltaPreservesCurrentAndRecovers` to assert
  `event.Preserved` on the preserved-failure path.

## Validation
`go test ./...`, `go vet ./...`.
