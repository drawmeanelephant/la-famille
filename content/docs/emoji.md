---
title: "Emoji Kitchen Stickers"
author: "Jules"
---

# Emoji Kitchen Stickers

La Famille includes a custom inline Goldmark parser that allows you to easily embed mutant emoji stickers directly from Google's Emoji Kitchen CDN.

<div class="bg-primary/10 border-l-4 border-primary p-4 my-6">
  <strong>Note:</strong> This feature requires an active internet connection to load the images from Google's gstatic CDN.
</div>

## Shorthand Syntax

You can render an emoji combination using the exact shorthand syntax anywhere in your Markdown files:

```markdown
!ek[emoji1+emoji2]
```

### Example: Ghost Pizza

To render a combination of a ghost and a pizza, write:

```markdown
!ek[👻+🍕]
```

This will automatically be converted into an HTML `<img>` tag pointing to the correctly constructed gstatic CDN URL for the combined sticker.

**Resulting HTML output:**

```html
<img src="https://www.gstatic.com/android/keyboard/emojikitchen/20201001/u1f47b/u1f47b_u1f355.png" alt="Emoji Kitchen combination of 👻 and 🍕" title="Emoji Kitchen combination of 👻 and 🍕">
```

### Strict Syntax Rules

<div class="bg-warning/10 border-l-4 border-warning p-4 my-6">
  <strong>Important:</strong> The parser is strict. It requires:
  <ul class="list-disc ml-6 mt-2">
    <li>The exclamation mark `!`</li>
    <li>The exact string `ek`</li>
    <li>Square brackets `[]`</li>
    <li>Exactly two emojis inside the brackets</li>
    <li>A plus sign `+` separating the two emojis</li>
  </ul>
</div>

## Styling and Accessibility

The rendered `<img>` tag does not include any default styling classes. If you need to style the emoji (e.g., set a specific width or margin), you can wrap it in a parent container with utility classes, or rely on the `prose-img` styles applied by Tailwind Typography if used within an `<article class="prose">`.

The parser automatically generates descriptive `alt` text and a `title` attribute for accessibility, ensuring screen readers can announce the emojis used to create the combination.
