---
date: "2026-10-04"
title: "Ask This Site: Retired"
description: "The experimental local assistant has been removed. Search, RAG export, and Corpus Packs remain supported."
author: "Jules"
---

# Ask This Site has been retired

The experimental assistant was removed in
[#629](https://github.com/drawmeanelephant/la-famille/issues/629).
`la-famille ask` is no longer supported and fails as an unknown command.
Its flags, HTTP API, web UI, TUI menu entry, model providers, embeddings,
caches, and evaluation harness are no longer part of La Famille.
There is no replacement assistant or model runtime.

Existing user caches, archives, content, and installed Ollama models are
left untouched. La Famille no longer creates or reads assistant embedding
caches or contacts Ollama. Legacy private caches must still not be published.

These workflows remain supported without any model installed or running:

- [Build and preview a static site](setup.md).
- [Search the site](search.md), using the same generated assets and metadata.
- [Export RAG archives](rag.md) from the CLI or the TUI's **RAG Export** action.
- [Build and verify Corpus Packs](corpus-packs.md),
  [diff and apply updates](corpus-pack-deltas.md), and
  [publish, pull, and watch feeds](corpus-pack-feeds.md).

RAG archive encoding and Corpus Pack formats are unchanged for external
consumers. Historical plans and measurement reports describe the retired
feature, not current setup instructions.
