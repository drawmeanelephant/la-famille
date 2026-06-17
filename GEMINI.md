# La Famille Project Instructions

## Purpose
La Famille is a Go-based static site generator. This project is built using a highly AI-collaborative workflow with Jules taking a central role.

## Structure
- `cmd/la-famille/`: Main application entry point.
- `internal/`: Private application code.
- `pkg/`: Publicly usable libraries.
- `content/`: Markdown files that are read and parsed to generate the site.
- `templates/`: HTML layouts used by the generator to render content.
- `public/`: Output directory where the generated static site is placed.

## Conventions
- Follow standard Go idioms (`gofmt`, `go vet`).
- Use descriptive naming.
- Keep dependencies minimal.
- **Always tag `@jules` in GitHub PR comments or messages to ensure visibility and keep the AI looped into all discussions.**
