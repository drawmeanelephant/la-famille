---
date: "2026-07-24"
title: "Ask This Site (Local Assistant)"
author: "Jules"
---

# Ask This Site — Local Citation-Grounded Assistant

`la-famille ask` is an **opt-in, local-first** question-answering assistant
that runs entirely on your machine. It reads your existing RAG archive
(built by `la-famille rag`) plus your generated site metadata
(`graph.json`, `meta.json`, `search.json`) and serves a small loopback-only
web UI. Every answer must cite the exact source page — heading, snippet,
and a working "Open source" link — so you can verify what the assistant
claims.

The feature is **experimental**. It logs nothing about your questions by
default and never sends content off your machine. Provider adapters are
extensible; only Ollama (and a deterministic `fake` provider used in tests)
ship today.

## Quickstart

```bash
# 1. Make sure the corpus is fresh.
go run ./cmd/la-famille rag

# 2. Start the assistant. Defaults: 127.0.0.1:8090, provider ollama.
go run ./cmd/la-famille ask --model llama3.2
```

Your default browser will be opened to the local UI. Ask a question. The
answer will include bracketed citations like `[1]`, with a card below the
answer linking to the source page.

If you do **not** have Ollama running, exercise the pipeline with the
deterministic fake provider:

```bash
go run ./cmd/la-famille ask --provider fake
```

This is what the tests use; it returns a synthetic answer that includes
valid `[1]` citations so you can see the full flow without a model.

### Ask a Corpus Pack without a site checkout

```bash
la-famille ask --pack /absolute/path/corpus.tar --model llama3.2
# Deterministic citation evidence without Ollama:
la-famille ask --pack /absolute/path/corpus.tar --provider fake --no-browser
```

Pack mode needs no config, source checkout, `public/`, or loose `rag-archive/`.
It verifies the local v1 pack into a private opaque snapshot, parses the exact
verified bytes, then closes and removes the snapshot before serving. It never
extracts or executes members, reads an unrelated config, or serves unrelated
local site files. Only `rag-content.md` and its packaged citation/link metadata
enter retrieval; repository/config bundles and taxonomy HTML do not.

Generated metadata preserves source-card titles, slugs, and deployment
subpaths, including custom content directories. Pack citation URLs still point
at the published site paths; the pack does not contain or serve page HTML.
Corrupt/unsupported packs, missing content, and malformed consumed payloads
fail at startup rather than falling back to local directories.

`--pack` requires an absolute local file path. Do not combine it with
`--rag-dir`, `--output`, `--rebuild` (even `--rebuild=false`), `--config`,
`--project-root`, or the dataset-based `--eval` flags. Directory-backed Ask
keeps its existing behavior. Optional embeddings and graph expansion reuse
the existing scorers; pack embeddings use the OS user cache under
`la-famille/ask-packs`, or explicit `--embedding-cache`, and never read the
working directory's build cache.

