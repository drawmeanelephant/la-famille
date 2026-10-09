# Issue #642: asset_dir misconfiguration produces misleading errors

## Scope and ownership

- Owner: Devin on `t3/fix-config-cli`.
- Bug 1: `asset_dir` pointing at a regular file passes `os.Stat` in
  `CopyAssets` (`internal/asset/copy.go`), then `WalkDir` treats the file as
  the root and `CopyFile` fails with `failed to establish destination: open
  …/.public.staging-…/assets: is a directory` — blaming the destination and
  leaking the internal staging path.
- Bug 2: `asset_dir: "."` passes `Config.Validate` but every build fails at
  the copier's containment check (the project root contains the output
  directory). It cannot be made to work safely — walking the root would
  publish `content/`, `templates/`, `.git/` and the staging tree itself.

## Proposed change

- `internal/config/config.go`:
  - `ValidateResolved` (resolved, absolute paths only) stats `AssetDir` when
    it exists and rejects a non-directory with
    `asset_dir "<path>" is not a directory`; the displayed path is rendered
    project-relative when it lives under the project root. A missing asset
    dir stays legal (the copier tolerates it).
  - `validateOutputIsolation`: an `AssetDir` that resolves to the project
    root is rejected with a message explaining that copying the whole
    project tree is unsafe. `ContentDir`/template-dir == root stays legal
    (flat sites are supported for inputs that are read, not mirrored).
- `internal/asset/copy.go`: after `os.Stat` succeeds, reject a non-directory
  `AssetDir` with `asset_dir %q is not a directory` — a backstop for callers
  that skip validation, and it guarantees the staging path never surfaces.
- `internal/config/output_isolation_test.go`: move `asset_dir == project
  root` out of the allowed-layout cases into an explicit rejection test.

## Tests

- `internal/config`: `ValidateResolved` rejects a regular-file `asset_dir`
  (message `asset_dir "assets" is not a directory`) and rejects
  `asset_dir: "."`; missing dir still validates.
- `internal/asset/copy_test.go`: `CopyAssets` with a regular-file `AssetDir`
  fails naming `asset_dir` and "not a directory", with no staging path in
  the error.

## Breaking changes / pipeline impact

- `asset_dir: "."` and `asset_dir` pointing at a file now fail at config
  validation with a clear message instead of a mid-build copier error. Both
  layouts already failed every build, so no working site regresses.

## Verification

- `go test ./internal/config/ ./internal/asset/`, `go test ./...`,
  `go vet ./...`.
