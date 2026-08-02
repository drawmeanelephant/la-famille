# Release Provenance Fix (PR #447 branch)

## Task ID
`release-447`

## Goal
Fix the release-integrity problems in `.github/workflows/release.yml` so every
release job treats the requested release tag as the authoritative source ref,
and stop claiming byte-for-byte reproducibility the pipeline does not provide.

## Failure mode

A manual `workflow_dispatch` run currently:

- checks out the branch/ref from which the workflow was dispatched;
- labels the resulting archive with `inputs.tag`;
- embeds `$GITHUB_SHA` (the dispatch branch tip) as `main.buildCommit`;
- calls `gh release create "$TAG" --verify-tag`.

`--verify-tag` only proves the tag exists. If a maintainer dispatches on the
default branch while `inputs.tag` names an existing older tag, the workflow
can publish binaries built from the wrong commit under that tag.

## Fix

1. Resolve the release tag once per job:
   - tag-push runs use `github.ref_name`;
   - manual runs use `inputs.tag`;
   - accept `v1.2.3` or `1.2.3`, always normalized to the repository's actual
     `v1.2.3` form.
2. `actions/checkout` fetches full history and tags (`fetch-depth: 0`,
   `fetch-tags: true`) in both the test and package jobs.
3. A shared helper (`.github/scripts/release/tag.sh`) resolves the canonical
   tag, verifies it exists, checks out exactly that tag, and rejects any run
   where `git rev-parse HEAD` != `git rev-list -n 1 "$TAG"`.
4. `main.buildCommit`, release diagnostics, and any other commit metadata use
   the verified checked-out commit, never the workflow event SHA.
5. `gh release create --verify-tag` stays as a final backstop, not the only
   provenance check.
6. Keep the six OS/architecture targets and the tar.gz + `SHA256SUMS`
   contract.

## Reproducibility wording

`.github/scripts/release/tag.sh` and the workflow describe the pipeline as a
repeatable release workflow producing verified release artifacts. Nothing
claims byte-for-byte reproducibility (build timestamps and archive mtimes are
the current time); the earlier "reproducible" wording in the hardening plan is
corrected to "repeatable, source-independent".

## Tests / validation

`.github/scripts/release/test.sh` exercises the helper against a scratch Git
repository and validates:

- tag push for `v1.2.3`;
- manual input `v1.2.3`;
- manual input `1.2.3`;
- nonexistent tag;
- checkout commit does not match tag;
- released commit equals `git rev-parse HEAD`.

Local validation:

```bash
bash .github/scripts/release/test.sh
go test ./...
go test -race ./...
go vet ./...
golangci-lint run
shellcheck .github/scripts/release/tag.sh
```

plus YAML syntax validation of the edited workflow files.

## Breaking changes

- None to the static build pipeline. Manual dispatches for a tag that does not
  exist now fail in the test job instead of publishing an archive under an
  existing tag that was never built from its commit.