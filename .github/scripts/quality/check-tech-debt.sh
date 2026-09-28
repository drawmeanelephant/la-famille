#!/bin/bash
# Reject untracked technical-debt markers in source and configuration files.
# Every marker must point to an issue so agents and maintainers can find the
# owner and exit criteria for the deferred work.
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

marker_pattern='(^|[^[:alnum:]_])(TODO|FIXME|HACK|XXX)[[:space:]]*[\(:]'
issue_pattern='(^|[^[:alnum:]_])(TODO|FIXME|HACK|XXX)[[:space:]]*\((#[0-9]+|[A-Za-z][A-Za-z0-9_-]*-[0-9]+|https://github\.com/[^[:space:])]+/issues/[0-9]+)\)'

markers="$(
  git grep -nI -E "$marker_pattern" -- \
    '*.go' '*.sh' '*.yml' '*.yaml' '*.json' '*.html' '*.js' '*.css' \
    ':(exclude).github/scripts/quality/check-tech-debt.sh' || true
)"

if [ -z "$markers" ]; then
  echo "technical-debt marker check passed: no markers found"
  exit 0
fi

unlinked="$(printf '%s\n' "$markers" | grep -Ev "$issue_pattern" || true)"
if [ -n "$unlinked" ]; then
  echo "Unlinked technical-debt markers found. Use TODO(#123):, FIXME(PROJ-123):, HACK(#123):, or XXX(PROJ-123):." >&2
  printf '%s\n' "$unlinked" >&2
  exit 1
fi

echo "technical-debt marker check passed: all markers link to issues"
