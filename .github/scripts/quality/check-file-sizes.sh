#!/bin/bash
# Keep tracked source and configuration files small enough for agents to
# inspect and review in focused chunks.
set -euo pipefail

max_lines="${MAX_FILE_LINES:-1500}"
files="$(
  git ls-files -- \
    '*.go' '*.sh' '*.yml' '*.yaml' '*.json' '*.html' '*.js' '*.css'
)"
violations=0

while IFS= read -r file; do
  [ -n "$file" ] || continue
  lines=$(wc -l < "$file")
  if [ "$lines" -gt "$max_lines" ]; then
    printf 'file exceeds %s-line limit: %s (%s lines)\n' "$max_lines" "$file" "$lines" >&2
    violations=$((violations + 1))
  fi
done <<< "$files"

if [ "$violations" -ne 0 ]; then
  exit 1
fi

echo "file-size check passed: maximum ${max_lines} lines"
