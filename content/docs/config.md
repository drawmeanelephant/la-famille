---
title: "Configuration Guide"
author: "Jules"
---

# Configuration Guide

La Famille uses a `config.yaml` file to manage site-wide settings. This file allows you to customize the behavior of the generator without having to pass multiple flags every time you build your site.

## Initializing the Configuration

If you don't have a `config.yaml` file in the root of your project, you can easily generate one with sensible defaults by running:

```bash
go run ./cmd/la-famille init
```

This will create a `config.yaml` file that looks something like this:

```yaml
# La Famille Site Configuration
#
# site_name: The name of your site, used in the navbar and footer.
site_name: "La Famille"

# template: The path to the HTML layout file used to render pages.
template: "templates/layout.html"

# content_dir: The directory containing your markdown source files.
content_dir: "content"

# output_dir: The directory where the generated HTML site will be placed.
output_dir: "public"

# theme: The DaisyUI theme applied to the site (e.g., retro, dark, cupcake, corporate).
theme: "retro"

# port: The port on which the local development server will run.
port: 8080
```

## Configuration Fields

Here is a breakdown of each available field:

*   **`site_name`** (string): The title of your website. This is often used by layouts in the header or footer navigation. *Default: "La Famille"*
*   **`template`** (string): The default HTML layout used for rendering pages. You can override this on a per-page basis using [frontmatter](templates.md). *Default: "templates/layout.html"*
*   **`content_dir`** (string): The source directory containing your Markdown `.md` files. *Default: "content"*
*   **`output_dir`** (string): The destination directory where the fully generated static site (HTML, JSON graphs, etc.) will be placed. *Default: "public"*
*   **`asset_dir`** (string): The directory containing static assets. *Default: "assets"*
*   **`rag_dir`** (string): The directory where RAG markdown bundles will be exported. *Default: "rag-archive"*
*   **`theme`** (string): The DaisyUI theme you want to apply globally to your site. This allows you to easily switch between "light", "dark", "retro", "synthwave", and many more! *Default: "retro"*
*   **`port`** (integer): The local network port used by the built-in HTTP server (`go run ./cmd/la-famille serve`). *Default: 8080*

## CLI Flag Overrides

While `config.yaml` sets the baseline, you can temporarily override several of these settings using Command Line Flags when running the `build` command.

For example, if you want to build an alternative content directory into a different output folder, you can run:

```bash
go run ./cmd/la-famille build -c my_docs -o dist
```

Any flags provided at runtime will take precedence over the values defined in `config.yaml`. See the [CLI Reference](cli.md) for more details.
