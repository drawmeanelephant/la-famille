# La Famille Project Instructions

## Purpose
La Famille is a Go-based static site generator. Antigravity and AI coding agents work directly alongside human operators on feature implementation, bug fixing, code review, and PR management.

## Structure
- `cmd/la-famille/`: Main application entry point.
- `internal/`: Private application code.
- `pkg/`: Publicly usable libraries.
- `content/`: Markdown files that are read and parsed to generate the site.
- `templates/`: HTML layouts used by the generator to render content.
- `public/`: Output directory where the generated static site is placed.

## Conventions
- **Direct Implementation:** AI agents can directly write code, resolve merge conflicts, fix bugs, run local tests, and manage PRs.
- Follow standard Go idioms (`gofmt`, `go vet`).
- Use descriptive naming.
- Keep dependencies minimal.
- **GitHub & PR Management:** Maintain clean git workflows. Run local validation (`go test ./...`, `go vet ./...`) before committing or pushing.
