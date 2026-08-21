# Task Plan: 472 Build Cold vs Incremental Bench

## Task ID
`472-build-bench` (GitHub Issue #472) — Milestone 3. Build Correctness and Performance
Closes: https://github.com/drawmeanelephant/la-famille/issues/472
Branch: `t3code/472-build-bench` (from `origin/master` at `3e89b46`)
Source: `content/meta/roadmap.md:36`

## Objective
Quantify the payoff of the incremental cache with reproducible cold vs warm numbers — not anecdotes.

## Context
- Existing bench `internal/generator/generator_test.go:501` `BenchmarkBuild` builds 1000 dummy files in `b.TempDir()` and loops `Build(cfg)` with `b.ResetTimer()`. It measures hot cache but not cold vs single-file-touch, and always caches after first iteration.
- Cache hit path `internal/generator/generator.go:125` `loadBuildCache` + `cacheUsable` short-circuits staging + re-render; fingerprint `internal/generator/cache.go:87` covers content/templates/assets/`.gitignore` + `config` + `generatorIdentity`.
- Issue wants 2 fixtures: small (current `content/` ~20 pages) + large synthetic 200–500 pages in `t.TempDir()` or `assets/testdata/sites/`.

## Scope
In scope:
- `internal/generator/bench_test.go` (new) — add:
  - Helper `syntheticSite(b, n)` generating `n` markdown files with links/tags in `b.TempDir()` (use `setupTestSite` pattern but parameterized).
  - `BenchmarkBuild_Cold_25` / `BenchmarkBuild_WarmHit_25` / `BenchmarkBuild_WarmSingleTouch_25`
  - `BenchmarkBuild_Cold_300` / `BenchmarkBuild_WarmHit_300` / `BenchmarkBuild_WarmSingleTouch_300`
  - Cold: remove cache file beside `ProjectRoot`, run `Build` (miss). WarmHit: no touch (hit). WarmSingleTouch: touch single file (miss but minimal). Report `ns/op`.
  - Keep CI time <5s per bench (use benchtime=1x for 300 pages or smaller n).
  - All hermetic via `b.TempDir()`, never touch real `content/`/`public/`.
- `internal/generator/cache.go:87` — no algorithm change (out of scope).
- Docs: after numbers stable, append results table to `content/docs/generator.md` or `content/meta/roadmap.md` shipped section. Keep out of `README.md` marketing per issue. Defer if bench flaky.

Out of scope:
- Changing cache algorithm, `cacheVersion`, or `cacheFingerprint`.
- Watcher/concurrent audit (`#473`).

## Static-Output Impact
None. Bench-only + docs. No `public/` artifact change. No breaking change.

## Ownership
Single-stream after `471`. Isolated bench file. No contention.

## Tests & Verification
- `go test -bench=BenchmarkBuild -benchtime=1x -count=1 ./internal/generator` reports cold vs warm lines.
- `go test ./...` still passes (bench not counted).
- `go vet ./...`.
- Manual: `go test -bench=. ./internal/generator -benchtime=1s` — warm hit is 10–50x faster than cold on 300-page fixture.

## Potential Breaking Changes
None. Measurement only.

## Steps
1. Create `internal/generator/bench_test.go` with `syntheticSite` + cold/warm benchmarks.
2. Run `go test -bench=. -benchtime=1x ./internal/generator` to sanity-check.
3. Validate `go test ./...` `go vet ./...`.

## Status
- [x] Plan created
- [x] Branch cut
- [x] `bench_test.go` added (small + large fixtures, cold vs warm)
- [x] Bench runs locally — Cold25 46ms vs WarmHit25 12ms (3.8x), Cold300 173ms vs WarmHit300 46ms (3.7x), `go test -bench` ok
- [ ] Docs updated (optional post-stability — deferred per issue, bench file is measurement deliverable)
- [x] Validation passing — `go test ./...` ok, `go vet ./...` clean, `go test -race ./internal/generator` ok, `TestBuild_BenchColdVsWarm` 25+100 pass 3.57x/2.03x ratio
