# Real Ollama retrieval and cache evidence (#602)

Measured 2026-10-04. **Measurement acceptance is met, ready for owner review
and closure of #602**, not a claim that hybrid retrieval passes every gate.

## Outcome

- Hard answerable recall@5: **0.5833 → 1.0000**; precision@5:
  **0.3333 → 0.4833**.
- Original answerable recall@5: **1.0000 → 1.0000**; precision@5:
  **0.4375 → 0.3333**, a regression.
- Lexical and unavailable-embedding fallback pass all strict abstention
  controls. Real hybrid fails all three controls across the two datasets.
  Its eval commands correctly exit **1**, not 0.
- Cold runs embedded **20 original / 31 hard chunks**. Repeat runs embedded
  **zero chunks**, reused all seven indices byte-for-byte, and still performed
  **9 original / 8 hard query embeddings**.
- Index construction/load wall time (sum across sites): original
  **1252.628 → 2.343 ms**; hard **423.922 → 3.064 ms**. These are observations
  from one pair per dataset, not latency guarantees.

Merged PR #604 (`d13e48345a8c54e877379197ac5ef7c073ffd389`) is on this
master. Its diff changes neither frozen dataset. The explicit lexical runs
confirm the corrected warranty abstention and both expected recall baselines.
There was **no lexical discrepancy** to investigate. The old issue wording
about a continuing lexical warranty failure is stale. Dense retrieval remains
independent of the lexical coverage guard and supplies nonempty context for
the unsupported questions; that is the observed hybrid abstention regression.

## Provenance and conditions

The companion [raw evidence](ask-retrieval-602-results.json) contains exact
commands, UTC timestamps, process exit codes, unabridged stdout/stderr,
per-site/per-query nanosecond measurements, external hash/byte-comparison
receipts, dataset/source hashes, and provider/build metadata.

| Item | Recorded identity |
| --- | --- |
| La Famille compiler/generator source commit | `6f1764839576fe45bf5f3808aa9be92f8edee29d` (current master at measurement) |
| Measurement build | That commit plus the eval-only instrumentation in this change; `vcs.modified=true`; exact source SHA256 values and `go version -m` are in raw evidence |
| Compiled binary SHA256 | `a0184126ccbf7d70e684fed1fc5fd4a9eb1fc58ba146b04b5a84207a65b939ad` |
| Go toolchain | `go1.27.1 darwin/arm64`, gc, CGO enabled for the application |
| Go compiler build ID | `uKjYHBFGYwkqi-xJqWFY/MuiVpFI_UydqAK6EKgaF/m2mGaat--G0d2NAYqTK-/xHQmP_HlJBsHwPbvoPCH` |
| Go compiler source commit | Not embedded by this installed Homebrew toolchain; no separate Go source checkout was available. Do not infer a commit from the release version |
| Platform | macOS 27.2, build `26B5091g`, Darwin arm64, Apple M4, 17,179,869,184 bytes RAM |
| Ollama server/client | `0.35.1`; existing Homebrew binary `/opt/homebrew/opt/ollama/bin/ollama` |
| Endpoint | `http://127.0.0.1:11434`, listener verified with `lsof`; cloud disabled |
| Embedding model | `nomic-embed-text:latest`, 137M, F16 GGUF, 768 dimensions, context length 2048 |
| Model digest | `0a109f422b47e3a30ba2b10eca18548e944e8a23073ee3f3e947efcf3c45e59f` |
| Original dataset SHA256 | `96ca0f28e97014e53effd4e45e5ec2e8bcbb1c692368b56dc613926bf9823418` |
| Hard dataset SHA256 | `c34fdd10603a7c3bdfb5f4c87097e19e0ed03d327105f7f1a6e9b0946f02b5fd` |
| Task-owned working data | `/tmp/la-famille-602.dbYmuu`, isolated model store and fresh vector directories |

The first availability check failed with `[Errno 61] Connection refused`;
`ollama` was outside PATH and no Ollama app was installed. The owner explicitly
authorized installation/model download. Homebrew inspection found an existing
binary, so **no software installation was needed**. Only the authorized model
was downloaded into the isolated model store. The daemon's first startup
generated its standard local identity under `~/.ollama`; no existing identity
or user model/vector cache was cleared.

