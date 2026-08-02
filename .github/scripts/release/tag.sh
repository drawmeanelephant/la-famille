#!/usr/bin/env bash
# Resolve and verify the release tag for the La Famille release pipeline.
#
# The requested release tag is the authoritative source ref for every release
# job, regardless of which branch a manual workflow_dispatch was launched from.
# This script normalizes the raw input (github.ref_name for tag pushes or
# inputs.tag for manual runs) to the repository's actual "vX.Y.Z" tag form,
# checks that exact tag out, and refuses to proceed unless the checked-out
# commit is the commit referenced by the tag.
#
# This is a repeatable release workflow producing verified release artifacts.
# It does not claim byte-for-byte reproducibility: the embedded build date and
# archive mtimes are still the current time.
set -euo pipefail

# Environment variables exported by resolve_and_checkout_tag:
#   RELEASE_TAG    - canonical tag, e.g. v1.2.3
#   RELEASE_COMMIT - verified commit SHA referenced by that tag.

# canonical_release_tag normalizes INPUT (v1.2.3 or 1.2.3) to the repository's
# v-prefixed tag form. Prints the canonical tag; exits non-zero on empty or
# non-semver input.
canonical_release_tag() {
	local raw="${1:?release tag input is required}"
	local tag
	case "$raw" in
		v*) tag="$raw" ;;
		*) tag="v$raw" ;;
	esac
	if [[ ! "$tag" =~ ^v[0-9]+(\.[0-9]+){2}$ ]]; then
		printf 'invalid release tag %q: expected vX.Y.Z or X.Y.Z\n' "$raw" >&2
		return 1
	fi
	printf '%s\n' "$tag"
}

# resolve_release_tag RAW. Prints the canonical release tag after verifying
# that the tag exists in this repository. Exits non-zero if the tag is missing.
resolve_release_tag() {
	local raw="${1:?release tag input is required}"
	local tag
	tag="$(canonical_release_tag "$raw")"
	if git rev-parse --verify --quiet "refs/tags/$tag^{commit}" >/dev/null; then
		printf '%s\n' "$tag"
	else
		printf 'release tag %s does not exist in this repository\n' "$tag" >&2
		return 1
	fi
}

# require_head_release_tag TAG fails unless the current checked-out commit is
# exactly the commit referenced by the tag. This is the provenance backstop the
# workflow enforces before running any test or build.
require_head_matches_tag() {
	local tag="${1:?release tag required}"
	local expected current
	expected="$(git rev-parse --verify --quiet "refs/tags/$tag^{commit}")" || {
		printf 'release tag %s does not exist in this repository\n' "$tag" >&2
		return 1
	}
	current="$(git rev-parse HEAD)"
	if [[ "$current" != "$expected" ]]; then
		printf 'release tag %s points at %s but the checked-out commit is %s; refusing to release\n' \
			"$tag" "$expected" "$current" >&2
		return 1
	fi
}

# resolve_and_checkout RAW normalizes the input tag, verifies it exists, checks
# out exactly that tag, and verifies the checked-out commit equals the tag's
# commit. Exports RELEASE_TAG and RELEASE_COMMIT. Exits non-zero on any failure
# so an accidental publication can never begin from an invalid ref.
resolve_and_checkout() {
	local raw="${1:?release tag input required}"
	local tag
	tag="$(resolve_release_tag "$raw")"
	git checkout --quiet --detach "$tag"
	require_head_matches_tag "$tag"
	RELEASE_TAG="$tag"
	RELEASE_COMMIT="$(git rev-parse HEAD)"
	export RELEASE_TAG RELEASE_COMMIT
	printf 'resolved release tag %s at commit %s\n' "$RELEASE_TAG" "$RELEASE_COMMIT" >&2
}

main() {
	resolve_and_checkout "${1:?release tag input required}"
	printf 'RELEASE_TAG=%s\nRELEASE_COMMIT=%s\n' "$RELEASE_TAG" "$RELEASE_COMMIT"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
	main "$@"
fi