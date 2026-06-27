---
title: "Shredding the Pipeline (Cyberpunk Layout Showcase)"
layout: cyberpunk
---

## Neon Commits

The Skater shreds through the CI/CD pipeline, moving fast and breaking nothing. Rapid pull requests drop into the codebase like a kickflip down a ten-stair.

### Pipeline Tricks

Here's how The Skater hits the pipeline:

- The Kickflip (Build)
  - `go build`
  - `docker build`
  - Push image
- The Grind (Test)
  1. `go test ./...`
  2. Playwright E2E
  3. Coverage report

> "Move fast, commit often, and don't forget to push your tags. The pipeline is just another rail to grind." — The Skater / The Agilist

### Raoul(s) Roster

| Persona | Responsibility | Associated Image Asset |
|---|---|---|
| The Janitor | Cleaning the repo, clearing branches | `Octopus_mascot_cleaning_litterbox_202606200817.jpeg` |
| The Skater | Shredding the CI/CD pipeline | `Octopus_riding_skateboard_holdin…_202606200817_2.jpeg` |
| The Maestro | Curating Flow Music | `Octopus_mascot_writing_music_dia…_202606200817.jpeg` |

### Rapid Sanitization

When moving fast, security still matters. Here's a quick escape trick for user input:

```go
import "html"

// Quick XSS dodge
func RenderInput(input string) string {
    return html.EscapeString(input) // Note: Need bluemonday for real UGC!
}
```
