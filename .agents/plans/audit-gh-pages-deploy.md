# Task Plan: Audit GitHub Pages Deployment Workflow (audit-gh-pages-deploy)

## Overview & Scope
Audit and fix `.github/workflows/deploy.yml` so that `la-famille build` receives the deployed public site URL without hardcoding a guessed URL.

Specifically:
1. Re-order `.github/workflows/deploy.yml` steps so `Setup Pages` (`actions/configure-pages@v6`) with `id: pages` runs *before* `Build Static Site`.
2. Determine `SITE_URL` dynamically from workflow inputs (`${{ inputs.site_url }}`), repository/environment variables (`${{ vars.SITE_URL }}`), or `actions/configure-pages` output (`${{ steps.pages.outputs.base_url }}`).
3. Pass `--site-url` to `go run ./cmd/la-famille build` when `SITE_URL` is set in `deploy.yml`.
4. Add environment variable fallback (`SITE_URL` or `LA_FAMILLE_SITE_URL`) in `cmd/la-famille/main.go` so `la-famille build` automatically picks up `SITE_URL` from the environment if `--site-url` was not explicitly passed on the CLI and `cfg.SiteURL` is empty.
5. Update docs (`publishing.md`, `multi-cloud-deployment.md`, `config.md`) to document CI/CD public URL resolution and workflow usage.
6. Add workflow/config regression unit tests in `cmd/la-famille`.

## Impact Analysis
- **Static Output Impact:** Sites built in GitHub Pages deployment workflow will now have populated canonical tags, OG URLs, RSS links, sitemap locations, and `robots.txt` Sitemap directives using the exact public URL assigned by GitHub Pages (or overridden by repository variables/workflow inputs).
- **Template/Theme Impact:** Zero modifications to page templates or themes.

## Verification
- Unit tests: `go test ./...`
- Go vet: `go vet ./...`