The pack-mode fake provider emits a synthetic `[1]` citation, not a factual
model answer. Inspect the returned source-card excerpt to verify retrieved
facts. [Compiled-binary evidence](https://github.com/drawmeanelephant/la-famille/blob/master/docs/pack-backed-ask-demo.md) records
both fixture note titles and deployment-subpath URLs.

### Optional hybrid retrieval

Lexical BM25-lite ranking remains the default. To opt in to local semantic
embeddings, start Ollama with an embedding model installed (for example,
`ollama pull nomic-embed-text`) and run:

```bash
go run ./cmd/la-famille ask --embeddings --embedding-model nomic-embed-text
# Instantly revert to the unchanged lexical ranker:
go run ./cmd/la-famille ask --no-embeddings
```

The completion model (`--model`) and embedding model are independent.
`--provider fake --embeddings` still uses Ollama for embeddings. No vector
database or external API is needed. Corpus chunks are embedded once into a
private `.la-famille-vectors.json` next to the project build cache, keyed by
the generator's fingerprint, embedding model and exact corpus digest; query
vectors are computed on demand. A valid index is reused without embedding
chunks on the next run. Changing site inputs, model or corpus invalidates it.
If Ollama is unavailable, serving falls back to the original lexical ranker.
Only the local Ollama endpoint is accepted, including on redirects.

### Optional link-graph retrieval

Graph expansion is a separate **lexical-only** mode, not an embedding feature:

```bash
go run ./cmd/la-famille build
go run ./cmd/la-famille rag
go run ./cmd/la-famille ask --graph-expansion --no-embeddings
```

The checkbox **Expand along site links** can be switched on and off for the
same question without restarting. Both checkbox states compare against the
original lexical scorer. On an embedding-enabled server, hybrid remains the
default until you change this checkbox; after that, both states select the
lexical comparison. `--graph-expansion --embeddings` is rejected.

Graph mode reads the existing `graph.json` and `backlinks.json`, the same
directed links used by `/graph/`. It promotes reciprocal neighbors only when
they have strong, complementary lexical evidence. Two independently matching
pages can pull in their shortest connecting route, including a bridge with no
query words. Routes are limited to four hops, 256 visited pages, and the request's
chunk budget. Only rendered, published corpus pages participate, not missing
stubs, repository files, or unrendered notes. A single-page query does not
automatically add every neighbor.

**How I got there** shows the ordered, cited route with source links. Traversal
can follow backlinks; the outbound edge list preserves actual link direction.
Links establish navigation, not causation or proof of a factual relationship.
The model receives source text, not just the source-card preview, and must cite
every route page. A partially cited route answer falls back to no-answer.
The citation verifier checks source membership, not factual accuracy.

Missing graph artifacts leave lexical retrieval usable. Malformed optional
graph artifacts produce a warning. Rebuild both site and archive after changes
so paths and source text stay consistent.

API clients can post `"graph_expansion": true` or `false` to `/api/ask` to select
the graph or lexical arm explicitly. Omitting the field retains startup
defaults. Responses include `paths` (page IDs, chunk IDs, citation keys, URLs,
directed edges) and the selected `diagnostics.retrieval_mode`.

## Privacy Guarantees

The default configuration is **loopback-only**: the assistant binds to
`127.0.0.1` and never accepts requests from outside your machine. The CLI
also refuses to bind to `0.0.0.0` unless you explicitly pass
`--expose-host`, in which case it logs a clear warning so you cannot
expose the service by accident.

In its default configuration:

- The HTTP listener binds only to loopback addresses.
- Requests whose `Host` header does not name that loopback address are
  rejected with `403`. Binding to `127.0.0.1` keeps other *machines* out but
  not other *origins*: a web page can point a hostname it controls at
  `127.0.0.1` (DNS rebinding) and would otherwise be treated as same-origin,
  able to query the assistant and read the answers. Checking `Host` closes
  that path. The check is skipped when you pass `--expose-host`, since an
  intentionally exposed deployment is reached under other hostnames.
- No external network calls are made. Ollama is contacted at
  `http://127.0.0.1:11434` when configured.
- Prompts, answers, and the corpus text are never logged. Only the
  timing of retrieval and generation is exposed via the diagnostics
  drawer.
- Request bodies are capped at 8 KB so the endpoint cannot be used to
  pipeline arbitrary content out of the machine.

## CLI Reference

```text
la-famille ask [flags]

Flags:
  --provider string       Local provider (ollama, fake). Default "ollama".
  --model string          Model identifier, e.g. "llama3.2". Empty falls back to the provider default.
  --host string           Bind address. Default "127.0.0.1".
  --port int              HTTP port. Default 8090.
  --rag-dir string        Path to the RAG archive directory. Default "rag-archive".
  --output string         Generated site output directory (used for citation URLs). Default "public".
  --pack string           Verified local Corpus Pack, absolute path; replaces directory inputs.
  --rebuild               Regenerate the RAG archive inline before starting the server.
  --no-browser            Do not try to open the UI in a browser.
  --max-context int       Maximum context characters per request (default 6000).
  --eval string           Run the BM25-lite retrieval evaluation against a golden-question JSON dataset.
  --eval-k int            Override eval chunk depth (1-100); zero uses the dataset depth.
  --graph-expansion       Opt in to bounded lexical link-graph retrieval.
  --eval-compare-graph    Run graph on/off against lexical scoring in one eval.
  --embeddings            Opt in to local Ollama embeddings and hybrid fusion.
  --no-embeddings         Force lexical ranking, overriding --embeddings.
  --embedding-model       Local embedding model (default nomic-embed-text).
  --embedding-cache       Private vector cache directory (optional).
  --verbose               Verbose logs.
  --expose-host           Allow non-loopback binds. Warnings are emitted at startup.
```

### Retrieval evaluation

Pass `--eval <dataset.json>` to run the checked-in golden-question harness
instead of starting the assistant UI. Fixture paths in the dataset are
resolved from the project root. The original version-1 nested dataset is
retained. The report includes per-question recall, precision, missing pages,
optional graph-route coverage, class summaries and aggregate regression floors.
Eval defaults to `fake`, starts no listener and builds content-only fixtures in
disposable projects with real generated citation and graph metadata.

The command fails if a question or dataset gate fails. Strict unanswerable
controls require **zero retrieval** and the canonical server fallback; nonempty
near-miss retrieval fails without invoking a model. `--eval-k 8` overrides the
chunk budget; recorded recall@5 comparison is skipped at other depths. K limits
chunks before page deduplication. Precision counts relevant unique retrieved
pages divided by all unique retrieved pages, not by K.

The separate `golden-questions-hard.json` dataset has twelve-page linked sites,
paraphrase probes, decoys and a missing graph bridge. With lexical ranking it
passes **8/8** questions, including strict zero-context abstention for the absent
sensor warranty duration. Answerable recall@5 remains **0.5833** and precision@5
**0.3333**; the original regression set remains **9/9**, recall **1.0000** and
precision **0.4375**. Neither frozen dataset nor its thresholds changed.

Before scoring, lexical retrieval requires a strict majority of unique
meaningful query terms to occur in the indexed corpus (body, headings or title).
Conversational and structural words such as “what”, “they”, “page” and “heading”
do not count toward that coverage. If half or more of the terms are absent, it
returns no context; graph expansion respects the same guard. Minor vocabulary
gaps remain allowed and accepted queries retain their BM25 scoring/order.
Queries composed only of ignored words also retrieve nothing.

This is a conservative **lexical coverage heuristic**, not proof that the site
answers every requested fact. It can reject answerable sparse paraphrases and
can miss unsupported questions whose words are common in the corpus, even on
unrelated pages. Independent near-miss/answerable tests include non-warranty
topics and a corpus that actually supplies warranty duration. Strict retrieval
abstention bypasses completion entirely; these tests do not demonstrate general
live-model factual correctness or that all hallucinations are prevented.

The hard set's zero-floor paraphrase probes remain measurements, not successful
retrieval claims. Its frozen evidence labels still exclude the graph bridge:
the optional hard graph comparison can fail its precision non-dilution gate
even though its individual questions pass. Use the separate graph dataset for
grounded-route acceptance. RRF does **not** mathematically guarantee improved
recall/precision. Dense retrieval is a separate opt-in arm and does not inherit
the lexical coverage guarantee; report actual local-model measurements.

Eval's fixture builds are disposable. Embedding indices persist in the user's
cache directory (`la-famille/ask-eval` under the OS cache by default), keyed
by fixture identity and content digest, or in `--embedding-cache`. The report
identifies hybrid ranking or an explicit lexical fallback when Ollama is
offline. No-answer controls retain their strict zero-retrieval gate.
Per-site measurement JSON records corpus/index hashes, constructor time,
chunk/query embedding counts and time, and each query's actual ranker mode;
durations are nanoseconds and exclude completion.

[Real Ollama measurements for #602](https://github.com/drawmeanelephant/la-famille/blob/master/docs/ask-retrieval-602.md)
are complete: `nomic-embed-text` improves hard recall@5 from 0.5833 to 1.0000,
but original precision regresses and hybrid fails all three abstention controls.
Lexical and offline fallback pass them, including the warranty check fixed by
#604. Warm indices are byte-identical with zero chunk re-embedding. This is
retrieval/cache evidence, not a live-completion benchmark; ready for owner review.

```bash
go run ./cmd/la-famille ask \
  --eval assets/testdata/ask-eval/golden-questions.json \
  --provider fake
go run ./cmd/la-famille ask --eval assets/testdata/ask-eval/golden-questions-hard.json --embeddings
go run ./cmd/la-famille ask --eval assets/testdata/ask-eval/golden-questions-hard.json --no-embeddings
go run ./cmd/la-famille ask --eval assets/testdata/ask-eval/golden-questions-graph.json --eval-compare-graph --no-embeddings
```

The separate Phase 3 graph dataset adds three multi-hop questions and two
single-page controls without changing either previous golden file. Its
`require_grounded_path` gate checks that the answer names and cites every page
on the labeled route, not just that retrieval found them. With the deterministic
fake provider, graph off/on gives recall@5 **0.8667 → 1.0000** and precision@5
**0.4867 → 0.6067**. Single-page precision is unchanged, including all eight
frozen regression questions (aggregate 0.4375). Fake route answers explicitly
identify themselves as synthetic: these measurements prove retrieval, prompt,
citation, and UI contracts, not real-model reasoning quality. Use
`--provider ollama` for local completion checks.

The paired report keeps both arms, per-question retrieved pages, route coverage,
and aggregate deltas. It fails on a graph gate failure, decreased aggregate
recall/precision, or decreased recall/precision on any single-page question.
The lexical arm is expected to fail the new multi-hop grounding gates; this
does not conceal its results. Lexical ranking passes the frozen hard warranty
abstention gate; the separately measured hybrid arm does not.

### Examples

```bash
# Use a different port and an explicit model.
go run ./cmd/la-famille ask --model llama3.2 --port 8091

# Pin the assistant to the same address as the rest of the dev server.
go run ./cmd/la-famille ask --host 127.0.0.1 --port 8090

# Rebuild the archive before starting.
go run ./cmd/la-famille ask --rebuild --model llama3.2

# Skip autostarting the browser.
go run ./cmd/la-famille ask --no-browser
```

## How It Works

```text
            ┌──────────────┐
            │ UI (HTML/CSS │  loopback UI served by internal/ask
            │ JS, embedded)│
            └──────┬───────┘
                   │ POST /api/ask
            ┌──────▼───────┐
            │ internal/ask │  orchestrator
            └──────┬───────┘
                   │ 1. corpus load
            ┌──────▼───────┐
            │ internal/    │  BM25-lite lexical ranking
            │ retrieval    │  + heading-bounded chunking
            └──────┬───────┘
                   │ 2. top-K chunks + stable keys
            ┌──────▼───────┐
            │ internal/llm │  Provider interface; ollama today
            └──────┬───────┘
                   │ 3. answer + [N] citations
            ┌──────▼───────┐
            │ citation     │  Verifier strips invented keys,
            │ verifier     │  drops them from the UI, returns "no answer"
            └──────┬───────┘
                   │ 4. JSON w/ sources + diagnostics
            ┌──────▼───────┐
            │ UI renders   │  Renders answer, citation tags, source cards
            └──────────────┘
```

## Retrieval and Citations

Each question flows through:

1. **Loading.** The corpus is loaded once at startup from
   `rag-content.md`, `rag-system.md`, and `rag-config.md` plus any
   available `meta.json` / `search.json` from the generated site.
2. **Chunking.** Markdown is split at `##` / `###` heading boundaries into
   stable chunks. Each chunk carries its page ID, title, heading trail,
   generated URL, and approximate token count.
3. **Ranking.** A small BM25-lite scorer (in-memory inverted index,
   `k1=1.5`, `b=0.75`) returns the top-K chunks for a query. When opted in,
   Ollama embeddings rank chunks by cosine similarity, then reciprocal rank
   fusion combines the complete lexical and dense rankings before top-K.
   Graph mode instead uses lexical seeds and bounded link traversal, with no
   dense ranking or embedding calls.
4. **Prompt construction.** The top-K chunks are flattened into a
   numbered citation-key map. The model sees the keys, the heading
   trail, and the chunk text — but never a fabricated URL.
5. **Generation.** The provider is queried for a single completion.
   The local Ollama adapter targets `http://127.0.0.1:11434`. Local
   providers are pluggable behind `internal/llm.Provider`.
6. **Verification.** Bracketed key patterns in the answer (`[1]`,
   `[ 42 ]`, etc.) are checked against the citation map. Keys the
   model invented are dropped and surfaced as warnings in the UI.
7. **No-answer fallback.** If retrieval returns nothing or the model
   produces a source-less answer, the client receives a *no-answer*
   status with the canonical "This site does not provide enough
   information to answer that question." message.

## UI Features

The shipped UI is a single HTML page served at `/`, no JavaScript
framework, no CDN dependencies. It includes:

- **Question input.** Press Enter to submit, Shift+Enter to insert a
  newline, Escape to clear.
- **Status indicator.** A live region (`aria-live="polite"`) reports
  *Ready*, *Retrieving…*, *Answer ready*, or *Error*.
- **Citation tags.** Inline `[1]`, `[2]` markers are rendered as small
  badge spans. They are visual only — the source cards below the
  answer do the actual citation work.
- **Source cards.** Each verified key gets a card with title, heading
  trail, excerpt, and a working "Open source" link. Cards include the
  stable chunk ID so you can grep the corpus for the same text.
- **Graph toggle and route.** Compare lexical graph on/off in the same server.
  Fully cited connecting paths appear as an ordered **How I got there** list.
- **Copy button.** Copies the answer plus a "Sources:" footer with
  URLs. Uses the system clipboard when available.
- **Diagnostics drawer.** Toggled by the "Diagnostics" button or `Esc`.
  Shows corpus version, document count, chunk count, provider, model,
  bind address, and last retrieval/generation timings.

The UI ships in `assets/ask/` and is embedded into the `la-famille`
binary by `internal/ask/ui_assets.go`. **It is never copied into the
generated `public/`** — the assistant is a local developer/author tool,
not a hosted chat service.

## Architecture Constraints

The feature respects the project's package boundaries:

| Package | Responsibility |
| --- | --- |
| `internal/llm` | Provider interface, Ollama adapter, deterministic test fake. |
| `internal/retrieval` | Corpus loading, lexical reranker, citation verification, prompt builder. |
| `internal/ask` | HTTP server, lifecycle, embedded UI, request/response types. |
| `cmd/la-famille/ask.go` | Cobra command, flag validation, signal handling. |
| `cmd/la-famille/tui_ask.go` | TUI launch helper and screen view. |

The build pipeline (`build`), the dev server (`serve`), the
checker (`check`), and the TUI's other screens are unchanged. The
generator never sees the `ask` UI.

## Troubleshooting

| Symptom | Likely cause | Fix |
| --- | --- | --- |
| Server prints `provider unavailable` and exits. | Ollama daemon is not running. | Run `ollama serve` and ensure the model is pulled (`ollama pull llama3.2`). |
| Empty answers. | Your RAG archive is empty or stale. | Re-run `la-famille rag` (or pass `--rebuild` to `ask`). |
| "Refusing to start" when binding a public IP. | Loopback-only safeguard. | Pass `--expose-host` (and accept the privacy implications). |
| Port already in use. | Something else is bound to 8090. | Run with `--port 8091` and update `--host` accordingly. |
| Citations removed from the answer. | The model emitted keys that don't exist in the retrieved set. | The retrieval has changed or the model is hallucinating. Rebuild and retry. |
| Browser doesn't open. | `xdg-open` / `open` / `rundll32` missing. | Navigate to the printed URL manually or pass `--no-browser`. |
| Bugs you cannot reproduce. | You hit an edge case. | Open a PR with `go test -race` output and a minimal corpus. |