One binary was compiled before timing. Runs were sequential in this order:
lexical original, lexical hard, hybrid cold original, hybrid warm original,
hybrid cold hard, hybrid warm hard, stop daemon, unavailable original,
unavailable hard. No concurrent quality checks ran during measurements.
This was a normal developer desktop, not a controlled idle/power/thermal lab.
No artificial embedding delay or synthetic embedding provider was used.

`/api/ps` was empty before the runs. The first original chunk batch therefore
includes model startup; later sites and the hard cold-cache run use the resident
model. Post-run `/api/ps` reports the same digest with `size_vram=370031984`.
“Cold” means **absent vector indices**, not cold filesystem pages or a freshly
unloaded model for each site. The warm runs immediately repeat each dataset,
with unchanged model and corpus.

## Reproduction

Use this revision with its measurement instrumentation. Do not install or
download anything without owner authorization. The commands below use a fresh
temporary root and never clear an existing cache. Run from the repository root.
In the actual run, `RUN=/tmp/la-famille-602.dbYmuu` and the daemon PID was
`93292`; the raw evidence records every expanded eval command.

```bash
OLLAMA=/opt/homebrew/opt/ollama/bin/ollama
RUN=$(mktemp -d /tmp/la-famille-602.XXXXXX)
export OLLAMA_HOST=127.0.0.1:11434
# First ensure no existing user's daemon owns port 11434. Never kill it.
OLLAMA_MODELS="$RUN/models" OLLAMA_NO_CLOUD=1 \
  "$OLLAMA" serve >"$RUN/ollama.log" 2>&1 &
OLLAMA_PID=$!
# Wait for /api/version to succeed before the authorized pull.
"$OLLAMA" pull nomic-embed-text
go build -o "$RUN/la-famille" ./cmd/la-famille
go version -m "$RUN/la-famille"
go version -m "$(go env GOTOOLDIR)/compile"
go tool buildid "$(go env GOTOOLDIR)/compile"
sw_vers
sysctl -n machdep.cpu.brand_string hw.memsize
lsof -nP -iTCP:11434 -sTCP:LISTEN
shasum -a 256 "$RUN/la-famille" assets/testdata/ask-eval/golden-questions*.json
mkdir -m 700 "$RUN/original" "$RUN/hard"
```

Record `/api/version`, `/api/tags`, `/api/ps` with a direct loopback GET:

```bash
python3 - <<'PY'
import urllib.request
opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
for path in ("/api/version", "/api/tags", "/api/ps"):
    with opener.open("http://127.0.0.1:11434" + path, timeout=5) as r:
        print(path, r.read().decode())
PY
```

Run these exact flag combinations, retaining output **even on exit 1**:

```bash
"$RUN/la-famille" ask --eval assets/testdata/ask-eval/golden-questions.json --provider fake --no-embeddings --embedding-model nomic-embed-text --embedding-cache "$RUN/original"
"$RUN/la-famille" ask --eval assets/testdata/ask-eval/golden-questions-hard.json --provider fake --no-embeddings --embedding-model nomic-embed-text --embedding-cache "$RUN/hard"
"$RUN/la-famille" ask --eval assets/testdata/ask-eval/golden-questions.json --provider fake --embeddings --embedding-model nomic-embed-text --embedding-cache "$RUN/original"
# Snapshot original index bytes/hashes here, then repeat unchanged:
"$RUN/la-famille" ask --eval assets/testdata/ask-eval/golden-questions.json --provider fake --embeddings --embedding-model nomic-embed-text --embedding-cache "$RUN/original"
"$RUN/la-famille" ask --eval assets/testdata/ask-eval/golden-questions-hard.json --provider fake --embeddings --embedding-model nomic-embed-text --embedding-cache "$RUN/hard"
# Snapshot hard index bytes/hashes here, then repeat unchanged:
"$RUN/la-famille" ask --eval assets/testdata/ask-eval/golden-questions-hard.json --provider fake --embeddings --embedding-model nomic-embed-text --embedding-cache "$RUN/hard"
kill "$OLLAMA_PID"  # Only the task-owned daemon.
# Verify /api/version and /api/tags now return connection refused.
mkdir -m 700 "$RUN/unavailable-original" "$RUN/unavailable-hard"
"$RUN/la-famille" ask --eval assets/testdata/ask-eval/golden-questions.json --provider fake --embeddings --embedding-model nomic-embed-text --embedding-cache "$RUN/unavailable-original"
"$RUN/la-famille" ask --eval assets/testdata/ask-eval/golden-questions-hard.json --provider fake --embeddings --embedding-model nomic-embed-text --embedding-cache "$RUN/unavailable-hard"
```

