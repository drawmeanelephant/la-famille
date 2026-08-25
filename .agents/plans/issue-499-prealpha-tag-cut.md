# Cut v0.1.0-prealpha Tag and Curated Changelog Snapshot

## Task ID
`issue-499-prealpha-tag-cut`

## Objective
Close #499: first-ever release cut — push `v0.1.0-prealpha`, verify the
release pipeline publishes all archives + SHA256SUMS, write the curated
changelog snapshot, sanity-check README links.

## Blocker found during planning
`.github/scripts/release/tag.sh:canonical_release_tag` enforces strict
`^v[0-9]+(\.[0-9]+){2}$`, which **rejects** `v0.1.0-prealpha` — the exact tag
this milestone requires. Semver permits `-` prerelease suffixes; the pipeline
must accept them before the tag can be pushed.

## Proposed Changes
- `.github/scripts/release/tag.sh`: allow semver prerelease suffix in
  `canonical_release_tag` (`^v[0-9]+(\.[0-9]+){2}(-[0-9A-Za-z.-]+)?$`).
- `.github/scripts/release/test.sh`: cover acceptance of `v0.1.0-prealpha`
  end-to-end (normalize + resolve + provenance) and continued rejection of
  malformed tags.
- After merge + green CI on master:
  - push annotated tag `v0.1.0-prealpha` at master HEAD;
  - watch `release.yml` complete (6 archives + SHA256SUMS);
  - verify checksums locally against the published release;
  - write curated snapshot section in `content/docs/changelog.md` per the
    2026-08-20 convention (#466);
  - sanity-check README links point at the live release page.

## Potential Static-Output Impact
The site deploy workflow runs on pushes to master; the changelog entry changes
the published changelog page only.

## Verification
```bash
bash .github/scripts/release/test.sh
go test ./...
go vet ./...
```

## Handoff
- Status: pipeline fix implemented; tag push follows after merge.
