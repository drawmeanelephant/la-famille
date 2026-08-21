# Task Plan: 471 Build Cache Invalidation Regression

## Task ID
`471-build-cache-invalidation` (GitHub Issue #471) — Milestone 3. Build Correctness and Performance
Closes: https://github.com/drawmeanelephant/la-famille/issues/471
Branch: `t3code/471-cache-invalidation` (from `master` at `87fe85b`)
Source: `content/meta/roadmap.md:35`

## Objective
Make the transactional build + incremental cache trustworthy as the default production path by proving every invalidation edge with table-driven regressions (`internal/generator/generator.go:104` `Build` + `internal/generator/cache.go:87` `cacheFingerprint`).

## Context
- Cache key: `internal/generator/cache.go:87` hashes `generatorIdentity()` + `cfg` (minus `WatchMode`) + `hashTree` of `ContentDir`, `Template` dir, `AssetDir`, and `.gitignore`. `cache.go:217` `cacheUsable` additionally content-hashes every file in `OutputDir` via `generatedFiles`.
- Existing coverage is strong but issue remains open:
  - `internal/generator/cache_invalidation_test.go` (not yet on master — only `cache_integrity_test.go:11` + `generator_test.go:181` exist). On master the matrix is missing; `cache_integrity_test.go` covers tamper+identity, `generator_test.go:181` covers WatchMode ignore + `.gitignore` + missing output.
  - Need full matrix per issue: modify content -> rebuild, modify template -> all pages re-render, add/remove asset -> public/assets reflects, modify config.yaml (site_name/siteurl/theme/graph_explorer) -> re-render, delete .md -> output removed + graph/search/taxonomy updated.
- Gaps: `config.yaml` toggle `graph_explorer` + `siteurl` + full delete chain (output removed, `graph/search/taxonomy` updated) not explicitly in existing tests on master. Benchmark branch already has some but not merged.

## Scope
In scope:
- `internal/generator/cache_invalidation_test.go` (new file) — table-driven matrix with `setupTestSite(t)` helper (`t.TempDir()` isolated):
  - `1_UnchangedMarkdown_ProducesCacheHit` — two Builds without touch => second `CacheHit==true`
  - `2_ChangedMarkdown_TriggersRebuild` — modify `page1.md` body => miss + `public/page1/index.html` contains new text
  - `3_DeletedMarkdown_RemovesGeneratedPage` — delete `page2.md` => miss + `public/page2/index.html` gone + `search.json` no longer contains deleted page
  - `4_ChangedTemplates_TriggersRebuild` — modify `layout.html` => miss + output reflects new markup
  - `5_ChangedAssets_TriggersExpectedOutputUpdate` — modify/add/delete asset under `AssetDir` => miss + `public/assets/...` updated/added/removed
  - `6_ChangedConfiguration_TriggersRebuild` — mutate `cfg.SiteName`, `cfg.Theme`, plus `cfg.SiteURL` and `cfg.GraphExplorer` toggle; each => miss. For `SiteURL` verify `search.json`/`sitemap.xml` URLs reflect new base; for `graph_explorer` verify `public/graph/index.html` appear/disappear (ties to #473 cleanup but regression here).
  - `7_RemovedGeneratedArtifacts_DoNotSurviveLaterBuild` — remove `search.json` or inject orphan `stale_artifact.txt` into output => miss + file restored/cleaned
  - Use deterministic temp dirs, do not touch real `content/`/`public/`. Reuse `setupTestSite` pattern from previous draft (contentDir/templateDir/assetDir/outputDir + `config.DefaultConfig()` with `ProjectRoot=tempDir`).
- `internal/generator/cache.go` — no change expected; verify `json.Marshal(cfg)` includes `GraphExplorer`/`SiteURL` (it does).
- Keep `internal/generator/generator.go` unchanged — matrix is verification only.

Out of scope:
- Changing cache algorithm or `cacheVersion` (`cache.go:24`).
- Watcher/concurrent audit (deferred to `#473`).
- Docs bench (deferred to `#472`).

## Static-Output Impact
None beyond test fixtures. No `public/` artifact change. No breaking change.

## Ownership
Agent parallel-safe: test-only file `cache_invalidation_test.go`. No lock contention with `472` (bench) — bench is separate file. Coordinate with `473-build-audit` on `graph_explorer` toggle assertions — share expectation, avoid duplicating fix.

## Tests & Verification
- `go test ./internal/generator -run TestCacheInvalidationMatrix -count=1 -race` must pass + retain existing `TestBuild_ModifiedOutputInvalidatesCache`, `TestBuild_UsesAndInvalidatesCache`.
- `go test ./...` + `go vet ./...` + `go test -race ./...` (issue verification spec).
- Manual: `go test -v ./internal/generator -run TestCacheInvalidationMatrix` shows 7 sub-tests.

## Potential Breaking Changes
None. Test-only.

## Steps
1. Read `internal/generator/cache.go:87`, `generator.go:104`, `config/config.go:Validate`.
2. Create `cache_invalidation_test.go` with `setupTestSite` + table-driven matrix.
3. Run `go test -count=1 ./internal/generator` + full `go test ./...` + `go vet ./...`.
4. Update this plan status, open PR.

## Status
- [x] Plan created
- [x] Branch cut
- [x] Matrix created (7 sub-tests, siteurl + graph_explorer toggle)
- [x] Tests pass `go test ./...` `go vet ./...` `go test -race ./...`
- [x] PR ready