The captured runner uses `subprocess.run(..., capture_output=True, text=True,
timeout=180)` with `time.perf_counter_ns()` immediately before/after it.
It retains return codes rather than treating failed gates as failed evidence.
This avoids timing Go compilation as retrieval. To reproduce process-wall
timings and external reuse receipts, wrap each expanded command as follows:

```python
before = {str(p.relative_to(cache_dir)): p.read_bytes()
          for p in cache_dir.rglob(".la-famille-vectors.json")}
start = time.perf_counter_ns()
result = subprocess.run(command, capture_output=True, text=True, timeout=180)
wall_ns = time.perf_counter_ns() - start
after = {str(p.relative_to(cache_dir)): p.read_bytes()
         for p in cache_dir.rglob(".la-famille-vectors.json")}
assert before.keys() == after.keys()  # Warm run only.
assert all(before[k] == after[k] for k in before)  # Direct bytes, not just hashes.
hashes = {k: hashlib.sha256(v).hexdigest() for k, v in after.items()}
```

Use `pathlib.Path` cache paths and import `hashlib`, `subprocess`, and `time`.
The CLI's `Measurement JSON (durations in ns; excludes completion):` line
contains independent per-site constructor hashes and actual provider-call
counts. Save it with the stdout and external receipts; do not substitute
fake-provider test counts for these real-provider runs.

## Interpretation

- K=5 limits **chunks**, then page IDs are deduplicated in rank order.
  Recall is required evidence pages found / labeled evidence pages.
  Precision is relevant unique pages / retrieved unique pages.
  Aggregates are macro averages over answerable questions only.
- Strict unanswerable controls have recall/precision **n/a**, not zero values
  included in answerable aggregates. They pass only with zero retrieved
  chunks and the canonical server no-answer message:
  “This site does not provide enough information to answer that question.”
- Empty answerable paraphrase results are recorded as **zero context**, not
  successful abstention. Their frozen zero-floor measurement gates pass even
  when recall is zero.
- Every query's actual ranker is recorded, not inferred from `--embeddings`.
  Mode **L** is `BM25-lite`; **H** is `BM25-lite + Ollama embeddings (RRF)`;
  **F** is `BM25-lite + Ollama embeddings (RRF; lexical fallback: unavailable)`.
  No query in a real hybrid run fell back.
- Both offline runs use fresh caches, actually attempt `/api/embed`, and record
  `provider unavailable: Post "http://127.0.0.1:11434/api/embed": dial tcp
  127.0.0.1:11434: connect: connection refused`. No vector index is written.
- `IndexDuration` times `NewHybridRanker` only: cache read/validation or
  chunk embedding/index write plus its lexical index construction.
  Hashing, fixture generation/load, queries, and completion are outside it.
  `ChunkEmbedding.Duration` is nested provider-call wall time, not an
  additional time to sum with `IndexDuration`.
- Query embedding duration is separate from `RankDuration`, which subtracts
  provider-call wall time from the ranking call. The residual is wall time,
  **not CPU time**. Process wall includes fixture build/load, no-answer
  verification, diagnostics/hash/output, and process overhead.
- Completion is **not benchmarked**. `--provider fake` isolates retrieval:
  answerable probes do not invoke completion; strict empty controls return
  before completion; nonempty controls fail without calling a model.
  There were zero completion calls in these datasets/runs. This does not
  establish real-model factual reasoning, answer correctness, or hallucination
  rates.

