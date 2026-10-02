# Corpus Packs: compiled-binary local diff/apply demonstration

Recorded 2026-10-02 for #583, subissues #613/#614, milestone 7.
Base: current master `ffdc421`, including merged #612.
Platform: darwin/arm64, Go 1.27.1. All corpus operations use local files.

## Reproduce from the repository root

```bash
set -eu
repo=$(pwd)
demo=$(mktemp -d /tmp/la-famille-pack-delta-demo.XXXXXX)

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
"$demo/la-famille" --project-root "$demo" pack build --output before.tar
shasum -a 256 "$demo/before.tar"

python3 - "$demo" <<'PY'
import pathlib, sys
page = pathlib.Path(sys.argv[1]) / "content/care-guide.md"
text = page.read_text().replace("  - care\n", "  - preservation\n", 1)
text += "\n## Local pack update\n\nDry each vessel fully before storage. See the [studio overview](index.md).\n"
page.write_text(text)
PY

"$demo/la-famille" --project-root "$demo" build
"$demo/la-famille" --project-root "$demo" rag
"$demo/la-famille" --project-root "$demo" pack build --output after.tar
"$demo/la-famille" --project-root "$demo" pack diff before.tar after.tar --output delta.tar
"$demo/la-famille" --project-root "$demo" pack diff before.tar after.tar --output repeated.tar --json > "$demo/report.json"
cmp "$demo/delta.tar" "$demo/repeated.tar"
"$demo/la-famille" --project-root "$demo" pack apply before.tar delta.tar --output result.tar
"$demo/la-famille" --project-root "$demo" pack verify result.tar
cmp "$demo/after.tar" "$demo/result.tar"
shasum -a 256 "$demo/before.tar" "$demo/after.tar" "$demo/result.tar" "$demo/delta.tar" "$demo/repeated.tar"
```

Python is only a local demonstration aid, not a pack implementation dependency.
The implementation uses standard-library Go facilities and no new modules.
RAG export produces three bundles; packs still select only `rag-content.md`.

## Observed results

- Binary: `dev`, commit/date `unknown`, `darwin/arm64`, `go1.27.1`.
- Both fixture builds: 21 pages, no content warnings.
- Base and target packs: 23 payload members each.
- Member diff: **1 added, 1 removed, 8 changed**.
- Delta: **9 payload members**, plus metadata and the complete target manifest.
  Its only names are `pack-delta.json`, `pack-manifest.json`,
  `payload/0000` through `payload/0008`.
- Change Ledger report: care-guide changed, 2 taxonomy membership changes,
  1 added link and 1 added graph edge (`care-guide` → `index`).
  Its taxonomy/sitemap removal findings are informational; pack diff is not a
  Ledger gate.
- Both `cmp` commands: **exit 0**.
- Apply and verification: **exit 0**, 23 members, target content root
  `317a895bc4a7dce3bcfba0f500146535ba2b4aa8ee127c606091ece5e10f2fbb`.
- Base content root:
  `937fc10dbb15c44f7a8224c2e4606c63f40c79c89b03c3ea60e8e99a71bdc0ca`,
  matching the prior v1 demonstration.

Complete archive SHA256:

| Archive | SHA256 |
| --- | --- |
| Base, before and after apply/failure tests | `9f044628f055ebe0881839aabd2fae179e90536eb95ab229706fa7212df2bd14` |
| Target and applied result | `424a8744d617359e103ab92997aca9c1a42012ed0b12b4c0f25cead0e5037578` |
| Delta and repeated delta | `7af2c6e90b00c30fe5054ad89eb7e713bc6aa81cfce495bccec8b33373a0d1a7` |

Added member: `tags/preservation/index.html`.
Removed member: `tags/care/index.html`.
Changed members: `backlinks.json`, `graph.json`, `graph/data.json`, `meta.json`,
`rag-content.md`, `search.json`, `site-manifest.json`, `tags/index.html`.

