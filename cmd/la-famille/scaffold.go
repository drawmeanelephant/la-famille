package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/runtimeassets"
)

// demoContentFiles builds the starter pages `init` writes into an empty
// content directory: a homepage and an about page that read like a real site
// (not tool documentation), a theming page pinned to a different bundled
// layout so a fresh site visibly demonstrates per-page theme switching, and
// a markdown page that showcases every element the goldmark engine renders
// (credit included) so a new author sees the full palette at first glance.
func demoContentFiles(theme string, now time.Time) map[string][]byte {
	// An empty --theme means the default config. Resolve the actual default
	// (octoburger) rather than assuming a hardcoded layout name: since the
	// default changed, the alternative picker must still switch away from it.
	if theme == "" {
		theme = config.DefaultLayoutPath
	}
	defaultName := strings.TrimSuffix(filepath.Base(theme), ".html")
	alternative := alternativeBundledTheme(defaultName)
	date := now.Format("2006-01-02")

	index := fmt.Sprintf(`---
title: "Home"
description: "The homepage of a site built with La Famille."
date: "%s"
tags:
  - welcome
---

# Welcome to your site

This is your homepage. Replace this draft with real words about what you make,
write, or care about: a proper welcome, a paragraph on what this site is for,
and links to the pages you care about.

Say hello on the [About](about.md) page, peek at [Theming](theming.md) —
a page that uses a different bundled layout — and browse the
[Markdown](markdown.md) page to see every element this site can render.
`, date)

	about := fmt.Sprintf(`---
title: "About"
description: "About this site and the person behind it."
date: "%s"
---

# About this site

Who are you, and what is this site for? Write a proper introduction here —
this draft is a placeholder until you make it yours.
`, date)

	theming := fmt.Sprintf(`---
title: "Theming"
description: "A demo page pinned to a different bundled layout."
date: "%s"
layout: %s
---

# Theming

Every other page on this site renders with the site default layout, `+"`%s`"+`.
This page is pinned to a different bundled layout in its frontmatter:

`+"```"+`yaml
layout: %s
`+"```"+`

That is the whole trick: one line of frontmatter per page. The site-wide
default lives in the `+"`template:`"+` line of config.yaml, and
`+"`la-famille themes`"+` lists every layout in the packet.
`, date, alternative, defaultName, alternative)

	markdown := fmt.Sprintf(`---
title: "Markdown"
description: "A living showcase of every markdown element this site can render."
date: "%s"
---

## Headings

Use `+"`#`"+` through `+"`######`"+` for headings. This page's title is the h1; the
sections below start at h2 so the outline stays tidy.

## Inline styles

This is **bold**, this is *italic*, and this is ~~strikethrough~~. You can
combine **bold and *nested* emphasis**, and mark `+"`code`"+` inline like so.

## Links

Here is a [relative link](about.md) and an absolute one:
[https://goldmark.dev](https://goldmark.dev). Bare URLs like
https://example.com are linked automatically (GFM linkify).

## Quotes

> A blockquote pulls a passage aside. Markdown engines like goldmark make it
> a first-class element, so the theme can dress it up properly.

## Lists

- unordered item
- another item
  - nested item

1. ordered item
2. ordered item

And task lists, straight from GitHub Flavored Markdown:

- [x] shipped
- [ ] in progress

## Code

Fenced code blocks keep their language and get a dark, readable surface:

`+"```"+`go
func hello() string {
    return "hello, world"
}
`+"```"+`

## Tables

| Feature | Status |
| ------- | ------ |
| Tables  | works  |
| Figures | works  |

## Rules and images

A horizontal rule splits long pages:

---

![Raoul(s), the La Famille octopus mascot](/assets/img/mascot-default.jpeg "Standalone images become figures with captions")

Emoji Kitchen blends are first-class too: !ek[🐢+🔥]

## The engine behind this page

Everything above is rendered by [goldmark](https://goldmark.dev), the
Markdown parser La Famille is built on — a complete CommonMark
implementation with GitHub Flavored Markdown (tables, strikethrough, task
lists, autolinks) and smart punctuation layered on top. What you see here is
what every theme ships with, no plugins required.
`, date)

	return map[string][]byte{
		"index.md":    []byte(index),
		"about.md":    []byte(about),
		"theming.md":  []byte(theming),
		"markdown.md": []byte(markdown),
	}
}

// alternativeBundledTheme picks the bundled layout the theming demo pins: one
// that differs from the site default and reads as a deliberate showcase.
// Octoburger is the flagship, so it is the showcase when the site default is
// anything else; when the default *is* octoburger, editorial gives the most
// distinct designed contrast (the plain look is a poor first impression).
func alternativeBundledTheme(theme string) string {
	preferred := "layout-editorial"
	if theme != "layout-octoburger" {
		preferred = "layout-octoburger"
	}
	for _, candidate := range runtimeassets.CuratedLayoutNames {
		if candidate == preferred {
			return preferred
		}
	}
	for _, candidate := range runtimeassets.CuratedLayoutNames {
		if candidate != theme {
			return candidate
		}
	}
	return "layout"
}

// scaffoldDemoContent writes the starter pages missing-only, leaving any
// existing file untouched. It reports which files it created.
func scaffoldDemoContent(dir string, demos map[string][]byte) ([]string, error) {
	var created []string
	for rel := range demos {
		target := filepath.Join(dir, filepath.FromSlash(rel))
		if _, err := os.Stat(target); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("inspect %s: %w", target, err)
		}
		created = append(created, rel)
	}
	if len(created) == 0 {
		return nil, nil
	}
	sort.Strings(created)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create content directory %s: %w", dir, err)
	}
	for _, rel := range created {
		data := demos[rel]
		target := filepath.Join(dir, filepath.FromSlash(rel))
		if err := writeFileNew(target, data); err != nil {
			return nil, fmt.Errorf("scaffold demo content: %w", err)
		}
	}
	return created, nil
}

// writeFileNew creates the file exclusively so a file appearing between the
// existence check above and this write is never clobbered.
func writeFileNew(target string, data []byte) error {
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("create %s: %w", target, err)
	}
	if _, writeErr := file.Write(data); writeErr != nil {
		_ = file.Close()
		return fmt.Errorf("write %s: %w", target, writeErr)
	}
	if closeErr := file.Close(); closeErr != nil {
		return fmt.Errorf("close %s: %w", target, closeErr)
	}
	return nil
}

// formatThemeChoices renders the bundled packet as an aligned list used by
// the unknown-theme error.
func formatThemeChoices() string {
	themes := runtimeassets.CuratedThemes()
	width := 0
	for _, theme := range themes {
		if len(theme.Name) > width {
			width = len(theme.Name)
		}
	}
	lines := make([]string, 0, len(themes))
	for _, theme := range themes {
		lines = append(lines, fmt.Sprintf("  %-*s  %s", width, theme.Name, theme.Description))
	}
	return strings.Join(lines, "\n")
}