## Limitations and acceptance

These small synthetic fixtures are not a production corpus benchmark. One
cold/warm pair per dataset, normal desktop scheduling, model residency, and
filesystem cache effects limit timing generalization. Query vectors are still
computed on every warm run. The model digest was unchanged; the current cache
keys use the model **name** and corpus identity, not an automatic digest lookup.
This measurement does not test replacing a model under the same tag.

The hard graph route still has 2/3 retrieval coverage for `sensor-to-assay`;
hybrid does not recover its unlabeled bridge, and graph redesign is out of scope.
No frozen probe, label, gate, lexical scorer, threshold, or default changed.
Hybrid's three failed abstention controls and original precision regression
are acceptance **results**, not hidden blockers to this measurement task.
During measurement, no issue was closed or commented on and nothing was pushed
or deployed. The owner subsequently authorized publication as an evidence PR,
leaving #602 open for owner review and closure.
The task-owned daemon was stopped; existing user caches were not cleared.

The sections below contain every probe, including missing pages and failed
gates, plus full cache identities and separate timing/count aggregates.

## Complete results

| Run | Mode | Recall@5 | Precision@5 | Question gates | Strict abstention | Exit | Process wall ms |
| --- | --- | ---: | ---: | --- | --- | ---: | ---: |
| lexical-original | L | 1.0000 | 0.4375 | 9/9 | 1/1 | 0 | 387.375 |
| lexical-hard | L | 0.5833 | 0.3333 | 8/8 | 2/2 | 0 | 53.048 |
| hybrid-cold-original | H | 1.0000 | 0.3333 | 8/9 | 0/1 | 1 | 1458.825 |
| hybrid-warm-original | H | 1.0000 | 0.3333 | 8/9 | 0/1 | 1 | 206.122 |
| hybrid-cold-hard | H | 1.0000 | 0.4833 | 6/8 | 0/2 | 1 | 546.072 |
| hybrid-warm-hard | H | 1.0000 | 0.4833 | 6/8 | 0/2 | 1 | 116.008 |
| unavailable-original | F | 1.0000 | 0.4375 | 9/9 | 1/1 | 0 | 93.206 |
| unavailable-hard | F | 0.5833 | 0.3333 | 8/8 | 2/2 | 0 | 59.864 |

Warm and cold hybrid question results are identical in every field below.
Every warm question is also retained independently in the raw evidence.
Original classes: standard L/F recall=1.0000 precision=0.4375, H recall=1.0000 precision=0.3333.
Hard classes: multi-hop L/F/H recall=1.0000 precision=0.5000; multi-page L/F recall=0.8333 precision=0.5000, H recall=1.0000 precision=0.5000; paraphrase L/F recall=0.0000 precision=0.0000, H recall=1.0000 precision=0.4500.
All recall and precision aggregate floors pass; only the hybrid strict abstention gates fail.

### lexical-original (actual mode L)

| Site/question | Retrieved pages, in rank order | Missing labeled pages | Recall | Precision | Context/abstention | Gate |
| --- | --- | --- | ---: | ---: | --- | --- |
| anchor-links/other-page-heading | `other`, `index` | [] | 1.0000 | 0.5000 | context | PASS |
| anchor-links/unanswerable | [] | [] | n/a | n/a | canonical no-answer PASS | PASS |
| artisanal-ceramics/shino-glaze | `collection/wheel-thrown-vessels`, `index`, `journal/2026-07-15-glazing-techniques`, `care-guide` | [] | 1.0000 | 0.2500 | context | PASS |
| artisanal-ceramics/washing-stoneware | `care-guide`, `collection/wheel-thrown-vessels`, `index` | [] | 1.0000 | 0.3333 | context | PASS |
| artisanal-ceramics/pine-ash-firing | `journal/2026-07-15-glazing-techniques`, `care-guide`, `collection/wheel-thrown-vessels`, `index` | [] | 1.0000 | 0.2500 | context | PASS |
| clean-urls/author-is-cool | `bio` | [] | 1.0000 | 1.0000 | context | PASS |
| clean-urls/blog-post | `index`, `blog/post` | [] | 1.0000 | 0.5000 | context | PASS |
| ask-eval-deep-chain/northern-kiln-temperature | `observations/archive/blue-glass-run`, `observations/field-log`, `index` | [] | 1.0000 | 0.3333 | context | PASS |
| ask-eval-research-branch/cobalt-concentration | `study/samples/cobalt-clay`, `study/field-notes`, `index` | [] | 1.0000 | 0.3333 | context | PASS |

