---
title: "Multi-Cloud Static Deployment Guide"
author: "Core Architecture Team"
date: "2026-07-07"
description: "How to deploy La Famille static site outputs to GitHub Pages, Cloudflare Pages, and standard worker pipelines."
---

# Multi-Cloud Static Deployment

La Famille compiles into a flat, zero-dependency directory containing static HTML, asset trees, and JSON metadata graphs. This makes the build output universally portable across modern web-hosting infrastructures.

---

## ☁️ Deployment Targets

### 1. Cloudflare Pages
Cloudflare Pages provides global edge distribution with fast deployment turnarounds.

**Configuration Parameters:**
*   **Build Command:** `go run ./cmd/la-famille build`
*   **Build Output Directory:** `public`
*   **Root Directory:** *(Leave as project root)*

### 2. GitHub Pages (Actions Workflow)
To deploy natively using GitHub Actions, ensure your repository contains a deployment execution file within `.github/workflows/deploy.yml`. The workflow should trigger on push events, compile the project, and pass the standard artifact back to the static routing bucket.

### 3. Bitbucket Pipelines & Generic Web Buckets
Because the output contains no server-side execution routines, any traditional container or static pipeline can handle the build loop:

```bash
# 1. Pull dependencies and compile content
go run ./cmd/la-famille build

# 2. Sync public directory to your hosting provider's bucket or worker
# (e.g., rsync, aws s3 sync, or cloudflare wrangler login pages deploy)
```
