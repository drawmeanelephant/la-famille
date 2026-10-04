---
date: "2026-10-01"
title: "Your first La Famille site"
description: "Download a binary, write a page, and build a website you can keep."
---

## 1. Get the binary

Open [the releases page](https://github.com/drawmeanelephant/la-famille/releases)
and download the archive for your operating system and architecture, plus
`SHA256SUMS`. Compare the archive's SHA-256 digest with its matching entry
before extracting it. On macOS use `shasum -a 256 <archive>`; on Linux use
`sha256sum <archive>`. A different digest means stop and download again.

Extract the verified archive and put `la-famille` on your `PATH`, or substitute
the executable's absolute path in the commands below. Then check its identity:

```bash
la-famille --version
```

**No Go installation is needed for the released binary.** La Famille is
pre-alpha, so keep your content in version control and expect some rough edges.

## 2. Give it a folder

Run these commands from the directory where you want to keep your site:

```bash
mkdir my-site
cd my-site
la-famille init
la-famille build
```

`init` installs configuration, a starter homepage, bundled layouts, and runtime
assets. `build` writes the complete static site into `public/`. Your Markdown
stays in `content/`; it is not replaced by the generated HTML.

## 3. Write something worth linking to

```bash
la-famille new notes/first-note --title "My first note"
```

Open `content/notes/first-note.md` in your editor and replace the starter text.
Link it from `content/index.md` with ordinary Markdown:

```markdown
[Read my first note](notes/first-note.md)
```

Run the checker and rebuild:

```bash
la-famille check
la-famille build
```

The checker reports content problems; the build generates your pages, search
index, graph, and publishing metadata. [Frontmatter](frontmatter.md) covers
titles, dates, tags, and `publish: false`.

## 4. See it locally

```bash
la-famille serve --watch
```

Open `http://localhost:8080`. The server rebuilds when source files change.
Press `Ctrl+C` to stop it. Prefer the terminal UI? Run `la-famille tui`.

You can also work from another directory:

```bash
la-famille --project-root /absolute/path/to/my-site build
```

Paths in configuration and CLI output options resolve against the selected
project root. Absolute paths avoid ambiguity when driving another project.

## 5. Publish files, not your workspace

Set `siteurl` in `config.yaml` to your full public address. A subdomain uses
an origin such as `https://notes.example.com`; a GitHub Pages project site
must include its `/repository` subpath.

Build again, run `la-famille publish-check`, and upload **only `public/`** to
your static host. Do not upload your private build cache or the entire source
checkout. See [the publishing contract](publishing.md).

Corpus export is a separate, optional step. Build first, then export:

```bash
la-famille rag --output "$PWD/public/rag-archive"
```

This creates content, system, and configuration bundles. Review what each
contains before publishing any of them; a content-only site normally only needs
`rag-content.md`. A later build may replace `public/`, so re-export afterward.

## Working on La Famille itself?

Use the Go version declared in the repository's `go.mod` (currently Go 1.26):

```bash
git clone https://github.com/drawmeanelephant/la-famille.git
cd la-famille
go build -o ./bin/la-famille ./cmd/la-famille
./bin/la-famille --version
```

For this project's flagship site, use its separate configuration:

```bash
./bin/la-famille --config website.yaml build
./bin/la-famille --config website.yaml serve
```

The source build includes newer features that the last published pre-alpha
release may not have. Always check `--version` and the command's `--help`.

## Pick your next path

- [Themes and templates](templates.md): make the site look like you.
- [The terminal UI](tui.md): the same workflow, with an octopus.
- [Configuration](config.md): paths, names, and publishing addresses.
- [Corpus exports](rag.md): portable content for external tools.
- [Real sites](../showcase/index.md): see what other content looks like.