### hybrid-cold-original (actual mode H)

| Site/question | Retrieved pages, in rank order | Missing labeled pages | Recall | Precision | Context/abstention | Gate |
| --- | --- | --- | ---: | ---: | --- | --- |
| anchor-links/other-page-heading | `other`, `index` | [] | 1.0000 | 0.5000 | context | PASS |
| anchor-links/unanswerable | `index`, `other` | [] | n/a | n/a | nonempty context, FAIL | FAIL |
| artisanal-ceramics/shino-glaze | `collection/wheel-thrown-vessels`, `index`, `care-guide`, `journal/2026-07-15-glazing-techniques` | [] | 1.0000 | 0.2500 | context | PASS |
| artisanal-ceramics/washing-stoneware | `care-guide`, `index`, `journal/2026-07-15-glazing-techniques` | [] | 1.0000 | 0.3333 | context | PASS |
| artisanal-ceramics/pine-ash-firing | `journal/2026-07-15-glazing-techniques`, `care-guide`, `collection/wheel-thrown-vessels`, `index` | [] | 1.0000 | 0.2500 | context | PASS |
| clean-urls/author-is-cool | `bio`, `index`, `blog/post` | [] | 1.0000 | 0.3333 | context | PASS |
| clean-urls/blog-post | `blog/post`, `index`, `bio` | [] | 1.0000 | 0.3333 | context | PASS |
| ask-eval-deep-chain/northern-kiln-temperature | `observations/archive/blue-glass-run`, `observations/field-log`, `index` | [] | 1.0000 | 0.3333 | context | PASS |
| ask-eval-research-branch/cobalt-concentration | `study/samples/cobalt-clay`, `study/field-notes`, `index` | [] | 1.0000 | 0.3333 | context | PASS |

### unavailable-original (actual mode F)

| Site/question | Retrieved pages, in rank order | Missing labeled pages | Recall | Precision | Context/abstention | Gate |
| --- | --- | --- | ---: | ---: | --- | --- |
| anchor-links/other-page-heading | `other`, `index` | [] | 1.0000 | 0.5000 | context | PASS |
| anchor-links/unanswerable | [] | [] | n/a | n/a | canonical no-answer PASS | PASS |
| artisanal-ceramics/shino-glaze | `collection/wheel-thrown-vessels`, `index`, `journal/2026-07-15-glazing-techniques`, `care-guide` | [] | 1.0000 | 0.2500 | context | PASS |
| artisanal-ceramics/washing-stoneware | `care-guide`, `collection/wheel-thrown-vessels`, `index` | [] | 1.0000 | 0.3333 | context | PASS |
| artisanal-ceramics/pine-ash-firing | `journal/2026-07-15-glazing-techniques`, `care-guide`, `collection/wheel-thrown-vessels`, `index` | [] | 1.0000 | 0.2500 | context | PASS |
| clean-urls/author-is-cool | `bio` | [] | 1.0000 | 1.0000 | context | PASS |
| clean-urls/blog-post | `index`, `blog/post` | [] | 1.0000 | 0.5000 | context | PASS |
| ask-eval-deep-chain/northern-kiln-temperature | `observations/archive/blue-glass-run`, `observations/field-log`, `index` | [] | 1.0000 | 0.3333 | context | PASS |
| ask-eval-research-branch/cobalt-concentration | `study/samples/cobalt-clay`, `study/field-notes`, `index` | [] | 1.0000 | 0.3333 | context | PASS |

### lexical-hard (actual mode L)

