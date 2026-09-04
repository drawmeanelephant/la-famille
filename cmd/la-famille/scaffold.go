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
description: "A cozy corner of the web, built with La Famille."
date: "%s"
tags:
  - welcome
---

# Welcome to your site

This is your corner of the web: a quiet place for thoughts, projects, recipes, notes, and
whatever else feels worth sharing with friends and curious wanderers.

Take off your coat and have a look around:
- Read a little [About me](about.md) and what I'm making.
- Peek at [Theming](theming.md) to see how individual pages can wear different clothes.
- Wander through the [Markdown](markdown.md) showcase to see how words, code, and images live here.
`, date)

	about := fmt.Sprintf(`---
title: "About"
description: "A little about who lives here, what I care about, and how to reach me."
date: "%s"
---

# About me & this space

Hello! This page is a work-in-progress letter to visitors.

A few things about me and why this site exists:
- **What I'm building:** Replace this with a project, a passion, or your daily craft.
- **What I'm thinking about:** Books, ideas, gardens, or quiet observations.
- **Where to find me:** Link your email, Mastodon, Bluesky, or GitHub.

This site is handcrafted and independent — no algorithms, no trackers, just
plain words and HTML.

Head back to the [Home](index.md) porch, see how [Theming](theming.md) works,
or browse the [Markdown](markdown.md) showcase.
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

// projectRootFiles returns the developer and agent files bootstrapped by init.
func projectRootFiles() map[string][]byte {
	return map[string][]byte{
		".gitignore":                      []byte(defaultGitignore()),
		"README.md":                       []byte(defaultReadme()),
		"AGENTS.md":                       []byte(defaultAgentsManual()),
		".github/copilot-instructions.md": []byte(defaultCopilotInstructions()),
		".github/workflows/deploy.yml":    []byte(defaultPagesWorkflow()),
		".agents/plans/README.md":         []byte(defaultPlansReadme()),
	}
}

// scaffoldProjectRootFiles writes developer and agent guidance files missing-only,
// leaving any existing file untouched so operators and repos are never overwritten.
func scaffoldProjectRootFiles(dir string) ([]string, error) {
	files := projectRootFiles()
	var created []string
	for rel := range files {
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
	for _, rel := range created {
		target := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return nil, fmt.Errorf("create directory for %s: %w", target, err)
		}
		if err := writeFileNew(target, files[rel]); err != nil {
			return nil, fmt.Errorf("scaffold project file %s: %w", rel, err)
		}
	}
	return created, nil
}

func defaultGitignore() string {
	return `# La Famille static site build output
/public/

# Build cache
.la-famille-cache.json

# Logs
*.log
la-famille.log

# Configuration backups
config.yaml.bak
`
}

func defaultReadme() string {
	return `# Welcome to your La Famille site 🐙 🍔

A handcrafted, local-first website powered by [La Famille](https://github.com/drawmeanelephant/la-famille).

> *"Eight arms, one burger, infinite enthusiasm."*

## Quickstart

Preview your new home locally with live reload:

` + "```bash\nla-famille serve --watch\n```" + `

Visit http://localhost:8080 in your browser.

## Authoring Content

- **Draft a new page:**
  ` + "```bash\nla-famille new hello --title \"Hello World\"\n```" + `
- **Tend your links & diagnostics:**
  ` + "```bash\nla-famille check\n```" + `
- **Bake the static site:**
  ` + "```bash\nla-famille build\n```" + `

## Project Layout

- ` + "`content/`" + `: Markdown source files with YAML frontmatter.
- ` + "`templates/`" + `: HTML layout templates using Go ` + "`html/template`" + `.
- ` + "`assets/`" + `: Static CSS, JavaScript, and image assets.
- ` + "`config.yaml`" + `: Site configuration settings.
- ` + "`public/`" + `: Generated static output (ready to deploy).
- ` + "`AGENTS.md`" + `: Operating manual for AI coding assistants.

## Theming

List all bundled layouts:

` + "```bash\nla-famille themes\n```" + `

To change the default theme, update ` + "`template:` in `config.yaml`" + `. To change the theme for a single page, add ` + "`layout: <theme-name>`" + ` to that page's frontmatter.

## Deployment

To deploy to **GitHub Pages**:
1. Set ` + "`siteurl` in `config.yaml`" + ` to your hosted URL (e.g. ` + "`https://<user>.github.io/<repo>`" + `).
2. A ready-to-use GitHub Actions workflow is provided in ` + "`.github/workflows/deploy.yml`" + `.
`
}

