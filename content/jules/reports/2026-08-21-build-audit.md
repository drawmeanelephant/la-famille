---
date: "2026-08-21"
title: "Build Correctness and Performance Audit — Transactional Output & Watcher"
author: "OpenCode"
layout: "report"
---

# Build Audit — Output Cleanup, Concurrent Builds, Watcher Rebuilds

**Milestone:** 3. Build Correctness and Performance — `roadmap.md:37` / Issue #473  
**Source:** `content/meta/roadmap.md:37` — "Audit generated-output cleanup, concurrent builds, and watcher-triggered rebuilds."  
**Scope:** `internal/generator/generator.go:104` `Build` (staging + `replaceOutputDirectory` + `lockOutputDir`), `internal/generator/cache.go:87` `cacheFingerprint`, `internal/watcher/watcher.go:17` debounce, `internal/asset/copy.go`, `internal/feed/write.go`, `internal/graphexplorer/graphexplorer.go`

## Summary

The transactional output pipeline is correct. No code fix was required beyond verification via new regressions. Staging beside `outputDir` plus atomic `Rename` guarantees `public/` is never half-written, even under watcher bursts or concurrent `Build` calls in one process. Orphan and stale-asset cleanup, `render:false` toggles, `feed.xml` removal, and `graph_explorer` toggles are all handled by the same mechanism: a fresh staging tree that omits the artifact, then swaps into place.

## Audit Checklist

### Cleanup

| Item | Expected | Result | Evidence |
|------|----------|--------|----------|
| Deleted source removes stale output + taxonomy/search | Staging omits file, swap deletes old tree | **PASS** | `generator_test.go:134` `TestBuildRemovesOutputForDeletedSource` + new `build_audit_test.go:84` `deleted_source_removes_stale_output` (tags audit removed, `search.json` purged) |
| `render:false` toggles clean up old `.html` | Previously rendered `foo/index.html` must vanish, raw `foo.md` verbatim remains | **PASS** | `build_audit_test.go:12` `render_false_toggle_removes_old_html` — verifies `public/page1/index.html` gone, `public/page1.md` present, `search.json` excludes page |
| `feed.xml` removed when zero dated pages | `internal/feed/write.go:58` `len(items)==0` removes existing file; staging swap removes stale copy | **PASS** | `feed/write_test.go:45` + `build_audit_test.go:38` `feed_removed_when_no_dated_pages` — dated → feed exists, undated → feed gone |
| `graph/index.html` + `graph/data.json` removed when `graph_explorer:false` | `graphexplorer.Write:98` disabled returns without writing; config change invalidates fingerprint so cache miss goes via staging | **PASS** | `generator_test.go:1050` `TestBuild_GraphExplorerDisabledSkipsPage` + `build_audit_test.go:55` `graph_explorer_toggle_cleans_output` (toggle true→false→true) |
| Asset add/remove reflects change, stale removed | `hashTree` of `AssetDir` invalidates cache; staging copies new set, swap deletes old asset | **PASS** | `cache_invalidation_test.go:186` `ChangedAssets` + `build_audit_test.go` asset safety check |

*Note on `feed.xml` in staging:* `internal/feed/write.go:60` `os.Remove(feedPath)` in staging is a no-op when staging is empty; the real stale removal is the swap that replaces the old `public/` tree. The `Remove` path matters only for incremental callers that build directly into `outputDir` without staging (legacy). Verified that the staging path alone would also delete stale `feed.xml` via swap, even without the explicit removal.

### Concurrent Builds

| Question | Answer |
|----------|--------|
| Two `la-famille build` in parallel on same `outputDir` — does staging dir + atomic `Rename` avoid interleaving? | **PASS for single process, documented limitation for cross-process.** `internal/generator/generator.go:55` `buildLocks` + `lockOutputDir:63` is a `sync.Mutex` per `filepath.Abs(outputDir)` — it serializes concurrent `Build` calls in one process (the watcher debounce can overlap a build, hence `build_concurrency_test.go:10` `TestBuild_ConcurrentBuildsShareOutputDirectory` with 6 goroutines). Two separate `la-famille` processes on the same `outputDir` are **not** serialized — `flock` is not used. This is intentional per comment `generator.go:55` "The lock is process-local". Adding cross-process `flock` would require `syscall.Flock` and platform handling and is out of scope for this audit. Documented below. No intermediate dirs leaked (`build_concurrency_test.go:46` asserts no `.staging-` / `.previous-`). |