| Site/question | Retrieved pages, in rank order | Missing labeled pages | Recall | Precision | Context/abstention | Gate |
| --- | --- | --- | ---: | ---: | --- | --- |
| station/capture-and-share | `rationing`, `cistern`, `index`, `water-pump` | [] | 1.0000 | 0.5000 | context | PASS |
| station/passive-cooling | `air-conditioner`, `night-purge` | `thermal-mass` | 0.5000 | 0.5000 | context | PASS |
| station/drinkable-backup | [] | `cistern`, `rationing` | 0.0000 | 0.0000 | zero context (measurement miss) | PASS |
| station/unpowered-chill | [] | `thermal-mass`, `night-purge` | 0.0000 | 0.0000 | zero context (measurement miss) | PASS |
| estuary/sensor-to-assay | `sensor`, `sensor-catalog`, `assay`, `assay-manual` | [] | 1.0000 | 0.5000 | context | PASS |
| estuary/lot-custody-and-result | `assay`, `registry`, `sensor-catalog`, `sensor` | [] | 1.0000 | 0.5000 | context | PASS |
| estuary/absent-answer-hard | [] | [] | n/a | n/a | canonical no-answer PASS | PASS |
| estuary/sensor-warranty-absent | [] | [] | n/a | n/a | canonical no-answer PASS | PASS |

### hybrid-cold-hard (actual mode H)

| Site/question | Retrieved pages, in rank order | Missing labeled pages | Recall | Precision | Context/abstention | Gate |
| --- | --- | --- | ---: | ---: | --- | --- |
| station/capture-and-share | `cistern`, `rationing`, `index`, `water-pump` | [] | 1.0000 | 0.5000 | context | PASS |
| station/passive-cooling | `air-conditioner`, `night-purge`, `thermal-mass`, `index` | [] | 1.0000 | 0.5000 | context | PASS |
| station/drinkable-backup | `rationing`, `garden`, `cistern`, `first-aid` | [] | 1.0000 | 0.5000 | context | PASS |
| station/unpowered-chill | `air-conditioner`, `rationing`, `thermal-mass`, `night-purge`, `first-aid` | [] | 1.0000 | 0.4000 | context | PASS |
| estuary/sensor-to-assay | `sensor`, `assay`, `sensor-catalog`, `assay-manual` | [] | 1.0000 | 0.5000 | context | PASS |
| estuary/lot-custody-and-result | `assay`, `registry`, `sensor`, `sensor-catalog` | [] | 1.0000 | 0.5000 | context | PASS |
| estuary/absent-answer-hard | `safety`, `assay`, `sensor-catalog`, `assay-manual`, `sensor` | [] | n/a | n/a | nonempty context, FAIL | FAIL |
| estuary/sensor-warranty-absent | `sensor`, `sensor-catalog`, `assay-manual`, `assay` | [] | n/a | n/a | nonempty context, FAIL | FAIL |

### unavailable-hard (actual mode F)

| Site/question | Retrieved pages, in rank order | Missing labeled pages | Recall | Precision | Context/abstention | Gate |
| --- | --- | --- | ---: | ---: | --- | --- |
| station/capture-and-share | `rationing`, `cistern`, `index`, `water-pump` | [] | 1.0000 | 0.5000 | context | PASS |
| station/passive-cooling | `air-conditioner`, `night-purge` | `thermal-mass` | 0.5000 | 0.5000 | context | PASS |
| station/drinkable-backup | [] | `cistern`, `rationing` | 0.0000 | 0.0000 | zero context (measurement miss) | PASS |
| station/unpowered-chill | [] | `thermal-mass`, `night-purge` | 0.0000 | 0.0000 | zero context (measurement miss) | PASS |
| estuary/sensor-to-assay | `sensor`, `sensor-catalog`, `assay`, `assay-manual` | [] | 1.0000 | 0.5000 | context | PASS |
| estuary/lot-custody-and-result | `assay`, `registry`, `sensor-catalog`, `sensor` | [] | 1.0000 | 0.5000 | context | PASS |
| estuary/absent-answer-hard | [] | [] | n/a | n/a | canonical no-answer PASS | PASS |
| estuary/sensor-warranty-absent | [] | [] | n/a | n/a | canonical no-answer PASS | PASS |

