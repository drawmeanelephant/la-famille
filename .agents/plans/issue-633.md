# Plan: #633 — serve --watch infinite rebuild loop on flat layouts

## Bug
With `content_dir: "."`, `la-famille serve --watch` rebuilds forever after one
edit. Three generator artifacts live inside the watched/fingerprinted project
root:

- `.la-famille-cache.json` (and its `.tmp` sibling) — hashed by the fingerprint,
  so the fingerprint hashes the cache it just wrote and can never hit.
- `.<output>.staging-*` / `.<output>.previous-*` — the atomic swap's transient
  siblings. The watcher's create handler walks and watches them, so every file
  the build stages fires a change event. Watches survive the rename into
  `public/`, so post-swap writes like `public/meta.json` keep firing.
- `public/` itself — the swap's rename events fire on the watched parent.

## Fix
1. `internal/generator/cache.go`: exclude the cache file (+`.tmp`), the output
   directory, and `.<base>.staging-*` / `.<base>.previous-*` siblings from
   `hashTree` via a predicate built once per fingerprint. Export
   `CachePath(cfg)` and `BuildArtifactFilter(cfg)` so the watcher shares the
   exact same ignore rules.
2. `internal/watcher/watcher.go`: build the ignore predicate once; use it in
   the initial WalkDir (SkipDir, no watcher.Add), in the dynamic directory
   tracking, and to drop events before scheduling a build.

## Tests
- `internal/watcher`: flat-layout watch — writes to the cache file and a
  staging dir produce zero rebuilds while a content edit still rebuilds.
- `internal/generator`: flat-layout build → second build reports CacheHit.

## Breaking changes
- None to the artifact contract. Fingerprints change once (excluded paths),
  causing a single extra rebuild on first build after upgrade — self-healing.