**Recommendation:** Do not run two `la-famille build` CLI processes concurrently on the same `outputDir`. Within one process (including watcher-triggered rebuilds) the pipeline is safe. If cross-process parallelism is ever required, add a `flock` on `cachePath` directory — deferred, not implemented here.

### Watcher Rebuilds

| Item | Result | Evidence |
|------|--------|----------|
| Rapid `fsnotify` bursts coalesced into one rebuild | **PASS** | `watcher.go:52` `trigger chan struct{1}` + single builder goroutine `watcher.go:90` (one-slot collapse) + debounce `watcher.go:193` `time.AfterFunc(debounce)` non-blocking send. `watcher_test.go:61` `TestWatchDebouncesAndTracksNewDirectories` asserts burst of 4 writes produces `1 ≤ builds < writes`; `TestWatchCoalescesChangesDuringABuild` pins exact collapse via gated build. |
| No goroutine leak | **PASS** | `watcher.go:60` `buildCtx`/`stopBuilder` + `defer <-buildDone` ensures builder exits before `Watch` returns, even on `watcher.Add` failure or `ctx.Done`. `watcher_test.go:43` `TestWatchCancellation` verifies `context.Canceled` and no leak. |
| `public/` never half-written | **PASS** | Generator's `createStagingOutput` + `replaceOutputDirectory` (`generator.go:863`/`908`) stages beside output and swaps via `os.Rename` (atomic on POSIX). Watcher's builder is the only writer, single goroutine, so two builds never overlap output swap. `build_audit_test.go:95` `TestBuild_AuditWatcherContract` verifies no `.staging-`/`.previous-` left and all expected files present after rapid builds. |

**Additional:** `watcher.go:72` `runBuild` deliberately does **not** call `BroadcastReload` on `build(cfg)` error — failed build leaves previous `public/` intact and does not ask browser to reload unchanged bytes (`watcher.go:76` comment). Verified in `watcher_test.go` helpers.

## Tests Added

- `internal/generator/build_audit_test.go` — `TestBuild_AuditCleanup` (4 sub-tests), `TestBuild_AuditWatcherContract`, `TestBuild_AuditConcurrentIsProcessLocal`, `TestBuild_AuditAssetPathSafetyStillEnforced`. All use `t.TempDir()` / `setupTestSite(t)` hermetic.
- Existing tests retained: `cache_invalidation_test.go:126` matrix (7 cases, `SiteURL` + `graph_explorer` toggle), `cache_integrity_test.go`, `build_concurrency_test.go`, `watcher_test.go` (5 tests).

## Verification

```bash
go test ./internal/generator -run TestBuild_Audit -count=1 -v
go test ./internal/generator -run TestCacheInvalidationMatrix -count=1 -race
go test ./internal/watcher -count=1 -race
go test ./... -count=1
go vet ./...
go test -race ./...
```

All pass locally (`generator 0.26s audit, 3.4s race`).

## Static-Output Impact

Stale artifact removal (when toggling `graph_explorer` or removing all dates) changes output contract intentionally — previously leaked `graph/` or `feed.xml` could survive toggles if cache were hit, but fingerprint invalidation plus transactional swap now guarantees clean output. Document as **intentional correctness** — not a breaking change for sites that relied on stale files.

## Deliverable

This report (`content/jules/reports/2026-08-21-build-audit.md`) + `internal/generator/build_audit_test.go` + no generator code change required. Concurrent-build limitation documented in this report and in `internal/generator/generator.go:51` comment; no `flock` added.

## References

- `internal/generator/generator.go:51` `buildLocks` + `internal/generator/generator.go:908` `replaceOutputDirectory`
- `internal/generator/cache.go:87` `cacheFingerprint`, `internal/generator/cache.go:217` `cacheUsable`
- `internal/watcher/watcher.go:52` trigger channel + `internal/watcher/watcher.go:193` debounce
- `internal/feed/write.go:58` empty-feed removal, `internal/graphexplorer/graphexplorer.go:98` disabled path
- `internal/asset/copy.go:95` `pathutil.IsSafePath`

## Status

- [x] Cleanup audited (4 cases)
- [x] Concurrent builds audited (process-local lock, cross-process documented)
- [x] Watcher rebuilds audited (burst, leak, half-write)
- [x] Tests added, `go test ./...` / `go vet` / `go test -race` pass
- [x] Report written
