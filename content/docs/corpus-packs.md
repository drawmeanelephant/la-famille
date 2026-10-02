---
date: "2026-10-02"
title: "Corpus Packs v1: Build and Verify"
description: "Package existing public corpus artifacts deterministically and verify their integrity without extraction."
---

# Corpus Packs v1: Build and Verify

For member-level updates to existing v1 packs, see
[Local Diff and Apply](corpus-pack-deltas.md). The v1 format below is unchanged.
For subscriber-only acquisition and updates, see
[Pull from a Local Feed](corpus-pack-feeds.md).

This is Phase 1 of [#583](https://github.com/drawmeanelephant/la-famille/issues/583),
tracked by [milestone 6](https://github.com/drawmeanelephant/la-famille/milestone/6)
and subissues [#609](https://github.com/drawmeanelephant/la-famille/issues/609)
and [#610](https://github.com/drawmeanelephant/la-famille/issues/610).
It requires a current source build, not another release or completion of
Change Ledger's advisory/enforcement rollout.

## Commands

Generate the inputs first, then package them:

```bash
la-famille --project-root /path/to/site build
la-famille --project-root /path/to/site rag
la-famille --project-root /path/to/site pack build --output corpus.tar
la-famille --project-root /path/to/site pack verify corpus.tar
```

`pack build --output <pack.tar>` uses the configured `output_dir`, `rag_dir`,
`site_name` and `siteurl`. `--site-output <directory>` and `--rag-dir <directory>`
override the input directories; `--output` (`-o`) is the new archive file.
Relative paths resolve from the selected project root, like other CLI commands.
The destination's parent must exist, and an existing destination is never
overwritten. Keep packs outside the public/RAG inputs.

Packaging only reads existing artifacts. It does not build, export, fetch,
extract, execute, or update them. Refresh both inputs after content changes;
verification establishes integrity, not whether two exports came from the same
build. `pack verify <pack.tar>` does not require usable site configuration.
Successful commands print the member count and content root; any failure exits
nonzero. A changed payload reports its path and expected versus actual SHA256.

## Deliberate content-only allowlist

The builder selects exactly these paths, preserving public-relative paths:

| Input directory | Selected paths | Required? |
| --- | --- | --- |
| Public output | `graph.json`, `backlinks.json`, `search.json`, `meta.json`, `site-manifest.json` | All five |
| Public output | `graph/data.json` (graph explorer's public data, not its HTML/scripts) | If present |
| Public output | `tags/index.html`, `tags/**/index.html`, `categories/index.html`, `categories/**/index.html` | If present |
| RAG directory | `rag-content.md`, stored at archive root as `rag-content.md` | Yes |

Taxonomy indexes currently are generated HTML, not separate JSON artifacts.
Only their `index.html` files enter the pack. Ordinary page HTML, assets, feeds,
sitemap, and Change Ledger reports are not selected. Missing required inputs
fail with a build/RAG refresh hint. Selected files must be regular files;
symlink parents and symlinks in taxonomy trees are rejected. No broad source
or project-directory scan occurs.

**Do not automatically include** `rag-system.md`, `rag-config.md`, source
directories, build caches, workflows, or credentials. This is a deliberate
v1 file-selection boundary, not a claim that user-authored public content has
been scrubbed of secrets. Inspect what you publish. The ordinary `rag` command
still writes all three bundles; packaging chooses only `rag-content.md`.

## Container format

A pack is an **uncompressed POSIX USTAR archive**, built with Go's
`archive/tar`. No dependencies or compression layer are added.

1. The first regular-file member is `pack-manifest.json`.
2. Payload members follow in ascending, bytewise UTF-8 path order.
3. Builder headers use mode `0644`, UID/GID zero, empty owner/group/link names,
   Unix-epoch modification time and no access/change times or extension records.
   Filesystem metadata and packing time do not enter the archive.
4. Member padding is zero. Exactly two zero 512-byte end blocks terminate the
   archive, with no trailing data.

The manifest is UTF-8 JSON, formatted like the repository's JSON writer:
two-space indentation, Go's standard JSON escaping, and a final newline.
Its object field order follows these structures:

```json
{
  "schema_version": 1,
  "site": {
    "name": "Kintsugi & Co. Studio",
    "url": "https://kintsugi.example.com"
  },
  "provenance": {
    "generator": "la-famille",
    "version": "dev",
    "target": "darwin/arm64",
    "go_version": "go1.26.0"
  },
  "members": [
    {
      "path": "backlinks.json",
      "size": 123,
      "sha256": "<64 lowercase hexadecimal characters>"
    }
  ],
  "content_root": "<64 lowercase hexadecimal characters>"
}
```

This illustrates the schema, not valid example hashes. All top-level fields,
both site fields and every member's `path`, `size`, and `sha256` are required.
Site name and producer generator/version must be nonempty. URL may be empty
for sites without a canonical address.

Producer provenance comes from the existing binary identity helper:
generator, version, target, Go version, and release commit/build date when known.
Unknown commit/date fields are omitted. A release build date is fixed binary
metadata, **not** the time of packaging. Provenance identifies the pack producer,
not an inferred source revision or the generator that originally built the
inputs. `site-manifest.json` carries the existing build snapshot as payload.
No source checkout, Git credentials or environment variables are read for
provenance.

### Hashes and content root

Member hashes are lowercase hex SHA256 over exact member bytes. Sizes count
bytes, not characters. The manifest never lists itself.

The content root is lowercase hex SHA256 over the compact Go `encoding/json`
serialization of the sorted `members` array, with field order
`path`, `size`, `sha256`, standard HTML escaping enabled, and **no newline**.
For example, its input shape is:

```text
[{"path":"backlinks.json","size":123,"sha256":"..."}]
```

It commits to paths, sizes and hashes, not site identity or provenance. Identical
input bytes, site identity and producer provenance yield byte-identical packs.
Changes in producer metadata may change the archive without changing its
content root.

## Verification and resource bounds

Verification streams the archive without extraction or execution. It checks
USTAR headers before reading member bodies, rejects extension records and
non-regular entries (directories, symlinks, hard links, devices), and checks
padding and the archive terminator. The manifest must come first; subsequent
members can appear in any order, but every listed member must occur exactly
once and no unlisted archive member may occur.

Paths must be canonical relative UTF-8 slash paths, at most 240 bytes and
encodable in USTAR. Absolute paths, empty paths/components, `.` or `..`
components, repeated/trailing slashes, backslashes, colons, and control
characters are rejected. Duplicates are exact, case-sensitive path duplicates.
The manifest name is reserved. Payload sizes and hashes must match the
manifest, and the content root must match the sorted member records.

Limits apply to both building and verification:

| Resource | Bound |
| --- | --- |
| Manifest bytes | 1 MiB (1,048,576 bytes) |
| Payload member count | 1–4,096, excluding the manifest |
| Individual payload bytes | 64 MiB (67,108,864 bytes) |
| Total payload bytes | 256 MiB (268,435,456 bytes) |
| Entire archive bytes | 272 MiB (285,212,672 bytes) |
| Path length | 240 UTF-8 bytes, also subject to USTAR name/prefix limits |
| JSON nesting | At most 32 container levels |
| Builder taxonomy scan | At most 16,384 filesystem entries across both trees |

Sizes must be nonnegative integers. Manifest JSON rejects invalid UTF-8,
duplicate keys, nulls, unknown fields, missing required fields, trailing values
and incorrect types. Member records must be sorted. The schema version must be
exactly `1`.

**Unknown manifest-listed members are allowed** when paths, sizes, hashes and
the root check pass. This forward-compatible integrity rule is distinct from
the builder's fixed allowlist and from accepting an unsupported schema version.
Verification never interprets a payload's JSON, HTML, Markdown or executable
content. It is not signature verification, publisher authentication or a
guarantee of safe content. Anyone who can rewrite a pack can recompute its
hashes.

## Scope and completion

The fixed milestone criteria cover the artisanal-ceramics round trip,
byte-identical builds, named expected/actual hash failures, malformed-input
and unknown-member regression tests, a compiled-binary demonstration and
repository validation. See the [recorded demonstration](https://github.com/drawmeanelephant/la-famille/blob/master/docs/corpus-packs-v1-demo.md)
in the repository.

Pack diff/apply belongs to the separate
[Local Diff and Apply milestone](corpus-pack-deltas.md). Subscriptions,
pull/watch, remote downloads, pack-backed Ask, signatures, deployment, model
benchmarks and adoption evidence remain **not in scope**. New ideas belong in
separate issues. After review and merge with the
fixed criteria met, close #609, #610 and milestone 6; leave #583 open for later
milestones.
