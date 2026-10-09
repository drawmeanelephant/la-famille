# Issue #639: config — reject unknown YAML keys

## Scope and ownership

- Owner: Devin on `t3/fix-config-cli`.
- `config.Load` (`internal/config/config.go`) parses `config.yaml` with a plain
  `yaml.Unmarshal`, so typo keys (`output_dirr`, `site_nme`) are silently
  dropped and the build runs on defaults with exit 0 and no warning.

## Proposed change

- `internal/config/config.go`: decode with `yaml.NewDecoder` +
  `KnownFields(true)` so unknown top-level and nested keys (e.g. inside
  `site_links` entries) are rejected, naming the offending key and line.
  Treat `io.EOF` (empty file) as success to preserve the existing
  empty-config-means-defaults behavior. Keep returning the zero `Config` on
  error per the Load contract comment.

## Tests (same package)

- `internal/config/config_test.go`: new test asserting `Load` fails on a
  config containing `output_dirr`/`site_nme`, that the error names the bad
  key, and that the returned Config is the zero value.

## Breaking changes / pipeline impact

- Config files containing misspelled or unrecognized keys now fail `build`,
  `check`, `serve`, etc. with a load error instead of silently using
  defaults. All checked-in fixtures (`config.yaml`, `website.yaml`,
  `assets/testdata/*/config.yaml`) use only known keys.

## Verification

- `go test ./internal/config/`, `go test ./...`, `go vet ./...`.
