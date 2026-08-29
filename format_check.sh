#!/bin/bash
# Formatting, vet and dependency hygiene check. Exits non-zero if anything
# drifts, so it is safe for local use and for CI.
#   - gofmt: Go source must be formatted
#   - go vet: suspicious constructs are rejected
#   - go mod tidy -diff: go.mod / go.sum must already be tidy
#   - golangci-lint run: runs when the binary is installed
set -euo pipefail

# 1. gofmt: go fmt rewrites in place; any resulting diff to tracked files means
#    a contributor skipped formatting. --quiet guards against a dirty porcelain
#    caused by unrelated untracked files (e.g. .freebuff/).
echo ">> formatting (go fmt)"
go fmt ./...
if ! git diff --exit-code --quiet; then
  echo "Go files need formatting." >&2
  exit 1
fi

# 2. go vet
echo ">> vet (go vet)"
if ! go vet ./...; then
  echo "go vet found problems." >&2
  exit 1
fi

# 3. go mod tidy: -diff prints what would change and fails when the module is
#    not tidy, without modifying files.
echo ">> module hygiene (go mod tidy -diff)"
if ! go mod tidy -diff; then
  echo "go.mod/go.sum are out of date. Run 'go mod tidy' and commit the result." >&2
  exit 1
fi

# 4. golangci-lint when available
if command -v golangci-lint >/dev/null 2>&1; then
  echo ">> lint (golangci-lint run)"
  if ! golangci-lint run; then
    echo "golangci-lint found problems." >&2
    exit 1
  fi
else
  echo ">> lint (golangci-lint run) SKIPPED: binary not installed"
fi

echo "format check passed."