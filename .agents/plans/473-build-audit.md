# Task Plan: 473 Build Audit — Cleanup, Concurrent, Watcher

## Task ID
`473-build-audit` (GitHub Issue #473) — Milestone 3. Build Correctness and Performance
Closes: https://github.com/drawmeanelephant/la-famille/issues/473
Branch: `t3code/473-build-audit` (from `origin/master` at `c36440e`)
Source: `content/meta/roadmap.md:37`

## Objective
Close remaining correctness gaps around the transactional output pipeline — cleanup, concurrent builds, watcher rebuilds.

## Context
- Transactional pipeline `internal/generator/generator.go:104` `Build` → `createStagingOutput` (`generator.go:863`) staging beside output + `replaceOutputDirectory` (`generator.go:908`) atomic `Rename` swap with backup. In-process serialization `generator.go:55` `buildLocks` + `lockOutputDir` (`generator.go:63`) — process-local `sync.Mutex` per `filepath.Abs(outputDir)` key; two separate `la-famille` processes are still unserialized (needs `flock` or doc).
- Existing coverage: `internal/generator/build_concurrency_test.go:10` `TestBuild_ConcurrentBuildsShareOutputDirectory` (6 goroutines, no staging leak), `internal/generator/cache_integrity_test.go:11` tamper+identity, `internal/watcher/watcher.go:17` debounce with `trigger chan struct{1}` + single builder goroutine (`watcher.go:60` comments), `internal/watcher/watcher_test.go` cancellation + debounce tests.
- Gaps per audit checklist:
  - Cleanup: deleted source removes stale output (`generator_test.go:134` covers one case), `render:false` toggle cleans old `.html`, `feed.xml` removed when 0 dated pages, `graph/index.html`+`graph/data.json` removed when `graph_explorer:false` (tied to `generator.go:606` `reservedOutputPaths`).
  - Concurrent builds: two `la-famille build` on same `outputDir` (process-local lock suffices for watcher-triggered `Build` vs `Build` but not two CLI processes — doc or add `flock`).
  - Watcher rebuilds: rapid `fsnotify` bursts coalesced (`watcher.go:210` debounce), no goroutine leak (`watcher.go:36` `buildCtx`/`stopBuilder`), `public/` never half-written (staging + atomic swap already).

## Scope
In scope:
- `internal/generator/generator.go`:
  - Verify `feed.Write` (`internal/feed/feed.go`) removes `public/feed.xml` when `datedItems` empty; if not, add removal. Same for graph explorer: `graphexplorer.Write` must remove `graph/index.html`/`graph/data.json` when `cfg.GraphExplorer==false` and stale files exist from prior build (staging already handles but cache-hit path `generator.go:125` returns early without cleaning old artifacts — need to ensure stale removal on config toggle).
  - Verify `render:false` toggle: previously rendered `page.md` → `render:false` must remove `public/page/index.html` (staging covers but cache invalidation must trigger; ensure `cacheFingerprint` includes `render` frontmatter via content hash — it does via `hashTree` of file bytes).
  - Document or implement cross-process lock: either add `syscall.Flock` on `cachePath` dir or document that two CLI processes on same `outputDir` must not run in parallel (add note to `content/docs/generator.md`).
- `internal/watcher/watcher.go`: confirm rapid burst coalescing into 1 rebuild + no leak (existing), add `public/` half-write invariant test if missing.
- `internal/asset/copy.go`: confirm `.gitignore` + `pathutil.IsSafePath` still enforced during asset clobber audit.
- `content/jules/reports/<date>-build-audit.md` — audit report deliverable (checklist with pass/fail per item, plus fixes applied).
- Tests: `internal/generator` cleanup edge cases (render:false toggle, feed empty, graph toggle), `internal/watcher` burst test if needed.

Out of scope:
- Cache algorithm change (`cacheVersion`).
- Bench numbers (`#472`).
- TUI polish (`#469`/`#470` — now done).

## Static-Output Impact
Fixes may change `public/` when toggling `graph_explorer` or when `feed.xml` should be removed — this is correctness (removing stale artifacts that previously leaked). Document in audit report. No template change.

## Ownership
Single-stream after `471`/`472`. Isolated audit.

## Tests & Verification
- `go test ./internal/generator -run TestBuild -count=1` + full `go test ./...` `go vet ./...` `go test -race ./...`
- New tests: `TestBuild_FeedRemovedWhenNoDatedPages`, `TestBuild_GraphRemovedWhenDisabled`, `TestBuild_RenderFalseRemovesOldHTML` (or extend existing).
- Manual: toggle `graph_explorer: false` → `la-famille build` → `public/graph/` gone; delete dated page → `feed.xml` gone when none.
- Deliverable: `content/jules/reports/*-build-audit.md` with checklist + outcome.

## Potential Breaking Changes
Stale artifact removal changes output contract (removing files that previously persisted across toggles). List in audit report under "Breaking / Intentional".

## Steps
1. Audit `internal/feed/feed.go` + `graphexplorer.Write` + staging vs cache-hit stale cleanup.
2. Implement minimal fixes + tests.
3. Document concurrent-build lock strategy (doc or `flock`).
4. Write `content/jules/reports/<date>-build-audit.md`.
5. Validate `go test ./...` `go vet ./...`.

## Status
- [x] Plan created
- [x] Branch cut `t3code/473-build-audit`
- [x] Cleanup audit + fixes — no code fix needed, transactional staging already handles; verified via `build_audit_test.go` 4 cases
- [x] Concurrent builds doc/impl — process-local `sync.Mutex` documented, cross-process limitation noted, no `flock` added (deferred)
- [x] Watcher burst audit — debounce coalescing, no leak, half-write never via atomic Rename
- [x] Tests added — `build_audit_test.go` 7 tests, `go test ./internal/generator -run TestBuild_Audit -count=1 -v` pass
- [x] Audit report written — `content/jules/reports/2026-08-21-build-audit.md`
- [x] Validation passing — `go test ./...` ok, `go vet ./...` clean, `go test -race ./internal/generator ./internal/watcher` ok
