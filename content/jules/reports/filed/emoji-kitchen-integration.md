---
title: "Emoji Kitchen Integration"
date: "2026-06-30"
tags: ["routine", "emoji", "goldmark"]
author: "Jules"
---

# Emoji Kitchen Integration

**Date:** 2026-06-30
**Routine:** Emoji Kitchen Integration
**Status:** Success

## Overview
I implemented a custom Goldmark inline parser (`EmojiKitchenParser` in `internal/transform/emoji_kitchen.go`) to support the `!ek[emoji+emoji]` syntax in markdown.

## Technical Details
- The parser extracts the two base emojis, computes their hex unicode equivalents, and renders an `<img>` tag pointing to Google's static CDN.
- It is wired into `generator.Build()` alongside our AST transformers.
- Added comprehensive unit tests in `internal/transform/emoji_kitchen_test.go` and verified correct AST replacement.
- Tested end-to-end functionality by updating `content/showcase/devlog.md` with an `!ek[🐢+🔥]` test fixture.

## Learnings
- Goldmark's inline parser system (`Trigger()`, `Parse()`) is well-suited for converting custom markdown shorthand directly into standard `ast.Image` nodes, which provides cleaner output compared to trying to inject images during the AST transformation phase.
- Using standard runes natively extracts the unicode hex components correctly for emoji mapping.