## Cache identity and byte reuse

Eval fingerprints equal the exact `CorpusDigest`, independent of disposable build
paths. Every cold `IndexSHA256Before` is `absent`. In every warm run,
`cold after == warm before == warm after`, both internally and via an external
SHA256/whole-byte comparison. All seven files reused exactly; no index was rewritten.
Full private index paths and all before/after digests appear in the raw evidence.

### original: anchor-links (3 chunks)

- Corpus digest / eval fingerprint: `536d3212b7b4e13a9ae8debae557657a6c614481c539f83a0862aaeda0c0a450`.
- Index SHA256, cold after / warm before / warm after: `df56ed0bccf85fcff205d230208ceaa8a8ce2495c6d0a213382ea23715d07697`.
- Constructor wall time: cold 935.216 ms; warm 0.257 ms.
- Chunk calls / inputs / returned vectors: cold 1 / 3 / 3; warm 0 / 0 / 0.

### original: artisanal-ceramics (8 chunks)

- Corpus digest / eval fingerprint: `daeca4489ad0e3bf6e0efb5a19a9ab70321327f97573370886f0c342f765f688`.
- Index SHA256, cold after / warm before / warm after: `169641782966ed45d123b64ff860e4cdaa22d6213fbca23671170d46d60e22a9`.
- Constructor wall time: cold 183.875 ms; warm 1.344 ms.
- Chunk calls / inputs / returned vectors: cold 1 / 8 / 8; warm 0 / 0 / 0.

### original: clean-urls (3 chunks)

- Corpus digest / eval fingerprint: `70258859b5fbd67095b32c86ef970a22c9aa11fe526691144a2621a15187cd69`.
- Index SHA256, cold after / warm before / warm after: `01acca1d80e693b32e7fd8cfae814c6dd5406e8612b81030c078a58b747f0e59`.
- Constructor wall time: cold 26.778 ms; warm 0.245 ms.
- Chunk calls / inputs / returned vectors: cold 1 / 3 / 3; warm 0 / 0 / 0.

### original: ask-eval-deep-chain (3 chunks)

- Corpus digest / eval fingerprint: `fa4998dd86d222c38757e85ae191ea936419ede6b5b142334df82418110b8d6c`.
- Index SHA256, cold after / warm before / warm after: `7042bb07754984f60aae5e1151b7ef280329a34b77ecc2ae3b2c8496e78d2cb8`.
- Constructor wall time: cold 28.102 ms; warm 0.248 ms.
- Chunk calls / inputs / returned vectors: cold 1 / 3 / 3; warm 0 / 0 / 0.

### original: ask-eval-research-branch (3 chunks)

- Corpus digest / eval fingerprint: `f813384c91e1930a4d3ba562e53027dbde4d1f160644cb61a76798ef40e294b1`.
- Index SHA256, cold after / warm before / warm after: `764ab4dfbe09fbb4e6747fa3f4681787f617a51e20ae480c29f7b77d5ff76bf0`.
- Constructor wall time: cold 78.658 ms; warm 0.248 ms.
- Chunk calls / inputs / returned vectors: cold 1 / 3 / 3; warm 0 / 0 / 0.

### hard: station (16 chunks)

- Corpus digest / eval fingerprint: `18272b4ca1583de4768e99e821a42db7c3c25c27d2c3c6d4e80698e8cc708973`.
- Index SHA256, cold after / warm before / warm after: `a859a9f2b64693183ae76a99aa32a7e480833689afbb877d0b7633fb5fababa3`.
- Constructor wall time: cold 204.492 ms; warm 1.595 ms.
- Chunk calls / inputs / returned vectors: cold 1 / 16 / 16; warm 0 / 0 / 0.

### hard: estuary (15 chunks)

- Corpus digest / eval fingerprint: `7ce42addfd652c37c52752a3edfcea286d580cd3f1a1ff16183087e0511e5771`.
- Index SHA256, cold after / warm before / warm after: `a4dad3d3d3893f8af63be66c52e98f2fc42e5b68ad82e49a81b544b1c1320f02`.
- Constructor wall time: cold 219.430 ms; warm 1.469 ms.
- Chunk calls / inputs / returned vectors: cold 1 / 15 / 15; warm 0 / 0 / 0.

