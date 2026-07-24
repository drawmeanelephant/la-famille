# Task: Local-first "Ask This Site" Assistant (Moonshot)

Task ID: `ask-moonshot`

## Goal

Add an optional CLI command `la-famille ask` that serves a local
citation-grounded chat assistant over a generated site. The feature must work
with local LLM tooling (Ollama first) and never require hosted APIs.

## Scope

In scope:

- New `internal/llm` package: provider interface + Ollama adapter + fake provider.
- New `internal/retrieval` package: corpus loader, chunker, ranker, citation verifier.
- New `internal/ask` package: HTTP server with `/` UI + `/api/ask` + `/api/status`.
- New `cmd/la-famille/ask.go` (or new file): cobra command with all required flags.
- New static UI in `assets/ask/` (HTML, CSS, JS) shipped only at runtime, never
  copied into `public/` by the generator.
- TUI addition: "Ask This Site" menu entry that wraps the same server.
- Tests for each new package.
- Docs updates: README, cli.md, rag.md, setup.md, tui.md, new privacy page.

Out of scope (this PR):

- Vector embeddings (opt-in only, defer to follow-up).
- Remote providers (defer; keep interface ready).
- Persisted chat history.

## Architecture

```
cmd/la-famille/ask.go      -> cobra wiring, flag validation, startup banner
internal/ask/server.go      -> orchestrator, HTTP handlers, lifecycle
internal/ask/ui.go          -> embed.FS serving the local UI
internal/retrieval/loader.go-> load RAG archive + generated metadata
internal/retrieval/Chunker.go-> heading-bounded chunks with stable IDs
internal/retrieval/ranker.go-> lexical BM25-lite scorer, top-k selection
internal/retrieval/citation.go-> server-side citation validation
internal/retrieval/render.go-> render metadata (titles, headings, URLs)
internal/llm/provider.go    -> Provider interface, request/response types
internal/llm/ollama.go      -> HTTP client for http://127.0.0.1:11434
internal/llm/fake.go        -> deterministic echo provider used in tests
```

## Static-output impact

The `ask` command does not modify `public/`. The UI lives in `assets/ask/` and
is served by `internal/ask` directly via `net/http`. Any change to the build
pipeline is strictly additive (a new `assets/` subdir ignored by templates).

## Verification plan

1. `gofmt -w .` clean.
2. `go vet ./...` clean.
3. `go test -count=1 ./...` passes.
4. `go test -race ./...` passes.
5. End-to-end smoke from existing fixture using the fake provider.
6. Inspect the served UI HTML in a smoke test for accessibility markers.

## Status

- [ ] internal/llm package skeleton
- [ ] internal/retrieval package skeleton
- [ ] internal/ask package skeleton
- [ ] cmd wiring
- [ ] UI assets
- [ ] Tests (unit + integration + race)
- [ ] Docs updates
- [ ] Final validation
