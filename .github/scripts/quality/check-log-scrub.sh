#!/bin/bash
# Static guard against accidentally logging Ask request bodies or questions.
set -euo pipefail

if git grep -nE 'slog\.(Debug|Info|Warn|Error).*([Qq]uestion|[Pp]rompt|[Bb]ody)' -- internal/ask '*.go'; then
  echo "Ask logging must not include request bodies, questions, or prompts" >&2
  exit 1
fi

grep -Fq 'scrubLogText' internal/ask/server.go
echo "Ask log-scrubbing guard passed"
