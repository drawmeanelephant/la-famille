#!/bin/bash
# Formatting, vet and dependency hygiene check. Exits non-zero if anything
# drifts, so it is safe for local use, pre-commit hooks, and CI.
#   - gofmt: Go source must be formatted
#   - go vet: suspicious constructs are rejected
#   - go mod tidy -diff: go.mod / go.sum must already be tidy
#   - technical-debt markers: TODO/FIXME/HACK/XXX must link to an issue
#   - golangci-lint run: runs when the binary is installed
set -euo pipefail
repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

# 1. gofmt: list unformatted Go files without rewriting the working tree. This
#    keeps the check safe when a contributor has unrelated local changes.
echo ">> formatting (go fmt)"
unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then
  printf '%s\n' "$unformatted"
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

# 4. technical-debt markers
echo ">> technical-debt markers"
bash "$repo_root/.github/scripts/quality/check-tech-debt.sh"

# 5. golangci-lint when available
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