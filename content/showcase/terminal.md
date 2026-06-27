---
title: "The Janitor's Terminal (Terminal Layout Showcase)"
layout: layout-terminal
---

## The Terminal of Truth

The Janitor meticulously sweeps the repository, ensuring the "litterbox" of old branch artifacts is cleared out and rogue `serve.log` files are disposed of properly.

### Daily Chores

Here is a list of the daily operations:

- Sweep the floor
  - Delete `serve.log`
  - Remove `.DS_Store`
  - Clear out `/tmp`
- Clean the litterbox
  1. Fetch origin
  2. Prune old branches
  3. Close stale PRs

> "A clean codebase is a happy codebase. Don't leave your `serve.log` lying around, or I'll sweep it into the void." — The Janitor

### Raoul(s) Personas

| Persona | Responsibility | Associated Image Asset |
|---|---|---|
| The Janitor | Cleaning the repo, clearing branches | `Octopus_mascot_cleaning_litterbox_202606200817.jpeg` |
| The Skater | Shredding the CI/CD pipeline | `Octopus_riding_skateboard_holdin…_202606200817_2.jpeg` |
| The Maestro | Curating Flow Music | `Octopus_mascot_writing_music_dia…_202606200817.jpeg` |

### Security Protocol

The Janitor always ensures paths are sanitized to prevent directory traversal vulnerabilities during sweeping:

```go
// Prevent directory traversal when sweeping logs
func SweepLog(logPath string) error {
    if !filepath.IsLocal(filepath.FromSlash(logPath)) {
        return fmt.Errorf("invalid path: %s", logPath)
    }
    return os.Remove(logPath)
}
```
