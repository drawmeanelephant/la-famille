# Codex Plan: Asset Health Diagnostics

Extend `la-famille check` with optional asset-health diagnostics to detect asset anomalies, missing references, case collisions, and boundary escapes without disrupting existing exit code semantics.

## Objectives
1. Extend `internal/config.Config` with optional `CheckAssetHealth` (`check_asset_health`) and `MaxAssetSizeBytes` (`max_asset_size_bytes`) settings.
2. Add `--asset-health` and `--asset` flags to `la-famille check`.
3. Implement asset health diagnostics in `internal/checker`:
   - Large raster assets (> 5MB default threshold, configurable via `max_asset_size_bytes`).
   - Unsupported or suspicious image extensions (`.psd`, `.ai`, `.eps`, `.tiff`, `.heic`, `.raw`, etc.).
   - Asset paths escaping configured asset root.
   - Case collisions and duplicate destination risks in asset directory.
   - Missing referenced assets in content files.
4. Reuse existing `internal/asset` ignore and path safety logic (`.gitignore`, `.go`, `testdata/`, `pathutil.IsSafePath`).
5. Ensure warnings do not trigger command failure by default, preserving error vs. warning exit code semantics.
6. Add unit and CLI tests covering thresholds, path safety, ignore rules, missing references, and deterministic finding order.

## Planned Changes
- `internal/config/config.go`: Add `CheckAssetHealth` and `MaxAssetSizeBytes` fields.
- `internal/asset/copy.go`: Export ignore rule parsing and matching helpers (`IgnoreRule`, `ParseIgnoreRules`, `IsIgnored`) for checker reuse.
- `internal/checker/checker.go`: Implement `validateAssets` and integrate into `Validate`.
- `cmd/la-famille/check.go`: Add `--asset-health` and `--asset` flags to `checkCmd`.
- `internal/checker/checker_test.go`: Add unit tests for asset health checks.
- `cmd/la-famille/check_test.go`: Add CLI integration tests for `la-famille check --asset-health`.
