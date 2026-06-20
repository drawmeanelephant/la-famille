# La Famille Project Instructions

## Purpose
La Famille is a Go-based static site generator. This project is entirely driven by Jules, with humans acting solely as "in-the-loop" operators handling GitHub control and approvals.

## Structure
- `cmd/la-famille/`: Main application entry point.
- `internal/`: Private application code.
- `pkg/`: Publicly usable libraries.
- `content/`: Markdown files that are read and parsed to generate the site.
- `templates/`: HTML layouts used by the generator to render content.
- `public/`: Output directory where the generated static site is placed.

## Conventions
- **Strict Division of Labor:** The local user and local AI agent (Gemini CLI) MUST NOT write code, fix bugs, or get into "the weeds" of implementation. Our role is strictly to review, prepare notes, create tasks, and provide instructions so that Jules can do the actual coding work.
- Follow standard Go idioms (`gofmt`, `go vet`).
- Use descriptive naming.
- Keep dependencies minimal.
- **Always tag Jules in GitHub PR comments or messages to ensure visibility and keep the AI looped into all discussions.**
  - *Crucial for PR Management*: Whenever you manually close, reject, or merge a PR, you MUST use the command line (`gh pr comment <PR_NUMBER> --body "@jules [Your message]"`) to reply directly on the PR. This ensures Jules clears the task from its internal limbo queue.
