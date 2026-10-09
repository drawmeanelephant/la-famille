# Issue #641: init --force must not clobber through a symlinked config.yaml

## Scope and ownership

- Owner: Devin on `t3/fix-config-cli`.
- `writeInitialConfig` (`cmd/la-famille/main.go`) checks `os.Stat`, which
  follows symlinks, then `config.WriteDefaultWithLayout` truncates via
  `os.WriteFile` — writing defaults through `config.yaml -> shared.yaml` and
  destroying the shared team file. `la-famille new` already refuses to write
  through a symlinked destination.

## Proposed change

- `writeInitialConfig`: inspect the target with `os.Lstat`; when it is a
  symlink, refuse — with or without `--force` — matching `new`'s contract
  ("--force means replace the file you named, never follow a link and
  truncate the other end"). The refusal tells the operator to remove the
  link or select another file via `--config`. This also covers dangling
  symlinks, where `os.Stat` reported "not exist" and the write would create
  the linked target.

## Tests (cmd/la-famille)

- Extend `TestWriteInitialConfig` in `main_test.go`:
  - `init --force` on `config.yaml -> shared.yaml` errors, names the symlink,
    and leaves `shared.yaml` byte-identical (no `.bak` overwrite of the
    target either).
  - Non-force init on a symlinked config also refuses.
  - Dangling symlink refused rather than writing through to create the
    target.

## Breaking changes / pipeline impact

- `init --force` on a symlinked config.yaml now exits non-zero instead of
  replacing the link target. No asset-pipeline impact.

## Verification

- `go test ./cmd/`, `go test ./...`, `go vet ./...`.