## Separate embedding and ranking measurements

All times below are sums across sites/questions, in milliseconds. Chunk provider
time is contained in constructor time. Do not add it twice. Completion calls
are zero and completion time is not measured. `C/I/V` means attempted calls /
attempted input texts / successfully returned vectors. A failed input is not
a successfully embedded chunk.

| Run | Constructor ms | Chunk provider ms | Query provider ms | Rank residual ms | Chunk C/I/V | Query C/I/V |
| --- | ---: | ---: | ---: | ---: | --- | --- |
| lexical-original | 0.000 | 0.000 | 0.000 | 0.041 | 0 / 0 / 0 | 0 / 0 / 0 |
| lexical-hard | 0.000 | 0.000 | 0.000 | 0.044 | 0 / 0 / 0 | 0 / 0 / 0 |
| hybrid-cold-original | 1252.628 | 1247.746 | 105.610 | 0.519 | 5 / 20 / 20 | 9 / 9 / 9 |
| hybrid-warm-original | 2.343 | 0.000 | 110.228 | 0.170 | 0 / 0 / 0 | 9 / 9 / 9 |
| hybrid-cold-hard | 423.922 | 421.549 | 60.367 | 0.239 | 2 / 31 / 31 | 8 / 8 / 8 |
| hybrid-warm-hard | 3.064 | 0.000 | 62.841 | 0.354 | 0 / 0 / 0 | 8 / 8 / 8 |
| unavailable-original | 1.090 | 1.053 | 0.000 | 0.052 | 5 / 20 / 0 | 0 / 0 / 0 |
| unavailable-hard | 0.616 | 0.488 | 0.000 | 0.020 | 2 / 31 / 0 | 0 / 0 / 0 |

Offline attempts return no vectors, create no indices, and make no query
embedding calls after constructor fallback. Lexical mode makes no embedding
calls at all. Genuine hybrid returns one real query vector for every question,
including the strict controls. There are no artificial delays.

## Validation and handoff

All final checks passed:

```bash
go test ./internal/askeval ./internal/retrieval ./cmd/la-famille
go test ./...
go vet ./...
go test -race ./...
go test -shuffle=on -count=2 -parallel=4 ./...
PATH="/opt/homebrew/opt/golangci-lint/bin:$PATH" ./format_check.sh
git diff --check
```

The first plain `./format_check.sh` attempt failed because PATH selected
`~/.local/bin/golangci-lint` 1.64.5, built with Go 1.24, below this repository's
Go 1.26 target. The already-installed Homebrew linter 2.14.0, built with Go
1.27.1, passed with **0 issues** using the command above. No linter was
installed and no quality check was disabled or skipped.

The complete intended change was reviewed, including all new files and the
raw evidence. An independent evidence cross-check verified all 8 runs / 68
question results, exact unchanged dataset and instrumentation hashes,
cold/warm retrieval equality, external/internal index digests, and persisted
51 real 768-dimensional chunk vectors. Retrieval implementation and CLI flags
are unchanged; only eval diagnostics, their same-package tests, and documents
changed. No remaining measurement blocker; Go compiler source-commit metadata
and the timing/generalization limitations above remain explicitly disclosed.

Proposed owner completion comment (not posted):

> Real local Ollama measurements are complete in
> [docs/ask-retrieval-602.md](https://github.com/drawmeanelephant/la-famille/blob/master/docs/ask-retrieval-602.md)
> with full raw evidence. Hard recall improves 0.5833 → 1.0000; original recall
> stays 1.0000 while precision regresses. Lexical and unavailable fallback pass
> strict abstention after #604; hybrid fails all three controls. Warm runs reuse
> all seven byte-identical indices with zero chunk re-embedding. Frozen datasets,
> gates, scoring, and defaults are unchanged. Ready for owner review and closure
> of this measurement issue, not a claim that hybrid passes every retrieval gate.