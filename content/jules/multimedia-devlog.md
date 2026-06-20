# Multimedia Devlog Generation Routine

This routine handles the creation of a multimedia-focused development log update. Use this routine to summarize recent work into punchy, visually-driven content meant for social media or video platforms.

## Execution Steps

1. **Review Recent Work:** Analyze the recent commits, PRs, or project backlog to understand the latest development cycle.
2. **Draft Content:**
   - **Video Script:** Write a punchy, 10-second explainer video script summarizing the update. Keep it concise, engaging, and high-energy.
   - **Animation Cues:** Write visual directions corresponding to the video script, intended for an AI flow video creator tool (e.g., "Cut to Jules the Octopus typing furiously on a neon keyboard").
   - **Soundtrack Theme:** Draft a short snippet of song lyrics or a musical theme description that matches the vibe of the update.
3. **Create the Devlog File:**
   - Create a new markdown file in the `content/devlog/` directory. Use a descriptive filename (e.g., `content/devlog/added-search-feature.md`).
   - Include the standard frontmatter (`title`, `author`, `date`) PLUS the new multimedia fields (`video_script`, `animation_cues`, `soundtrack_theme`, `layout`).
   - **Important:** Set `layout: devlog` in the frontmatter to ensure the correct template is applied.
   - Add a brief summary paragraph in the body content.

## Example File Output

```yaml
---
title: "La Famille Gets Search!"
author: "Jules"
date: "2024-05-20"
layout: devlog
video_script: "Searching for content just got easier! We've added full-text search to La Famille using bleve. Just type and find instantly."
animation_cues: "Zoom in on a search bar. The word 'octopus' is typed. Fast transition to a grid of results popping up with a neon glow."
soundtrack_theme: "Upbeat synthwave bassline. Lyrics: 'Searching the depths, finding the light, data flows fast into the night.'"
---

This week, we focused on making navigation seamless by integrating full-text search capabilities across all generated static pages.
```
