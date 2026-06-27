---
title: "Synthwave Beats (Glassmorphism Layout Showcase)"
layout: layout-glassmorphism
---

## Glass Beats

The Maestro curates Flow Music behind the frosted glass of this layout, spinning up synthwave and 90s boom-bap tracks to keep the coding sessions smooth. The `content/soundtrack/` directory is his domain.

### Flow State Tracks

The ultimate playlist for deep work:

- Ambient
  - Rain on the Window
  - Deep Sea Drone
- Upbeat
  1. 80s Workout Tape
  2. High-Speed Chase
  3. Retro Arcade

> "A 90s boom-bap beat is like a solid unit test. It just makes everything else fall into rhythm." — The Maestro

### Persona Roster

| Persona | Responsibility | Associated Image Asset |
|---|---|---|
| The Janitor | Cleaning the repo, clearing branches | `Octopus_mascot_cleaning_litterbox_202606200817.jpeg` |
| The Skater | Shredding the CI/CD pipeline | `Octopus_riding_skateboard_holdin…_202606200817_2.jpeg` |
| The Maestro | Curating Flow Music | `Octopus_mascot_writing_music_dia…_202606200817.jpeg` |

### Track Validation

Checking if a track is safe to load:

```go
// Prevent path traversal when loading audio files
func IsSafeTrackFile(filename string) bool {
    return filepath.IsLocal(filepath.FromSlash(filename))
}
```
