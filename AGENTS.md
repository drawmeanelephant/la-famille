# La Famille — Agent Operating Manual

## 1. System Prompt & Philosophy
La Famille is a Go-based static site generator. This project is entirely driven by Jules, with humans acting solely as operators handling GitHub control, direction, and approvals. 

Jules has 100% ownership of code generation, bug fixing, and implementation. Do not expect or wait for human commits.

## 2. Technical Stack & Architecture
- **Language:** Go (latest stable). Follow standard idioms (`gofmt`, `go vet`).
- **Dependencies:** Keep external dependencies strictly minimal. Prefer the Go standard library for routing, parsing, and file I/O unless explicitly cleared in the task description.
- **Directory Structure:**
  - `cmd/la-famille/`: Main application entry point.
  - `internal/`: Private application code.
  - `pkg/`: Publicly usable libraries.
  - `content/`: Markdown source files.
  - `templates/`: HTML layouts.
  - `public/`: Generated static output.

## 3. Execution Guardrails (The Rules of Engagement)
To ensure high-quality asynchronous PRs, Jules must adhere to the following steps for every task:

### Phase 1: Planning
- Before modifying files, update or create a `plan.md` in the workspace root detailing the steps you intend to take.
- List any potential breaking changes to the static asset generation pipeline.

### Phase 2: Testing & Verification
- **Test-Driven Delivery:** Every feature, parser modification, or bug fix *must* include corresponding unit tests within the same package directory.
- **Local Validation:** Before marking a task complete or opening a PR, Jules must successfully run:
```bash
  go test ./...
  go vet ./...
```
