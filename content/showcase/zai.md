---
title: "Z.ai Field Guide"
description: "Two languages, a custom theme, and a source-linked guide built from plain Markdown."
date: "2026-10-01"
---

## Two languages. One field guide.

[Visit the English edition](https://z.filed.fyi/),
[read the Chinese edition](https://z.filed.fyi/zh/), or
[browse the source](https://github.com/drawmeanelephant/z.filed.fyi).

The Z.ai Field Guide is an independent publication covering Z.ai and Zhipu AI:
products, models, company history, and links to its sources. It is not affiliated
with or endorsed by those companies.

## What La Famille does here

- Builds English and Chinese content trees into one static site.
- Applies custom, site-owned layouts and self-hosted assets.
- Generates search, taxonomy, graph data, and machine-readable content.
- Runs in CI using a checksum-verified, pinned release binary.

Bilingual navigation is implemented with mirrored paths, per-language templates,
and a theme-level switcher. La Famille does not supply an internationalization
framework or translate pages at build time.

## The useful edges

The site's repository documents its actual publishing pipeline, including
post-processing for search, graph styling, and language alternates. The finished
site is not entirely unmodified generator output. Its deployment currently
pins an older pre-alpha release, so limitations in its build notes must be
compared with that binary rather than assumed to apply to current source.

This is a good compatibility workload: a custom theme, two writing systems,
real navigation, and a release-based deployment. It is also a useful example of
the distinction between a generator feature and something a site composes.

## What to try

Open [the model guide](https://z.filed.fyi/models/), find a source-linked model
page, and switch to its Chinese counterpart. The site's
[La Famille section](https://z.filed.fyi/la-famille/) explains its build in more
detail.

This showcase does not independently verify the guide's vendor, pricing, model,
or licensing claims.

[Back to the showcase](index.md).
