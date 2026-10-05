---
date: "2026-10-03"
title: "Corpus Packs: Publish, Pull, and Watch"
description: "Publish content-only feeds and safely subscribe over local directories or opt-in HTTPS."
---

# Corpus Packs: Publish, Pull, and Watch

A subscriber needs only the `la-famille` binary and a feed source,
not the publisher's source checkout, configuration, templates, public output,
or a site build. The HTTPS and watch finishing slice is tracked by
[#626](https://github.com/drawmeanelephant/la-famille/issues/626) under
[#583](https://github.com/drawmeanelephant/la-famille/issues/583).
It reuses the existing [v1 pack format](corpus-packs.md) and
[local deltas](corpus-pack-deltas.md). Verified archives remain consumable by
external tools without a built-in assistant or model runtime.

## Subscriber commands

```bash
la-famille pack pull /path/to/feed --output subscriber-v1.tar
la-famille pack pull /path/to/feed --base subscriber-v1.tar --output subscriber-v2.tar
la-famille pack verify subscriber-v2.tar
```

`--output` (`-o`) is required. Its parent directory must exist and its
destination must be new. Existing files, directories, symlinks, and the base
are never overwritten, including destinations created concurrently.
Pull does not perform in-place updates. Base and output arguments are local
paths; a feed is a local directory unless HTTPS is explicitly enabled.
Relative paths resolve against the selected project root (the current
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

Generate the manifest, hashes, canonical pack and available deltas automatically:

```bash
la-famille build
la-famille rag
la-famille pack publish --output /path/to/feed-v1
# After editing and rebuilding/exporting at the publisher:
la-famille pack publish --previous /path/to/feed-v1 \
  --retain 3 --output /path/to/feed-v2
```

The output directory must be new. `--site-output` and `--rag-dir` select the
existing generated inputs. The publisher never scans source or includes
`rag-system.md`, `rag-config.md`, credentials, caches, or arbitrary directories.
Full names are `packs/<archive-sha256>.tar`; delta names include exact base and
target identities. Artifact bytes never change under the same name.

Retention defaults to three full versions and at most two deltas to the current
target (`--retain` accepts 1–8). Prior versions are verified, copied byte-for-byte,
and each generated delta is round-trip checked before advertising it.
The manifest is written last. Repeated publication with identical inputs and
binary provenance is deterministic. Missing history, a missing prior manifest,
or missing retained packs produces a full-only or reduced-history feed;
corrupt available history fails instead of publishing an inconsistent feed.
Subscribers older than retained history use explicit full fallback.

The existing flagship website workflow publishes the entire consistent Pages
artifact, including `/corpus-packs/pack-feed.json`. It restores the last
successfully deployed history from the existing GitHub Actions cache and saves
new history **only after successful production publication**. Cache eviction
(including GitHub's inactivity/quota eviction) is an expected missing-baseline
case, not a reason to fail a cold publication. PR builds cannot save production
history. The production environment approval and existing hosting are unchanged.
The manifest uses `Cache-Control: no-store`; hash-named archives are immutable
cache entries. Versions outside bounded retention can return 404, but their
names are never reused with other bytes.

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

## Explicit HTTPS subscription

```bash
la-famille pack pull https://publisher.example/corpus-packs/pack-feed.json \
  --allow-https --trace-http --output subscriber-v1.tar
la-famille pack pull https://publisher.example/corpus-packs/pack-feed.json \
  --allow-https --trace-http --base subscriber-v1.tar --output subscriber-v2.tar
```

The URL must name `pack-feed.json`. Remote acquisition requires `--allow-https`;
HTTP, URL credentials, queries, fragments and nonstandard ports are rejected.
Members resolve from the original manifest's directory, even after redirects.
Every redirect must stay HTTPS, on the identical origin and within that
directory. Traversal, URL escapes, symlink-like URL tricks, downgrade and
cross-origin redirects fail. HTTPS member names additionally reject `%`, `?`
and `#`; the local path contract is unchanged.

TLS certificates are verified normally. No cookies, authorization headers,
proxy settings or credentials are used. Every DNS result must be public;
loopback, private, link-local, multicast, reserved and IPv6 translation/tunnel
addresses are rejected. Sockets connect directly to a validated address,
preventing DNS re-resolution/rebinding bypasses. The same policy covers
redirects and later connections.

Each request, including body transfer, has a two-minute default deadline
(`--timeout`); dialing and TLS handshake are bounded to ten seconds and
response headers to fifteen seconds. Headers are bounded to 32 KiB.
There are at most three redirects per request and 64 requests per pull/poll.
Uncompressed body bytes retain the feed/archive limits above. Interrupt and
termination signals cancel requests and exit cleanly. Error output does not
include server bodies, corpus text, or credential-bearing URLs.
Optional `--trace-http` records requested URLs, status codes and received body
bytes, including redirects, not TLS framing. A matching delta request replaces
the full target request; there is no claim of page-sized transfer.

## Durable watch

```bash
la-famille pack watch https://publisher.example/corpus-packs/pack-feed.json \
  --allow-https --state /path/to/subscriber --interval 30s --trace-http
# Local directory feeds use the same command without --allow-https.
```

`--state` and a positive `--interval` are required. Watch polls immediately,
then waits the configured interval after each poll. A normal fast publication
is detected at the next poll; transfer duration and errors can delay completion.
Polls do not overlap. Watch needs no usable site configuration.

Verified versions are immutable `packs/<sha256>.tar` files. `current.json`
contains the schema version, exact archive hash and relative path of the current
verified pack. Watch syncs the new archive and version directory, then atomically
replaces and syncs the metadata. The previous verified version is never modified.
Use this pointer to select the current pack for verification or external tools:

```bash
CURRENT=$(python3 -c 'import json; print(json.load(open("/path/to/subscriber/current.json"))["path"])')
la-famille pack verify "/path/to/subscriber/$CURRENT"
```

Watch's unchanged polls fetch only the manifest,
not a full archive or delta, do not rewrite the pointer, and emit no false changes.
Updates report member additions/changes/removals and named pages through the
existing Ledger when both packs support it. Otherwise page semantics are
explicitly unavailable; members remain the transfer unit.

Missing, truncated, corrupt, wrong-base or target-mismatched selected deltas,
interrupted requests and publication races preserve the last verified current
pack. Watch reports failure and retries on the next poll, never silently
falling back for a broken selected delta. A verified orphan left by a process
stopping between archive and pointer publication can be reused on restart.
Corrupt subscriber state fails safely rather than being silently replaced.

One watcher owns each state directory through `.watch-lock`. Clean cancellation
removes the lock. After a crash, confirm that no process still owns the directory
before manually removing a stale lock. Subscriber versions are not automatically
deleted; operators may remove noncurrent versions only when no reader needs them.
The state filesystem must support hard links, atomic file replacement and
directory sync.

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
covers the historical cold pull → delta update → verify workflow, with the
full target unavailable and no subscriber source checkout or build.
Current compiled subscriber tests check the updated archive text and packaged
search/graph/page metadata directly, without a model or assistant.

The [finishing demonstration and operator checklist](https://github.com/drawmeanelephant/la-famille/blob/master/docs/corpus-pack-https-watch-demo.md)
records the historical automated/local evidence and hosted HTTPS checklist.
Ask and real-model answer demonstrations are retired by
[#629](https://github.com/drawmeanelephant/la-famille/issues/629), not remaining
acceptance requirements for Corpus Packs. Ongoing delivery work remains in #583.

**Git-ref sources are deferred, not delivered.** This slice delivers local
directory and opt-in HTTPS subscription only. Signatures/authenticity,
compression, per-page chunks, v2 formats, retrieval benchmarking, new services,
releases and adoption studies remain excluded. Integrity does not establish
publisher authenticity or factual truth.
