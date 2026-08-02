#!/usr/bin/env bash
# Test harness for .github/scripts/release/tag.sh.
#
# Builds a scratch Git repository and validates the normalization and release
# provenance rules that .github/workflows/release.yml depends on.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TAG_SCRIPT="$SCRIPT_DIR/tag.sh"

TEST_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/la-famille-release-test.XXXXXX")"
trap 'rm -rf "$TEST_ROOT"' EXIT

pass=0
fail() {
	printf 'FAIL: %s\n' "$*" >&2
	exit 1
}
note() {
	printf 'ok - %s\n' "$*"
	pass=$((pass + 1))
}

# make_repo prepares a scratch repository with two commits:
#   TAGGED_COMMIT - tagged v1.2.3
#   TIP_COMMIT    - a later commit that the tag does not reference.
make_repo() {
	git init -q "$TEST_ROOT/repo"
	(
		cd "$TEST_ROOT/repo"
		git config user.name "release test"
		git config user.email "release@example.com"
		printf 'tagged\n' > file.txt
		git add file.txt
		git commit -q -m "tagged commit"
		echo "TAGGED_COMMIT=$(git rev-parse HEAD)"
		git tag -m "v1.2.3" v1.2.3
		printf 'after tag\n' > tip.txt
		git add tip.txt
		git commit -q -m "tip after tag"
		echo "TIP_COMMIT=$(git rev-parse HEAD)"
	) > "$TEST_ROOT/vars"
	# shellcheck disable=SC1091
	. "$TEST_ROOT/vars"
}

# run_tag invokes the helper as a CLI from inside the scratch repo.
run_tag() {
	local raw="$1"
	(
		cd "$TEST_ROOT/repo"
		bash "$TAG_SCRIPT" "$raw" 2>/dev/null
	)
}

# case_tag_push simulates a tag push: the canonical tag is github.ref_name and
# the checked-out commit must be the tag commit.
expect_tag_push() {
	local out tag commit
	out="$(run_tag "v1.2.3")"
	tag="$(printf '%s\n' "$out" | sed -n 's/^RELEASE_TAG=//p')"
	commit="$(printf '%s\n' "$out" | sed -n 's/^RELEASE_COMMIT=//p')"
	local current
	current="$(git -C "$TEST_ROOT/repo" rev-parse HEAD)"
	[[ "$tag" == "v1.2.3" ]] || fail "tag push: got RELEASE_TAG=%s" "$tag"
	[[ "$commit" == "$TAGGED_COMMIT" ]] || fail "tag push: got RELEASE_COMMIT=%s want %s" "$commit" "$TAGGED_COMMIT"
	[[ "$current" == "$TAGGED_COMMIT" ]] || fail "tag push: HEAD is %s, want %s" "$current" "$TAGGED_COMMIT"
	note "tag push for v1.2.3 checks out and verifies the tag at $TAGGED_COMMIT"
}

# case_manual normalizes a manual workflow_dispatch input and insists the
# embedded commit equals the checked-out (tag) commit, not the dispatch HEAD.
expect_manual() {
	local raw="$1"
	local out tag commit
	out="$(run_tag "$raw")"
	tag="$(printf '%s\n' "$out" | sed -n 's/^RELEASE_TAG=//p')"
	commit="$(printf '%s\n' "$out" | sed -n 's/^RELEASE_COMMIT=//p')"
	local current tag_commit
	current="$(git -C "$TEST_ROOT/repo" rev-parse HEAD)"
	tag_commit="$(git -C "$TEST_ROOT/repo" rev-parse "refs/tags/v1.2.3^{commit}")"
	[[ "$tag" == "v1.2.3" ]] || fail "manual %s: got RELEASE_TAG=%s" "$raw" "$tag"
	[[ "$commit" == "$tag_commit" ]] || fail "manual %s: got RELEASE_COMMIT=%s want %s" "$raw" "$commit" "$tag_commit"
	[[ "$current" == "$tag_commit" ]] || fail "manual %s: HEAD is %s, want tag commit %s" "$raw" "$current" "$tag_commit"
	note "manual input $raw resolves to $tag and embeds the verified commit"
}

# expect_fail asserts resolution fails (nonexistent or malformed tag).
expect_fail() {
	local raw="$1"
	if out="$(run_tag "$raw")"; then
		fail "expected release resolution to fail for %s, got: %s" "$raw" "$out"
	fi
	note "release resolution rejected $raw"
}

make_repo
expect_tag_push
expect_manual "v1.2.3"
expect_manual "1.2.3"
expect_fail "v9.9.9"
expect_fail "9.9.9"

# The checkout line backstop: if a build ran from a commit that does not match
# the tag it must be rejected even when the resolution wrapper is not used.
(
	cd "$TEST_ROOT/repo"
	git checkout -q --detach "$TIP_COMMIT"
	# shellcheck source=tag.sh disable=SC1091
	if . "$TAG_SCRIPT" && require_head_matches_tag v1.2.3; then
		fail "mismatched checkout was not rejected"
	fi
) >/dev/null
note "a non-matching checked-out commit is rejected"

# The release metadata commit embedded in the binary must equal git rev-parse
# HEAD of the verified tag checkout.
release_commit="$(printf '%s\n' "$(run_tag "1.2.3")" | sed -n 's/^RELEASE_COMMIT=//p')"
head_after="$(git -C "$TEST_ROOT/repo" rev-parse HEAD)"
[[ "$release_commit" == "$head_after" ]] || fail "metadata commit %s != HEAD %s" "$release_commit" "$head_after"
note "release metadata commit equals git rev-parse HEAD"

printf '%d checks passed\n' "$pass"