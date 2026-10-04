# Corpus Packs finishing evidence (#626)

> Historical evidence and checklist. Ask was retired in #629, so the
> assistant/model steps below are no longer delivery requirements. Current
> commands are in `content/docs/corpus-pack-feeds.md`; Corpus Pack delivery
> continues in #583 without a model runtime.

## Status, not a completion claim

Implementation baseline: `ffe8579` (`master`, 2026-10-03).
The hosted HTTPS/real-model acceptance demonstration is **not complete**.
Neither #626 nor #583 should close on the strength of local tests.

Concrete operator prerequisites:

- Approval to deploy a fixture through the existing Cloudflare Pages hosting.
  No production deployment has been performed by this task.
- A genuinely separate subscriber environment, containing only the compiled
  binary, durable pack state and an existing local Ollama installation.
- Real Ollama and an installed local model. In this environment `ollama` was
  not on PATH and `http://127.0.0.1:11434/api/version` was unreachable.

Do not substitute the fake provider, injected TLS test transport, or a local
directory demonstration for these prerequisites.

## Recorded local compiled-binary evidence

Reproduce:

```bash
go test ./cmd/la-famille -run '^TestPackWatchCompiledSubscriber$' -count=1 -v
```

Recorded on Darwin/arm64, Go 1.27.1. The test compiles the actual CLI, builds
the two-note `assets/testdata/pack-ask` fixture only at the publisher, and
invokes `pack publish` for both versions. It deletes the disposable source
publisher before the subscriber starts. The separately created subscriber
directory contains only its watch state, with no checkout, public output,
loose RAG archive or site build.

Recorded identities (binary provenance can change them):

| Version | Exact archive SHA256 |
| --- | --- |
| Initial, seven days | `092fbebafd29ecb7833b37c1e96b3c85bed4dd7106a509b77e7b55737e5e4920` |
| Updated, three days | `6861bc7c7a30d0a93536208dc0ec32655d00cd7948b2965b9e2815b0e2e99e41` |

The compiled subscriber cold-starts `pack watch <local-feed> --state state
--interval 50ms`. Publishing a corrupt selected delta reports:

```text
Pack poll failed; current version preserved: selected delta "...": read delta pack-delta.json header: truncated archive header: unexpected EOF
```

The test checks byte-identical previous metadata and pack, then runs the
compiled `pack verify`. Restoring the valid delta reports:

```text
Mode: delta
Pack diff: 0 added, 0 removed, 3 changed members
  ~ rag-content.md
  ~ search.json
  ~ site-manifest.json
Pages: 0 added, 0 removed, 1 changed
  ~ birds (Bird Observation Notes)
```

The new pack equals the advertised canonical full bytes exactly, the previous
pack is unchanged, verification succeeds, and an interrupt exits successfully.
This proves local binary publishing/watch/recovery wiring, **not hosted proof**.

## Automated HTTPS and polling coverage

Same-package tests use an injected DNS/socket transport, trusted test TLS
certificate and injected poll wait, without exposing a CLI security bypass:

- HTTPS cold copy and exact-base delta requests, full fallback only for absent
  matching base, unknown-member compatibility and received-byte accounting.
- Reject credentials, HTTP downgrade, origin/directory escape, encoded
  traversal, private/link-local/loopback addresses, mixed DNS answers and
  rebinding on reconnect. Untrusted TLS is rejected.
- Feed size, response length/encoding/status, interrupted body, request
  timeout, redirect limits and cancellation leave no completed pull output.
- Watch unchanged behavior, metadata durability/restart, verified orphan
  recovery, named Ledger page reporting and unavailable page semantics.
- Missing/corrupt/truncated/wrong-base/racing selected deltas and interrupted
  HTTPS updates preserve the previous current version and recover later.
- Cancellation during transfer exits cleanly without advancing current state.
- Publisher first/repeat/edit/missing-history contracts and bounded history;
  CI restores before building and saves only after trusted production deploy.

These tests cannot establish publisher authenticity or factual answer quality.

## Validation and review

Passed after the review fixes:

- `./format_check.sh`, `go test ./...`, `go vet ./...`
- `go test -race ./...`
- `go test -shuffle=on -count=2 -parallel=4 ./...`
- Coverage check: 84.5% total, above the required 75%.
- `actionlint .github/workflows/website.yml`, website Python contract tests,
  file-size, AGENTS.md and log-scrubbing checks, release tag-helper tests.
- Compiled flagship build, automatic feed publication and `publish-check`.
- `git diff --check` and staged diff whitespace checks.

The initially installed golangci-lint was built with Go 1.24 and could not
check the Go 1.26 module. A temporary v2.14.0 build using the current toolchain
passed with zero issues; repository dependencies were unchanged.
The static website check passed with existing template-fallback warnings and
a generated `docs/missing` page warning, which were not changed in this task.

