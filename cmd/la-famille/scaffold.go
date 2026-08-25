package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tbuddy/la-famille/internal/runtimeassets"
)

// demoContentFiles builds the starter pages `init` writes into an empty
// content directory. The index page exercises ordinary frontmatter and links,
// and the theming page switches layout via per-page frontmatter so a fresh
// site demonstrates both picking and switching themes without reading source.
func demoContentFiles(theme string, now time.Time) map[string][]byte {
	// An empty --theme means the default config, whose template is the plain
	// bundled layout — normalize so the switch demo always differs from it.
	if theme == "" {
		theme = "layout"
	}
	alternative := alternativeBundledTheme(theme)
	date := now.Format("2006-01-02")

	index := fmt.Sprintf(`---
title: "Home"
date: "%s"
tags:
  - welcome
---

# Welcome to La Famille

This page was scaffolded by `+"`la-famille init`"+`. Edit it in place, or create
a new post:

`+"```"+`sh
la-famille new my-first-post --title "My First Post"
`+"```"+`

Then preview your site with live reloading:

`+"```"+`sh
la-famille serve --watch
`+"```"+`

## Themes

The binary ships with a small packet of layouts. Run `+"`la-famille themes`"+`
to list them with descriptions. The site default is the `+"`template:`"+` line in
config.yaml; [Theming](theming.md) shows how to switch layout for a single
page.
`, date)

	theming := fmt.Sprintf(`---
title: "Theming"
date: "%s"
layout: %s
---

# Theming

This page does not use the site default template. Its frontmatter pins a
different bundled layout:

`+"```"+`yaml
layout: %s
`+"```"+`

Every layout in the release packet is installed into templates/ by
`+"`la-famille init`"+`, so switching is a one-line change and works on a
binary-only install. Run `+"`la-famille themes`"+` to see what ships.
`, date, alternative, alternative)

	return map[string][]byte{
		"index.md":   []byte(index),
		"theming.md": []byte(theming),
	}
}

// alternativeBundledTheme picks a bundled layout different from theme so the
// scaffolded theming page visibly switches away from the site default.
func alternativeBundledTheme(theme string) string {
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
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
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