All 14 unchanged target payload members are absent from the delta. The changed
corpus is the **entire `rag-content.md` member**, not a page chunk.
The automated fixture records this exact boundary in
`internal/pack/testdata/artisanal-ceramics-delta.json` and asserts target
byte identity and repeated delta identity.

Complete archive hashes depend on toolchain/producer metadata. The byte-identity
contract compares canonical v1 builder packs with identical producer inputs,
not arbitrary alternate tar headers. Apply canonicalizes alternate layouts
while retaining exact verified manifest and payload bytes.

## Safe rejection using the compiled binary

```bash
python3 - "$demo" <<'PY'
import pathlib, sys, tarfile
root = pathlib.Path(sys.argv[1])
with tarfile.open(root / "delta.tar") as archive:
    offset = archive.getmember("payload/0000").offset_data
raw = bytearray((root / "delta.tar").read_bytes())
raw[offset] ^= 1
(root / "corrupted.tar").write_bytes(raw)
PY

set +e
"$demo/la-famille" --project-root "$demo" pack apply before.tar corrupted.tar --output refused-corrupt.tar
corrupt_status=$?
"$demo/la-famille" --project-root "$demo" pack apply after.tar delta.tar --output refused-wrong-base.tar
base_status=$?
set -e
test "$corrupt_status" -eq 1
test "$base_status" -eq 1
test ! -e "$demo/refused-corrupt.tar"
test ! -e "$demo/refused-wrong-base.tar"
shasum -a 256 "$demo/before.tar"
```

Observed: both failures exited **1**, both destinations absent, base SHA256
unchanged. Corrupted payload diagnosis:

```text
delta: member "payload/0000" SHA256 mismatch: expected 1e8422ecd08c79eeed072c712df478485d8b0c1b815ed5181cb51301447cc660, actual c02de46dc2421b6f8f937558a43cbb4eaea47c0ca0ef3812a2a367ec772110fd
```

Wrong-base diagnosis:

```text
wrong base SHA256: expected 9f044628f055ebe0881839aabd2fae179e90536eb95ab229706fa7212df2bd14, actual 424a8744d617359e103ab92997aca9c1a42012ed0b12b4c0f25cead0e5037578
```

Archive inspection reads headers/offsets only; no payload is extracted or
executed. Same-package regression tests also cover truncation, unsafe paths,
oversized archives/members/metadata, wrong bases with the same content root,
verified unknown members, manifest-only updates, original input replacement,
copy-time snapshot tampering, existing destinations and concurrent publication.

## Validation evidence

Passed:

```bash
./format_check.sh
go test ./...
go vet ./...
go test -race ./...
go test -shuffle=on -count=2 -parallel=4 ./...
go test ./internal/pack -run '^$' -fuzz '^FuzzVerifyDelta$' -fuzztime 5s -parallel 4
go test ./internal/pack -run '^$' -fuzz '^FuzzVerify$' -fuzztime 5s -parallel 4
bash .github/scripts/quality/check-coverage.sh
bash .github/scripts/quality/check-file-sizes.sh
bash .github/scripts/quality/check-agents-md.sh
bash .github/scripts/quality/check-log-scrub.sh
bash .github/scripts/quality/check-tech-debt.sh
```

`format_check.sh` included module hygiene and lint with **0 issues**.
The preinstalled lint binary is v1.64.5 built with Go 1.24; the repository uses
v2 configuration and Go 1.26+. A temporary golangci-lint **2.14.0**, built with
Go 1.27.1, was placed first in `PATH`. No project dependencies or lint config
changed. Repository coverage: **83.8%** (75% floor); pack package: **89.4%**.
Bounded fuzz runs passed after 2,878 delta executions and 87 full-pack executions.

No release, deployment, subscription, network corpus feature, Ask integration,
per-page repackaging or Ledger rollout change was performed.
Subissues and milestone remain open until review and merge; #583 stays open.