The full intended change was reviewed for reuse, compatibility provenance,
correctness and efficiency. Review fixes tightened nonpublic IPv6 rejection,
subscriber metadata fields and captured-snapshot identity, and removed
duplicate whole-archive reads without dropping verification.
Hosted/model acceptance is still unavailable, not a skipped check counted
as a pass. No production deployment or issue closure has occurred.

## Operator-run hosted proof

Use a fixture path in the existing hosting, not a new service or a public
mutation of hash-addressed production artifacts. Obtain deployment approval
first. Keep hashes, commands, timestamps, model/version, answer and citation
in an evidence appendix when the run is actually performed.

### 1. Publish v1 and cold-pull separately

At a disposable fixture publisher, copy `assets/testdata/pack-ask`, compile
the CLI, and run:

```bash
la-famille build
la-famille rag
la-famille pack publish --output /absolute/feed-v1
```

Publish that complete feed at an approved HTTPS fixture path using the existing
Pages deployment. The source must name the resulting `pack-feed.json`.
At the separately provisioned subscriber, copy only the binary and run:

```bash
mkdir subscriber
cd subscriber
la-famille pack pull "$FEED_URL" --allow-https --trace-http --output cold.tar \
  >cold.log 2>cold-http.log
la-famille pack verify cold.tar
la-famille pack watch "$FEED_URL" --allow-https --trace-http \
  --state state --interval 30s >watch.log 2>watch-http.log
```

Record the subscriber's inventory and cold hash. Watch cold-starts its separate
state once; thereafter unchanged polls request only the manifest.

### 2. Edit, republish and prove delta transfer

Change the bird note's migration survey interval from seven days to three days,
rebuild/export at the publisher, and run:

```bash
la-famille build
la-famille rag
la-famille pack publish --previous /absolute/feed-v1 --retain 3 \
  --output /absolute/feed-v2
```

Publish the entire new feed atomically with the existing Pages artifact.
Within the next configured poll (allowing transfer time), capture `Mode: delta`,
the named `birds` Ledger entry, and `state/current.json`.
Verify the current pack and compare its SHA256 to the automatically generated
manifest. Retain `watch-http.log`: it must name the matching hash-addressed delta
and its received byte count, with **no full target archive request**.
An edited page may replace **all of `rag-content.md`** and other members.
Do not promise page-sized transfer, or infer savings from member count.

### 3. Ask with a real local model

Record `ollama --version`, `ollama list`, exact model tag/digest, and the binary
version. Do not download a model or install software without operator approval.
Stop any previous Ask process and select only the updated verified pack:

```bash
CURRENT=$(python3 -c 'import json; print(json.load(open("state/current.json"))["path"])')
la-famille pack verify "state/$CURRENT"
la-famille ask --pack "$PWD/state/$CURRENT" --provider ollama \
  --model "$LOCAL_MODEL" --no-browser --port 8091
```

In a second terminal, query the local API:

```bash
python3 - <<'PY'
import json, urllib.request
request = urllib.request.Request(
    "http://127.0.0.1:8091/api/ask",
    data=json.dumps({"question": "What is the migration survey interval?"}).encode(),
    headers={"Content-Type": "application/json"},
)
with urllib.request.urlopen(request, timeout=180) as response:
    print(response.read().decode())
PY
```

Record the actual answer that says **three days**, the supporting citation
title **Bird Observation Notes**, URL `/field-guide/bird-observations/`, and
excerpt containing the updated fact. A fake-provider answer or a citation alone
does not satisfy the factual-model requirement.

### 4. Corruption and recovery without mutating immutable production paths

Build a third fixture version with another controlled fact edit and exact-base
delta from v2. At an approved *disposable fixture feed path*, publish its
manifest and full pack with an intentionally corrupt selected delta. Keep the
production hash-addressed artifacts immutable.

Before the failing poll, record current metadata and archive bytes/hashes.
Capture refusal, unchanged pointer, unchanged archive hash and successful
`pack verify`. Publish a consistent valid fixture update and capture recovery
on a subsequent poll. Verify the new current pack. Record both trace logs.
Stop watch and Ask cleanly when finished.

## Merge and issue close-out

After review and merge, append the actual hosted/model evidence here and obtain
operator acceptance. Then update #583's stale delivery status and phase
checklist with the merged PR and evidence link, explicitly stating:

> Local-directory and opt-in HTTPS subscription are delivered. Git-ref sources
> are deferred, not delivered. Deltas replace archive members, including the
> whole RAG content member; there is no page-sized transfer guarantee.

Only after merge **and** evidence acceptance, close #626 and #583 with a concise
summary. No parent auto-close reference belongs in an implementation PR while
this evidence remains pending.
