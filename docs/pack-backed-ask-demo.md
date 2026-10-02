# Pack-backed Ask: compiled-binary evidence

Scope: #583, milestone 8, #619 and #620. Baseline: `4b95932`.
This demonstrates the new Ask input only, not the already-complete
build/verify or diff/apply milestone contracts.

## Recorded run

Compiled the working branch with `go build -o <temp>/la-famille
./cmd/la-famille`. Copied `assets/testdata/pack-ask` to a disposable site.
Used the existing `build`, `rag`, and `pack build --output <temp>/corpus.tar`
commands solely to create the Ask fixture. Deleted the entire disposable site,
including its config, notes, templates, public output, and loose RAG archive.

Started the compiled binary from a new empty consumer directory:

```bash
la-famille ask --pack /absolute/temp/path/corpus.tar \
  --provider fake --no-browser --port <free-loopback-port>
```

Actual `/api/status` values:

```json
{
  "provider": "fake",
  "document_count": 2,
  "chunk_count": 2,
  "ready": true,
  "loopback_only": true
}
```

The original fixture site did not exist. The consumer directory remained
empty. No site config, public directory, or loose archive was present.

POSTed the following JSON bodies to `/api/ask`:

```json
{"question":"What do meadow bird counts measure?"}
{"question":"How do contour maps guide meadow survey routes?"}
{"question":"What is the migration survey interval?"}
```

All three responses had `status: "answered"`, one verified `[1]` citation,
and no dropped citations. Recorded source cards:

| Question | Source-card title | Source-card URL |
| --- | --- | --- |
| Meadow bird counts | Bird Observation Notes | `/field-guide/bird-observations/` |
| Contour maps | Contour Map Notes | `/field-guide/contour-maps/` |
| Migration survey interval | Bird Observation Notes | `/field-guide/bird-observations/` |

The bird excerpt was:

> \# Bird observations Meadow bird counts use repeated field observations to estimate migration. The migration survey interval is seven days.

The map excerpt was:

> \# Local contour maps Local contour maps show elevation lines that guide meadow survey routes.

The synthetic provider prose was:

> According to [1], the answer to "What do meadow bird counts measure?" is: yes.
>
> This is a deterministic test response.

This demonstrates retrieval and citation evidence, not factual model generation.
The fake provider's “yes” is synthetic; facts are checked in source excerpts.
Both generated slugs and the deployment subpath survived the pack boundary.
The listener was terminated after the requests.

## Regression evidence

- `TestPackAskFixtureCorpusAndTwoNoteCitations`: expected two-document/two-chunk
  corpus, custom nested content root, and both source cards in one fake-provider
  answer after deleting the original site.
- `TestPackAskSeesAppliedFactWithoutSourceDirectories`: uses the existing
  diff/apply APIs to produce an updated pack, removes all source/generated
  directories and the other archives, then asks only the applied pack.
  Its source excerpt contains **three days**, not **seven days**.
- `TestLoadCorpusSnapshotSurvivesPathReplacement`: replacing the input pathname
  after verification cannot change the consumed corpus.
- `TestLoadCorpusErrorsAndCleanup`: corrupt, unsupported, missing-content,
  malformed, and hash-mismatch failures, with snapshot cleanup on error/success.
- Payload tests cover metadata/RAG malformation, consistent content-root
  resolution (including overlapping nested source paths), canonical identity, title
  enrichment/search backfill, and equality with directory-backed retrieval.
- Ask tests reject conflicting inputs and refuse to serve unrelated local files.
  Bootstrap tests bypass unrelated broken config only for pack-backed Ask.

Final validation passed: `./format_check.sh`, `go test ./...`,
`go test -race ./...`, `go vet ./...`,
`go test -shuffle=on -count=2 -parallel=4 ./...`, and `git diff --check`.
The unchanged quality script used temporary `golangci-lint` v2.14.0 because
the installed v1.64.5 was incompatible with the repository's Go version.
Lint reported zero issues. The final binary repeated all three requests
successfully after fixing the review's overlapping-source-path finding.

No subscriptions, downloads, signatures, benchmarks, deployment or release
requirements were added. Milestone 8 should close only after its reviewed PR
merges; #583 remains open.
