#!/bin/bash
# Run the complete Go suite and enforce a repository-wide statement coverage
# floor. The profile is written outside the checkout by default.
set -euo pipefail

threshold="${COVERAGE_THRESHOLD:-75.0}"
profile="${COVERAGE_PROFILE:-${RUNNER_TEMP:-/tmp}/la-famille-coverage.out}"
trap 'rm -f "$profile"' EXIT

go test ./... -coverprofile="$profile"
total="$(go tool cover -func="$profile" | awk '/^total:/ {print $3}')"
if [ -z "$total" ]; then
  echo "coverage report did not contain a total percentage" >&2
  exit 1
fi

percentage="${total%\%}"
printf 'coverage: %s (required: %s%%)\n' "$total" "$threshold"
if ! awk -v actual="$percentage" -v required="$threshold" 'BEGIN { exit !(actual >= required) }'; then
  echo "coverage is below the configured threshold" >&2
  exit 1
fi

if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
  {
    echo "## Go coverage"
    printf -- '- Total statement coverage: **%s**\n' "$total"
    printf -- '- Required threshold: **%s%%**\n' "$threshold"
  } >> "$GITHUB_STEP_SUMMARY"
fi
