---
name: go-quality
description: Validate La Famille Go changes with deterministic formatting, tests, vet, module hygiene, and repository quality checks.
---

# Go quality workflow

Use this workflow after changing Go code, templates, CI, or agent
instructions:

1. Run `./format_check.sh`.
2. Run `go test ./...`.
3. Run `go test -race ./...` for concurrency or filesystem changes.
4. Run `go vet ./...`.
5. Run `go test -shuffle=on -count=2 -parallel=4 ./...` when changing tests
   or shared state.
6. Check generated static output contracts when changing the generator,
   templates, content parsing, or assets.

Keep technical-debt markers linked to an issue:
`TODO(#123):`, `FIXME(PROJ-123):`, `HACK(#123):`, or `XXX(PROJ-123):`.
