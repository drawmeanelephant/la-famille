# Task plan: `luna/wiki-links` (#582.1)

1. Add a Goldmark inline parser for `[[target]]`, aliases, and heading
   fragments, then resolve known targets through page paths and titles.
2. Reuse the link transformer, graph, backlinks, and stub pipeline for wiki
   links; label missing wiki targets as unresolved notes.
3. Add a fixture and build tests for known, alias, heading, and unresolved
   targets, including the stub's graph node and inbound edge.
4. Document wiki-link syntax and the generated stub metadata.

**Static-output impact:** Sites using wiki links gain rewritten HTML links,
graph edges, unresolved-note stub pages, and an additive title entry in
`meta.json`. Existing Markdown link behavior and CLI contracts stay unchanged.