func defaultAgentsManual() string {
	return `# La Famille Site — Agent Operating Manual

Welcome to the family. This repository contains a handcrafted, local-first static website generated with [La Famille](https://github.com/drawmeanelephant/la-famille). AI agents (Claude, Cursor, Copilot, Antigravity, Gemini) and human authors work collaboratively here as digital gardeners tending this web homestead.

## 1. System Philosophy & Architecture
- **Digital Homestead**: A personal corner of the web celebrating clean prose, local-first control, and craft aesthetics (*"eight arms, one burger, infinite enthusiasm"*).
- **Static Output**: La Famille compiles Markdown from ` + "`content/`" + ` and HTML templates from ` + "`templates/`" + ` into static files in ` + "`public/`" + `.
- **Never Edit ` + "`public/`" + `**: The ` + "`public/`" + ` directory is generated output. All edits belong in source files under ` + "`content/`" + `, ` + "`templates/`" + `, ` + "`assets/`" + `, or ` + "`config.yaml`" + `.
- **Direct Implementation**: Agents have ownership to create new posts, refine copywriting, adjust themes, fix broken links, and validate changes before handoff.

## 2. Directory Structure & File Conventions
- ` + "`content/`" + `: Markdown (` + "`.md`" + `) source files.
- ` + "`templates/`" + `: HTML templates using Go ` + "`html/template`" + ` syntax.
  - Bundled themes: ` + "`layout-octoburger.html`" + ` (flagship soul theme), ` + "`layout.html`" + `, ` + "`layout-terminal.html`" + `, ` + "`layout-editorial.html`" + `, ` + "`layout-midnight.html`" + `.
- ` + "`assets/`" + `: Static assets (` + "`css/`" + `, ` + "`js/`" + `, ` + "`img/`" + `).
- ` + "`config.yaml`" + `: Central site configuration.
- ` + "`public/`" + `: Generated static site (ignored by git).
- ` + "`.agents/plans/`" + `: Recommended directory for agent task plans and scratchpads.

## 3. Content Authoring Rules
- **Frontmatter Standard**:
  Every content file must begin with valid YAML frontmatter:
  ` + "```yaml\n---\ntitle: \"A Descriptive Title\"\ndescription: \"A concise summary for search engines and social cards.\"\ndate: \"YYYY-MM-DD\"\ntags:\n  - tag-name\ncategories:\n  - category-name\nlayout: layout-editorial  # optional: override default theme for this page\n---\n```" + `
- **Slug & File Naming**: Use lowercase alphanumeric characters and hyphens (e.g. ` + "`content/my-post.md`" + ` or ` + "`content/blog/announcement.md`" + `). Avoid uppercase letters, spaces, or special characters in filenames.
- **Internal Linking**:
  - Always link using relative markdown filenames: ` + "`[About](about.md)`" + ` or ` + "`[Overview](../guides/overview.md)`" + `.
  - Never use ` + "`.html`" + ` extensions in internal links; the generator resolves markdown paths to clean URL directories automatically.
  - Avoid broken internal links; run ` + "`la-famille check`" + ` to detect them.
- **Images & Figures**:
  - Place images in ` + "`assets/img/`" + ` and reference them with ` + "`/assets/img/<filename>`" + `.
  - Standalone images with titles automatically become HTML ` + "`<figure>` tags with `<figcaption>`" + `.

## 4. CLI Commands & Workflow
Always use the ` + "`la-famille`" + ` CLI to author and verify content:

- **Preview Site**:
  ` + "```bash\nla-famille serve --watch\n```" + `
  Starts local preview server at ` + "`http://localhost:8080`" + ` with auto-rebuild and live reload.

- **Scaffold New Content**:
  ` + "```bash\nla-famille new <slug> --title \"<Title>\" --description \"<Summary>\"\n```" + `
  Normalizes slug and populates valid frontmatter.

- **Diagnostic Check (Mandatory Verification)**:
  ` + "```bash\nla-famille check\n```" + `
  Verifies internal links, dates, metadata descriptions, and URL safety. Agents must ensure ` + "`check`" + ` reports 0 errors before finishing tasks.

- **Compile Static Output**:
  ` + "```bash\nla-famille build\n```" + `
  Generates static pages, sitemap.xml, RSS feed, and search index.

- **Export RAG Knowledge Bundles**:
  ` + "```bash\nla-famille rag\n```" + `
  Exports site content into RAG-friendly markdown bundles under ` + "`rag-archive/`" + `.

- **Explore Bundled Themes**:
  ` + "```bash\nla-famille themes\n```" + `

## 5. Execution Guardrails for Agents
1. **Planning**: Before complex refactoring or multi-file changes, create a brief plan in ` + "`.agents/plans/<task-id>.md`" + `.
2. **Quality Gate**: Before declaring any content authoring or theme task complete, run ` + "`la-famille check` and `la-famille build`" + ` locally. Both must succeed with zero errors.
3. **Clean Git Status**: Build output (` + "`public/`" + `) and cache (` + "`.la-famille-cache.json`" + `) are gitignored. Do not commit build artifacts to the source repository.
`
}

