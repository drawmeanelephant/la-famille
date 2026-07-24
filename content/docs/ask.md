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
  --rebuild               Regenerate the RAG archive inline before starting the server.
  --no-browser            Do not try to open the UI in a browser.
  --max-context int       Maximum context characters per request (default 6000).
  --verbose               Verbose logs.
  --expose-host           Allow non-loopback binds. Warnings are emitted at startup.
```

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
   `k1=1.5`, `b=0.75`) returns the top-K chunks for a query. Vector
   embeddings are intentionally **opt-in** and not required.
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
