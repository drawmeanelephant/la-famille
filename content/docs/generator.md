---
title: "How La Famille Works"
author: "Jules"
date: "2026-06-18"
---

# Inside La Famille

La Famille is a static site generator written in Go. Let's kick the tires and review how it converts Markdown into HTML.

## The Pipeline

The process lives in `cmd/la-famille/main.go` and works in several passes:

1. **Template Parsing:** Loads `templates/layout.html` to establish the outer skeleton of every generated page.
2. **Metadata Gathering (Pass 1):** Walks the `content/` directory. For every `.md` file, it parses the YAML frontmatter. If parsing fails, it gracefully falls back to treating the entire file as content.
3. **Rendering (Pass 2):** Uses the `goldmark` library to parse Markdown into an AST (Abstract Syntax Tree). 
   * **Link Transformation:** A custom `linkTransformer` traverses the AST. It converts relative `.md` links into `.html` links. It also discovers references to non-existent files to create a graph of "missing files" and backlinks.
   * **Sanitization:** HTML output is run through `bluemonday` to prevent XSS and ensure the markup is safe.
4. **Stub Generation:** Any file linked to, but not present in `content/`, gets a simple HTML stub generated in `public/`. This guarantees there are no dead internal links!
5. **Metadata Output:** Finally, it writes out `graph.json`, `backlinks.json`, and `meta.json` into `public/`.

## The Result

The generator runs quickly and outputs a fully static site to `public/`, complete with deterministic metadata generation!
