#!/bin/bash
# Validate that the commands agents are told to run remain present and that
# the repository quality entry points referenced by AGENTS.md exist.
set -euo pipefail

required_commands=(
  'go test ./...'
  'go vet ./...'
  'git config core.hooksPath .githooks'
)

for command in "${required_commands[@]}"; do
  if ! grep -Fq -- "$command" AGENTS.md; then
    echo "AGENTS.md is missing documented command: $command" >&2
    exit 1
  fi
done

test -x .githooks/pre-commit
test -x .github/scripts/quality/check-tech-debt.sh
test -f format_check.sh
echo "AGENTS.md validation passed"
