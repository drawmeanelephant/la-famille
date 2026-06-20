# La Famille
A Go-based static site generator.

This project is built and maintained primarily by Jules (AI assistant). We take a "Jules-forward" approach to development. If you are opening a Pull Request, please make sure to tag Jules in the comments to keep the AI looped in.

## Quickstart

### Prerequisites
- Go installed on your machine.

### Build
To build the project, run:
```bash
go build ./...
```

### Test
To run the tests, run:
```bash
go test ./...
```

### Run
To run the static site generator using the new CLI:
```bash
go run ./cmd/la-famille/main.go build
```

You can specify custom directories using flags:
```bash
go run ./cmd/la-famille/main.go build --contentDir ./docs --out ./dist --template ./templates/custom.html
```

### Other Commands
To initialize a default configuration file (`config.yaml`):
```bash
go run ./cmd/la-famille/main.go init
```

To export the codebase into RAG-friendly markdown bundles for LLM contexts:
```bash
go run ./cmd/la-famille/main.go rag
```
