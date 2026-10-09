# Plan: #637 — publish artifact mixes 0600 and 0644 file modes

## Bug
Pages and copied assets land in `public/` at 0644, but feed.xml, sitemap.xml,
robots.txt, graph.json, meta.json, backlinks.json, site-manifest.json,
diff.json/diff.txt, graph/index.html, graph/data.json, raw render:false .md
copies, and embedded-fallback assets are written 0600 — unreadable to a
web-server process running as another user.

## Fix
Switch every write that lands inside the output tree to 0644 (subject to the
process umask, matching how os.Create pages/assets already behave):

- `internal/jsonutil/write.go` — shared JSON writer (graph.json,
  backlinks.json, meta.json, site-manifest.json, diff.json, graph/data.json)
- `internal/discovery/write.go` — sitemap.xml, robots.txt
- `internal/diff/write.go` — diff.txt
- `internal/feed/write.go` — feed.xml
- `internal/generator/generator.go` — verbatim render:false markdown copy
- `internal/generator/vault_navigation.go` — backlinks-panel rewrite
- `internal/asset/copy.go` — embedded fallback assets
- `internal/graphexplorer/graphexplorer.go` — graph/index.html

Deliberately unchanged: `.la-famille-cache.json` (0600; lives beside the
project, rejected from the artifact by publisher.Check), diff archive temp
extraction, pack tar creation, log files, config writes.

## Tests
- `internal/generator`: build a fixture site exercising a dated page (feed),
  a tag (taxonomy), a dangling link (stub) and a render:false page, then walk
  `public/` asserting every file's perm equals a control file created 0644 in
  the same tree — uniform modes regardless of the runner's umask.

## Breaking changes
- None. Artifact file modes relax from owner-only to world-readable.
