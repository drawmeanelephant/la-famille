---
title: "The Maestro's Studio (Floating Cards Layout Showcase)"
layout: layout-floating-cards
---

## The Studio

The Maestro orchestrates the vibes in the `content/soundtrack/` directory. Curating Flow Music prompts and spinning up synthwave and 90s boom-bap tracks is essential after a long coding session.

### The Setlist

What's playing in the studio today:

- Synthwave
  - Neon Nights
  - Cybernetic Dreams
  - Outrun the Sun
- 90s Boom-Bap
  1. MPC Grooves
  2. Vinyl Scratch
  3. Lofi City

> "The code is the structure, but the beat is the soul. You need both to reach true flow state." — The Maestro

### The Band

| Persona | Responsibility | Associated Image Asset |
|---|---|---|
| The Janitor | Cleaning the repo, clearing branches | `Octopus_mascot_cleaning_litterbox_202606200817.jpeg` |
| The Skater | Shredding the CI/CD pipeline | `Octopus_riding_skateboard_holdin…_202606200817_2.jpeg` |
| The Maestro | Curating Flow Music | `Octopus_mascot_writing_music_dia…_202606200817.jpeg` |

### Streaming Logic

Safely streaming the tracks requires proper encoding:

```javascript
// Encode track names for safe streaming links
function getStreamUrl(trackName) {
    return `/stream?track=${encodeURIComponent(trackName)}`;
}
```
