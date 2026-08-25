# Black-box Review Pass on v0.1.0-prealpha Artifacts

## Task ID
`issue-500-black-box-review-prealpha`

## Objective
Close #500: review the published pre-alpha archives as a zero-context
operator, triage findings into labeled issues, and check in the durable
review checklist for future cuts.

## Scope executed this pass (2026-08-25)
- darwin/arm64 native full pass against the published release: checksum
  verify, unpack, quickstart flow, author tasks (frontmatter/tags, theme
  default switch, per-page switch, broken links), serve + clean stop,
  publish-check.
- darwin/amd64 verified under Rosetta.
- linux/* and windows/* require runners/VMs not available here; recorded as
  not-executed in the checklist. #500 stays open until those families get a
  pass.

## Findings triaged
- #516 publish-check passes artifacts containing Missing Page stubs silently.
- #517 init scaffold triggers fresh-site description warnings on first check.
- #518 no GitHub Pages deploy guidance for binary-only installs.

## Proposed Changes
- `.github/scripts/release/review-checklist.md`: durable per-platform flow +
  ground rules + v0.1.0-prealpha pass record.

## Potential Static-Output Impact
None — process artifact only.

## Verification
Checklist is documentation; nothing to test beyond link accuracy.

## Handoff
- PR references #500; issue remains open for the linux/windows passes.
