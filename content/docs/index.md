---
date: "2026-10-01"
title: "A field guide to La Famille"
description: "Start a site, find your way around, and understand what the generator actually publishes."
---

## Start with a real page

You don't need Go to use a released binary. You don't need a model, an account,
or a build service to make a site.

1. [Get started](setup.md): download, verify, initialize, write, build, preview.
2. [Write your content](frontmatter.md): titles, dates, taxonomy, and publication.
3. [Make it yours](templates.md): bundled layouts and your own HTML templates.
4. [Publish the artifact](publishing.md): what belongs in `public/`, and what doesn't.

## Find the right tool

| I want to… | Read this |
| --- | --- |
| Set the site name, paths, or canonical address | [Configuration](config.md) |
| Look up a command or flag | [CLI reference](cli.md) |
| Work from the terminal UI | [TUI guide](tui.md) |
| Understand client-side search | [Search](search.md) |
| Export content for other tools | [Corpus exports](rag.md) |
| Try local, citation-grounded questions | [Ask This Site](ask.md), experimental |
| Compare builds and gate regressions | [Change Ledger](change-ledger.md), source-build feature |
| Manage repository pull requests locally | [PR management](pr.md) |

## See the whole system

[How the generator works](generator.md) follows Markdown through the build.
[Architecture](architecture.md) maps the packages. The
[publishing contract](publishing.md) documents HTML, graph/search data, feeds,
and metadata. [How this site ships](site-publishing.md) records the flagship's
canonical domain and Cloudflare workflow.

For real examples, visit [the showcase](../showcase/index.md). For visual
starting points, use [the template gallery](../showcase/templates.md).

## Match the docs to your binary

This website is built from the repository's source. The latest released
pre-alpha binary may not contain every feature described here. Run
`la-famille --version`, then use the command's `--help` to check available flags.
The core quickstart uses the released publishing path; newer graph retrieval,
hybrid retrieval, and Change Ledger work require a current source build.

## Small examples

[Raw Markdown](raw.md) demonstrates passthrough rather than HTML rendering.
The [missing-page example](missing.md) intentionally demonstrates stub
generation; it is not a finished documentation page.
