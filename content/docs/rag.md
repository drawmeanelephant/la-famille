---
date: "2026-07-09"
title: "RAG Export Guide"
author: "Jules"
---

# Retrieval-Augmented Generation (RAG) Export

One of the unique features of La Famille is its ability to natively export your entire site into an optimized archive designed for Large Language Models (LLMs). This feature makes it trivial to use your documentation, blog, or wiki as the context foundation for AI tools.

## How It Works

Instead of feeding raw HTML or disjointed markdown files into an LLM, La Famille's RAG exporter (`internal/ragexport`) processes your generated output and metadata graph to create clean, structured text representations.

By structuring the data contextually, the LLM can understand not just the content of individual pages, but how the pages relate to one another within the site's architecture.

## Generating the Archive

To generate the RAG archive, you can use either the CLI or the TUI.

**Using the CLI:**
```bash
go run ./cmd/la-famille rag
```

The archive destination and source root are explicit CLI contracts:

```bash
la-famille --project-root /path/to/site rag \
  --output /path/to/site/public/rag-archive \
  --content content --asset-dir assets --template templates/layout-octoburger.html
```

This is the Pages-friendly form: it writes only to the selected archive
directory and does not copy files into or mutate the source checkout. Relative
values are resolved from `--project-root`.

From a parent directory, use `--project-root sites/zai rag --output public/rag-archive`
to write to `sites/zai/public/rag-archive`. Do not repeat `sites/zai` in the
relative `--output` value: it is already relative to the selected project root,
not the current directory.

**Using the TUI:**
1. Run `go run ./cmd/la-famille tui`
2. Select **RAG Export** from the main menu.

## Output Files

When the export process completes, it places the resulting datasets into the `rag-archive/` directory. The primary outputs include:

*   **`rag-system.md`**: Contains the overarching system prompt and metadata structures, providing the LLM with instructions on how to interpret the site data.
*   **`rag-config.md`**: Captures configuration parameters and global state details relevant to the site's generation.
*   **`rag-content.md`**: The bulk of the archive. This file compiles all the textual content from your pages, stripped of unnecessary HTML bloat, while preserving semantic meaning and linking structure.

`rag-archive/` is generated output. It is intentionally ignored by Git: regenerate it when needed, and do not edit or commit its files. The deployment workflow publishes a freshly generated archive at `public/rag-archive/` as part of the complete Pages artifact.

## Using the Output

You can take these generated files and use them as system context when configuring custom GPTs, uploading them to Claude Projects, or using them in local AI pipelines (like LangChain or LlamaIndex) to allow the model to answer questions accurately based exclusively on your site's data.

## Archive compatibility

The archive format remains available to external tools. Each file uses
`<file path="...">`, `<content>`, `</content>`, and `</file>` framing.
Body lines beginning with a framing marker (including already-escaped markers)
gain one leading backslash. Readers remove exactly one backslash from those
escaped marker lines to recover the original content.

[Corpus Packs](corpus-packs.md) package `rag-content.md` together with generated
site metadata for verification and delivery without the source checkout.
The retired built-in assistant does not change RAG export or its archive format.
