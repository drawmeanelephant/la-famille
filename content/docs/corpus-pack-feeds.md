---
date: "2026-10-02"
title: "Corpus Packs: Pull from a Local Feed"
description: "Obtain a verified corpus from a local directory and update a new pack by an exact-base delta."
---

# Corpus Packs: Pull from a Local Feed

A subscriber needs only the `la-famille` binary and a local feed directory,
not the publisher's source checkout, configuration, templates, public output,
or a site build. This bounded milestone under
[#583](https://github.com/drawmeanelephant/la-famille/issues/583) is tracked by
[milestone 9](https://github.com/drawmeanelephant/la-famille/milestone/9),
[#623](https://github.com/drawmeanelephant/la-famille/issues/623) and
[#624](https://github.com/drawmeanelephant/la-famille/issues/624).
It reuses the existing [v1 pack format](corpus-packs.md),
[local deltas](corpus-pack-deltas.md), and [pack-backed Ask](ask.md).

## Subscriber commands

```bash
la-famille pack pull /path/to/feed --output subscriber-v1.tar
la-famille pack pull /path/to/feed --base subscriber-v1.tar --output subscriber-v2.tar
la-famille pack verify subscriber-v2.tar
la-famille ask --pack /absolute/path/subscriber-v2.tar --provider fake --no-browser
```

`--output` (`-o`) is required. Its parent directory must exist and its
destination must be new. Existing files, directories, symlinks, and the base
are never overwritten, including destinations created concurrently.
In-place updates are deferred. Feed, base and output arguments are local paths;
relative arguments resolve against the selected project root (the current
directory by default). Pull needs no usable site configuration.

Without `--base`, pull verifies and copies the current full archive
**byte-for-byte**, including its original verified headers. With `--base`,
it first verifies and snapshots that archive, then selects a delta by the
SHA256 of **every exact base archive byte**, not its content root.
Only the selected delta's payload is read, not the full target's payload.
The full target may be absent during a delta update.

If no delta matches, successful pull explicitly reports:

```text
No matching delta for the exact base SHA256; using full-pack fallback.
```

The fallback verifies and copies the full pack. A selected delta that is
missing, corrupt, bound to the wrong base, has inconsistent change metadata,
or reconstructs a different target **fails without a full-pack fallback**.
A corrupt base also fails rather than falling back.

The readable result identifies `mode full` or `mode delta`, the exact target
archive SHA256, and the content root. When a base is provided, it lists `+`
added, `-` removed and `~` changed members, with the existing optional Change
Ledger comparison when available. These are **member changes, not page-sized
updates**; `rag-content.md` remains one whole member.

## Feed manifest v1

Place a regular UTF-8 JSON file named `pack-feed.json` at the feed root:

```json
{
  "schema_version": 1,
  "full": {
    "path": "packs/current.tar",
    "sha256": "<64 lowercase hex digits: exact current full archive SHA256>"
  },
  "deltas": [
    {
      "path": "deltas/from-previous.tar",
      "base_sha256": "<64 lowercase hex digits: exact previous full archive SHA256>"
    }
  ]
}
```

This illustrates the schema, not valid example hashes. All fields are required,
including an empty `deltas: []` when no updates are available. Every delta must
produce the one advertised current target. Delta metadata already commits to
its base and target manifest; pull additionally checks the reconstructed
**archive** digest against `full.sha256`. The v1 feed does not require a
separate delta archive digest.

Prepare packs and deltas at the publisher using existing commands:

```bash
la-famille pack build --output /path/to/feed/packs/current.tar
la-famille pack diff previous.tar /path/to/feed/packs/current.tar \
  --output /path/to/feed/deltas/from-previous.tar
```

The inputs must already have been built/exported at the publisher. Create the
feed JSON yourself using archive SHA256 values (for example,
`shasum -a 256 <archive>`); there is no feed publishing command or automatic
publisher. Publish a consistent manifest and its artifacts before pulling.

**Canonical v1 builder packs are the supported delta round-trip source.**
Delta application retains target manifest/payload bytes but writes canonical
v1 headers and member order. An alternate verified USTAR target layout may
be copied on a cold/full pull, but its generated delta cannot promise the
same archive bytes. Pull rejects that update if its reconstructed archive
digest differs from the advertised target; it does not hide the failure
with a fallback. Verified unknown members remain supported.

### Bounds and path rules

| Resource | Limit |
| --- | --- |
| Feed JSON bytes | 64 KiB (65,536 bytes), including whitespace |
| Available deltas | 0–128 entries |
| Each source path | 240 UTF-8 bytes |
| JSON nesting | At most 32 container levels |
| Full pack or delta archive | Existing 272 MiB archive limit |
| Pack/delta members and payload sizes | Existing v1 limits, unchanged |

The manifest rejects unsupported versions, invalid UTF-8, malformed JSON,
trailing values, duplicate keys, nulls, incorrect types, unknown fields,
case aliases, missing fields, malformed hashes, duplicate exact paths,
and repeated base identities (which would make selection ambiguous).
`pack-feed.json` is a reserved source path.

All manifest paths are canonical relative slash paths within the feed.
Absolute paths, URLs, `.` or `..` components, escaping paths, empty components,
repeated/trailing slashes, backslashes, colons, and control characters fail.
The chosen feed directory itself must not be a symlink. Source traversal uses
confined, pinned directory handles and rejects symlink parents and leaf
symlinks, including links whose target remains inside the feed.
The caller chooses the feed directory's location; ancestor directories outside
that boundary are not part of the manifest path contract.

All declared paths are validated, and existing declared sources must be
regular and link-free, even when unselected. Missing unselected archives
are allowed. Only selected archives must exist and are opened/read; checking
unselected paths inspects filesystem metadata, not archive payloads.

## Verification and publication

Full/base/delta verification captures the bytes into private, bounded opaque
snapshots. Copies and delta reconstruction consume those verified bytes, never
blindly reopening their original paths. Members are not extracted or executed.
Delta application checks the exact base binding and the complete change
inventory, then preserves unchanged verified members and copies replacements
from the verified delta.

The new archive is written beside the destination, verified and hashed through
the same descriptor, and synced before exclusive hard-link publication.
The resulting digest must equal `full.sha256`. Failures remove staging files,
leave no completed new output, and leave the base unchanged. The destination
filesystem must support hard links; there is no overwriting-rename fallback.

Hashes establish integrity, **not publisher authenticity, signatures, or safe
content**. Someone who can replace the feed can advertise different hashes.
Trust the feed source and inspect what you publish.

## Evidence and boundary

The [compiled subscriber demonstration](https://github.com/drawmeanelephant/la-famille/blob/master/docs/corpus-pack-feed-demo.md)
covers cold pull → delta update → verify → fake-provider Ask, with the full
target unavailable and no subscriber source checkout or build.
The fake provider's prose is synthetic; the updated fact is checked in the
retrieved citation excerpt.

HTTPS, Git-ref sources, timers/watch, automatic publishing, signatures,
compression, per-page chunks, in-place updates, deployments, releases,
model benchmarks, and adoption evidence are not included. After the reviewed
PR merges, close #623, #624 and milestone 9. Leave #583 open for later
transport/watch work. New ideas belong in separate issues, not extra closure
criteria.
