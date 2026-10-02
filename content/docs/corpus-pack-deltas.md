---
date: "2026-10-02"
title: "Corpus Packs: Local Diff and Apply"
description: "Deterministic member-level deltas and verified local reconstruction of v1 corpus packs."
---

# Corpus Packs: Local Diff and Apply

This bounded milestone under [#583](https://github.com/drawmeanelephant/la-famille/issues/583)
builds on merged v1 [#612](https://github.com/drawmeanelephant/la-famille/pull/612).
Tracking: [milestone 7](https://github.com/drawmeanelephant/la-famille/milestone/7),
[#613](https://github.com/drawmeanelephant/la-famille/issues/613) and
[#614](https://github.com/drawmeanelephant/la-famille/issues/614).
The [full pack v1 format](corpus-packs.md) and build/verify commands are unchanged.

## Commands and member boundary

```bash
la-famille pack diff before.tar after.tar --output delta.tar
la-famille pack diff before.tar after.tar --output delta-json.tar --json
la-famille pack apply before.tar delta.tar --output result.tar
la-famille pack verify result.tar
```

All paths are local. Relative paths resolve against `--project-root`.
`--output` (`-o`) is required and its parent directory must already exist.
Every destination must be new, including for diff. Verify, diff, and apply work
without usable site configuration. They never regenerate inputs, extract
payloads, execute content, or fetch anything.

**The unit of change is an archive member, not a page.** V1 stores
`rag-content.md` as one member. Editing one page may replace that entire corpus
member along with derived graph/search/site-manifest members. There is no
per-page chunk storage or promise of page-sized downloads.

The readable report lists `+` added, `-` removed and `~` changed paths.
JSON has `base_sha256`, `target_content_root`, sorted `added`, `removed` and
`changed` arrays, and an optional `ledger` report. Equal size **and** SHA256 means
unchanged. Site/provenance-only changes still update the target manifest even
when all three member arrays are empty.

When both packs contain supported `site-manifest.json` payloads, the report
reuses Change Ledger's existing comparison primitives, including graph-edge
and taxonomy membership changes. These semantic changes are informational,
not additional delta payloads or a Ledger gate. Missing, unsupported, malformed
or too-large site manifests produce `ledger_warning`; opaque member comparison
and application still work. Semantic reporting is bounded to 1 MiB and 4,096
pages per site manifest. Existing Ledger coverage warnings remain visible.
Graph/taxonomy members are still replaced whole when their bytes change.

## Delta schema v1

A delta is a separate **uncompressed regular-file USTAR container**, not a full
pack accepted by `pack verify`. Apply verifies the delta internally. Header
format, zero padding and exactly two zero end blocks follow full pack v1.
Canonical delta headers use the v1 builder's mode, ownership and epoch time.

Members, in canonical creation order:

1. `pack-delta.json`: versioned UTF-8 metadata, at most 1 MiB.
2. `pack-manifest.json`: the **complete, exact target full-pack manifest bytes**,
   including its site identity, provenance, sorted inventory and content root.
3. `payload/0000`, `payload/0001`, …: only added/changed target payloads.

Metadata fields, in canonical field order:

```json
{
  "schema_version": 1,
  "base_sha256": "<SHA256 of the entire exact base archive>",
  "target_manifest_sha256": "<SHA256 of exact target manifest bytes>",
  "added": ["new/member.bin"],
  "changed": ["rag-content.md"],
  "removed": ["obsolete/member.bin"]
}
```

This illustrates the shape, not valid hashes. Metadata uses two-space Go
`encoding/json` indentation, standard escaping and a final newline. Every field
is required, including empty arrays. Paths in each array are strictly sorted
bytewise, unique and disjoint across arrays. Hashes are 64 lowercase hex digits.
Schema versions other than `1` are rejected.

Payload indices are zero-based, four-digit decimal indices into the **sorted
union of `added` and `changed` paths**. The target manifest supplies each
payload's original path, size and SHA256. Numbered names avoid collisions with
verified unknown v1 paths, including `pack-delta.json` or `payload/0000`, without
introducing a path prefix that could exceed v1's path-length limit. Readers
require metadata then manifest; payload order can vary, but every indexed
payload must occur exactly once. Missing, duplicate, unlisted, unsafe,
oversized or corrupt entries fail. Removed paths must be absent from the target.
No unchanged payloads, graph patches, source files or page chunks are added.

## Exact-base binding and safe application

`base_sha256` hashes every byte of the verified base archive, including its
manifest, headers, payloads, padding and terminator. The v1 `content_root`
alone is insufficient: it excludes site identity/provenance and tar headers.
A different base with the same content root is rejected. This is integrity
binding, not signatures or publisher authentication.

Both inputs are copied and verified in the same bounded stream into private
opaque archive snapshots. Payloads are **not extracted**. Open descriptors and
verified offsets retain the bytes consumed; original input paths are never
reopened after verification. Each payload is hashed again as it is written.
Unchanged members, including verified unknown members, come from the base.
Added/changed members come from the delta. Explicitly removed members are omitted.
Apply recomputes the change arrays from the base and target manifests and
requires exact agreement with the metadata.

The reconstructed full pack uses the retained target manifest and v1 canonical
headers with sorted target members. It is written to a private temporary file
beside the destination, fully verified through the same descriptor, synced and
closed, then published with an exclusive hard link. A concurrent destination,
existing file, directory or symlink is never overwritten. Errors clean up
temporary files; invalid input leaves no completed destination and never
modifies the base. Diff uses the same verified, no-overwrite publication rule.
The destination filesystem must support hard links; unsupported filesystems
fail without falling back to an overwriting rename.

All v1 path and payload bounds remain: 240-byte relative UTF-8 paths, at most
4,096 target members, 64 MiB per member, 256 MiB total payload and 272 MiB per
archive. Both JSON members have independent 1 MiB limits and 32-level nesting
limits; malformed JSON, nulls, repeated/unknown/missing fields and unsupported
schemas fail. Delta payload count may be zero; added plus changed is at most
4,096, and removed is at most 4,096. Snapshots bound disk use by the input archive
limits; reconstruction is bounded by the target manifest. There is no
compression, extension record handling, network access or execution.

## Byte-identity contract and proof

**Applying a generated delta between canonical v1 builder packs produces a
full pack byte-identical to the target builder pack.** The target manifest bytes
are retained exactly; the same canonical header writer, sorted payload order,
payload bytes, zero padding and end blocks reconstruct the target container.
Repeated diff from identical input archive bytes produces byte-identical deltas,
independent of destination names or packing time.

The verifier also accepts some alternate regular USTAR headers and payload
orders. Apply canonicalizes those layouts. **It does not preserve arbitrary
alternate tar-header layouts or promise byte identity to a noncanonical target.**
It preserves the verified target manifest and payload bytes. Integrity hashes
do not establish authenticity or content safety.

The same-package artisanal-ceramics fixture/golden test edits one page, changes
one taxonomy term and adds a graph edge. It asserts full target byte identity,
repeated delta identity and the precise member payload count. Other tests cover
no-change, additions, modifications, removals, unknown members, wrong bases,
corruption, truncation, unsafe paths, size limits and destination protection.
See the [compiled-binary demonstration](https://github.com/drawmeanelephant/la-famille/blob/master/docs/corpus-pack-deltas-demo.md).

Ask integration, pull/watch, subscriptions, remote access, signatures,
compression, deployments, releases and per-page repackaging are not part of
this milestone. After review and merge, close #613, #614 and milestone 7.
Keep #583 open for later milestones. New ideas belong in separate issues, not
new acceptance criteria; Ledger rollout policy is unchanged.
