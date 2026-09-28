#!/bin/bash
# Measure the compiler build and publish a stable metric for CI artifacts and
# the Actions summary. Go's cache is configured by the calling workflow.
set -euo pipefail

start_seconds="$SECONDS"
go build ./...
elapsed_seconds=$((SECONDS - start_seconds))
metric="go_build_seconds ${elapsed_seconds}"
echo "$metric"

if [ -n "${BUILD_METRICS_FILE:-}" ]; then
  printf '%s\n' "$metric" > "$BUILD_METRICS_FILE"
fi

if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
  {
    echo "## Go build performance"
    printf -- '- Build duration: **%ss**\n' "$elapsed_seconds"
  } >> "$GITHUB_STEP_SUMMARY"
fi
