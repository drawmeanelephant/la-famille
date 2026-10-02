# Corpus Packs v1: compiled-binary demonstration

Recorded 2026-10-02 for #583, subissues #609 and #610, milestone 6.
Base: master at `08f9a8e`. Platform: darwin/arm64, Go 1.27.1.
This demonstration uses only local files and the compiled application.

## Reproduce from the repository root

```bash
set -eu
repo=$(pwd)
demo=$(mktemp -d /tmp/la-famille-pack-demo.XXXXXX)

cp -R "$repo/assets/testdata/sites/artisanal-ceramics/content" "$demo/content"
cp -R "$repo/assets/testdata/sites/artisanal-ceramics/assets" "$demo/assets"
cp -R "$repo/templates" "$demo/templates"
cat > "$demo/config.yaml" <<EOF
site_name: "Kintsugi & Co. Studio"
siteurl: "https://kintsugi.example.com"
content_dir: content
asset_dir: assets
output_dir: public
rag_dir: rag-archive
template: templates/layout.html
graph_explorer: true
EOF

go build -o "$demo/la-famille" ./cmd/la-famille
"$demo/la-famille" --version --json
"$demo/la-famille" --project-root "$demo" build
"$demo/la-famille" --project-root "$demo" rag
"$demo/la-famille" --project-root "$demo" pack build --output first.tar
"$demo/la-famille" --project-root "$demo" pack build --output second.tar
cmp "$demo/first.tar" "$demo/second.tar"
shasum -a 256 "$demo/first.tar" "$demo/second.tar"
"$demo/la-famille" --project-root "$demo" pack verify first.tar
```

Copy the templates directory, including its partials, rather than moving the
layout alone. All configuration paths are local to the isolated demo.
`rag` produces three bundles; the pack includes only `rag-content.md`.

## Observed successful results

- Binary identity: version `dev`, release commit/date `unknown`,
  target `darwin/arm64`, Go version `go1.27.1`.
- Site build: 21 pages, no content warnings.
- Each pack: 23 payload members.
- Content root:
  `937fc10dbb15c44f7a8224c2e4606c63f40c79c89b03c3ea60e8e99a71bdc0ca`.
- `cmp`: exit 0, byte-identical.
- SHA256 of both complete packs:
  `9f044628f055ebe0881839aabd2fae179e90536eb95ab229706fa7212df2bd14`.
- `pack verify first.tar`: exit 0, 23 members and the same content root.

The complete archive hash depends on the producer's Go version and build
identity; a different toolchain need not produce the recorded archive hash.
The determinism criterion compares identical inputs and producer provenance.

## Tamper demonstration

Change one payload byte without changing the manifest or archive headers.
Python is only a local demonstration aid, not an application dependency:

```bash
python3 - "$demo" <<'PY'
import pathlib, sys, tarfile
root = pathlib.Path(sys.argv[1])
with tarfile.open(root / "first.tar") as archive:
    member = archive.getmember("graph.json")
data = bytearray((root / "first.tar").read_bytes())
data[member.offset_data] ^= 1
(root / "tampered.tar").write_bytes(data)
PY

set +e
"$demo/la-famille" --project-root "$demo" pack verify tampered.tar
status=$?
set -e
test "$status" -eq 1
```

Observed exit: **1**. Error detail:

```text
member "graph.json" SHA256 mismatch: expected ba6e6947ae144f22f0ce7b86622909b0b8a9a71e65a21a3826f27e59259056d9, actual 5fcf061b4a2d569aff46cb9d46ce880bdc0617b5d33443f334e75980f1af8476
```

Neither successful nor failed verification creates extracted files.

## Regression tests and validation

Same-package tests in `internal/pack` cover the fixture, byte-identical builds,
allowlist exclusion, source changes during writing, canonical headers, malformed
archives/manifests, unsafe/duplicate paths, missing/unlisted members, sizes,
hashes, content roots, resource bounds and verified unknown members.
CLI tests in `cmd/la-famille/pack_test.go` cover syntax, relative paths,
provenance, failure exits and verification with unusable configuration.

Commands run:

```bash
./format_check.sh
go test ./...
go vet ./...
go test -race ./...
go test -shuffle=on -count=2 -parallel=4 ./...
go test ./internal/pack -run '^$' -fuzz FuzzVerify -fuzztime 5s -parallel 4
bash .github/scripts/quality/check-coverage.sh
bash .github/scripts/quality/check-file-sizes.sh
bash .github/scripts/quality/check-agents-md.sh
bash .github/scripts/quality/check-log-scrub.sh
bash .github/scripts/release/test.sh
```

The preinstalled lint binary was built with Go 1.24 and could not load this
repository's Go target. Installing the CI-pinned 2.13.1 from source also failed
because its module resolved a package from 2.14.0. A temporary
`golangci-lint` 2.14.0 built with Go 1.27.1 was placed first in `PATH` for the
quality script and commit hook. No module, lint configuration or dependency
files changed. With that compatible tool, `format_check.sh` passed with zero
lint issues. The other checks passed, including the flagship static publishing
contract. Final statement coverage: repository 83.5% (75% floor), pack package
88.9%. The bounded fuzz run passed after 4,181 executions.

No release, deployment, remote corpus download or Ledger rollout was performed.
