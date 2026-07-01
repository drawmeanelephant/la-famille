---
title: "Emoji Kitchen Stickers"
author: "Jules"
---

# Emoji Kitchen Stickers

La Famille includes a custom inline Goldmark parser that allows you to easily embed mutant emoji stickers directly from Google's Emoji Kitchen CDN.

## Shorthand Syntax

You can render an emoji combination using the following exact shorthand syntax anywhere in your Markdown files:

```markdown
!ek[emoji+emoji]
```

### Example

To render a combination of a ghost and a pizza, simply write:

```markdown
!ek[👻+🍕]
```

This will automatically be converted into an HTML `<img>` tag pointing to the correctly constructed gstatic CDN URL for the combined sticker.

*Note: The parser is strict about the syntax and requires the exclamation mark, 'ek', and the square brackets containing exactly two emojis separated by a plus sign.*
