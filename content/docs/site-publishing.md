---
title: "How this site ships"
description: "One repository, one canonical address, a validated artifact, and an explicit publishing decision."
date: "2026-10-01"
---

## The website lives with the generator

This flagship is built from the repository's `content/`, `templates/`, and
`assets/` directories. Its configuration is `website.yaml`, separate from the
`config.yaml` used for generator defaults and new-site scaffolding. Your newly
initialized site does not inherit our hostname or website-specific layout.

The canonical address is **https://la-famille.filed.fyi**.
The existing Cloudflare Pages project is **la-famille-go**, with the project
address **https://la-famille-go.pages.dev**. The project address is not a
second editorial site. Canonical metadata and the sitemap use the custom domain.
Associating that domain and configuring DNS are host/operator tasks, not build
steps.

## Build locally

From this repository, using the Go version in `go.mod`:

```bash
go build -o ./bin/la-famille ./cmd/la-famille
./bin/la-famille --config website.yaml build
./bin/la-famille --config website.yaml publish-check
./bin/la-famille --config website.yaml serve
```

The website intentionally uses a source-built compiler because it documents
recent work that is not all present in the last published pre-alpha release.
Existing documentation, template previews, and source-content paths stay
available.

## Cloudflare workflow

`.github/workflows/website.yml` builds and validates on pull requests and pushes.
It saves the complete `public/` tree as the `flagship-website` artifact.
Successful pushes to `master` publish that artifact automatically. Pull
requests and feature branches never deploy, and the build job does not receive
Cloudflare secrets.

Merging an approved PR to `master` publishes the site; there is no manual file
upload. To retry a publication, an authorized operator can also run
**Flagship website** manually on `master` and select **deploy = true**.
The deploy job requires a successful build and uses the existing repository
secret names `CLOUDFLARE_ACCOUNT_ID` and `CLOUDFLARE_API_TOKEN`. No secret value
belongs in configuration, content, output, or a launch record.

The publisher targets the existing Pages project. A GET-only API lookup reads
the project's current production branch and passes it to Wrangler, rather than
guessing `main` or `master` or creating a preview by mistake. The token must
permit that Pages project lookup and upload. The workflow does not create
the project, attach a domain, change DNS, or silently publish from a feature
branch. Configure environment approval rules for `cloudflare-pages` if another
human review is required.

## What's public

The artifact includes generated HTML, referenced assets, search and graph data,
taxonomy, feeds, the sitemap, and deterministic build manifests/ledger files.
The corpus exporter runs outside `public/`; only its nonempty
`rag-content.md` is copied into `public/rag-archive/`.

System and configuration RAG bundles stay outside the Cloudflare artifact.
The private build cache is not published. Public content still requires
editorial review; passing an artifact check is not proof that every prose
claim is true.

Some existing documentation intentionally demonstrates missing-page stubs.
The workflow uses the normal publishing validator and preserves those examples;
it does not claim a strict, stub-free content audit.

## GitHub Pages remains a fallback

The existing `.github/workflows/deploy.yml` remains independent. It uses the
GitHub Pages base URL and its existing artifact contract, including its
documented RAG bundles. The new homepage can render under the `/la-famille`
subpath; root-relative template URLs are rebased by the renderer.

There is one source of content, not two manually maintained documentation sites.
The flagship uses the website layout; the fallback preserves its existing
Octoburger default layout for other pages.

## Before calling a launch complete

1. Review the built artifact, source revision, canonical URL, and compiler.
2. Confirm project/domain authority and the production-branch setting.
3. Merge the approved source to `master`; the workflow publishes its validated artifact.
4. Verify the actual custom domain: homepage, docs, showcase, search, graph,
   referenced assets, sitemap, and content corpus.
5. Keep the last known-good deployment and use Cloudflare's operator-controlled
   rollback if needed.

Configuration in this repository is not evidence of a successful remote
deployment. The two supplied Cloudflare addresses returned HTTP 522 during the
initial implementation inspection; a later authorized launch must verify the
actual host rather than infer success from local checks.