func defaultCopilotInstructions() string {
	return `# GitHub Copilot Instructions for La Famille Site

This project is a static website built with [La Famille](https://github.com/drawmeanelephant/la-famille).

## Key Instructions for Copilot
- **Content**: Markdown source files live in ` + "`content/`" + `. Every page must have YAML frontmatter with ` + "`title`" + `, ` + "`description`" + `, and ` + "`date`" + `.
- **Links**: Use relative markdown links between content files (e.g. ` + "`[About](about.md)`" + ` or ` + "`[Post](../posts/my-post.md)`" + `). Do not write ` + "`.html`" + ` links.
- **Templates**: Layouts live in ` + "`templates/`" + ` using Go ` + "`html/template`" + ` syntax. Page layouts can be customized via frontmatter ` + "`layout: <theme>`" + `.
- **Validation**: After editing content or templates, run ` + "`la-famille check`" + ` to verify internal links, slugs, and metadata.
- **Build**: Run ` + "`la-famille build`" + ` to compile the static site into ` + "`public/`" + `. Never edit ` + "`public/`" + ` directly.
- **Full Agent Manual**: Refer to ` + "`AGENTS.md`" + ` for detailed rules of engagement and conventions.
`
}

func defaultPagesWorkflow() string {
	return `name: Deploy to GitHub Pages

on:
  push:
    branches: ["main", "master"]
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: "pages"
  cancel-in-progress: false

jobs:
  build-and-deploy:
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Pages
        id: pages
        uses: actions/configure-pages@v5
        with:
          enablement: true

      - name: Install La Famille
        run: |
          REPO="drawmeanelephant/la-famille"
          TAG=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
          if [ -z "$TAG" ]; then TAG="v0.1.0-prealpha"; fi
          VERSION="${TAG#v}"
          ARCHIVE="la-famille_${VERSION}_linux_amd64.tar.gz"
          URL="https://github.com/$REPO/releases/download/$TAG/$ARCHIVE"
          echo "Downloading La Famille $TAG..."
          curl -sL "$URL" -o /tmp/la-famille.tar.gz
          tar -xzf /tmp/la-famille.tar.gz -C /usr/local/bin la-famille
          chmod +x /usr/local/bin/la-famille
          la-famille --version

      - name: Build site
        env:
          SITE_URL: ${{ steps.pages.outputs.base_url }}
        run: |
          la-famille build --site-url "$SITE_URL"
          la-famille publish-check --site-url "$SITE_URL"

      - name: Upload Pages artifact
        uses: actions/upload-pages-artifact@v3
        with:
          path: public

      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v4
`
}

func defaultPlansReadme() string {
	return `# Agent Task Plans

This directory stores task plans and execution scratchpads created by AI agents (Claude, Cursor, Copilot, Antigravity, etc.) before making multi-file modifications.

## Plan Format
Agents should create ` + "`.agents/plans/<task-name>.md`" + ` with:
- Task goal & problem statement
- Proposed changes grouped by file
- Verification steps (` + "`la-famille check`" + `, ` + "`la-famille build`" + `)
`
}
