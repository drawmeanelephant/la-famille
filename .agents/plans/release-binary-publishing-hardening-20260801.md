# Release Binary and Publishing Hardening Plan

## Task ID
`release-binary-publishing-hardening-20260801`

## Goal
Move La Famille from a source-aware developer workflow toward a repeatable, source-independent release workflow while preserving the observed static-site output contract. Then rerun the same three dogfooding branches against the hardened implementation.

## Evidence
Use the three isolated reports as the baseline:

- Local static: `/private/tmp/la-famille-dogfood-local-static-isolated/.agents/dogfood/local-static-isolated.md`
- GitHub Pages: `/private/tmp/la-famille-dogfood-github-pages-isolated/.agents/dogfood/github-pages-isolated.md`
- Release binary: `/private/tmp/la-famille-dogfood-release-binary-isolated/.agents/dogfood/release-binary-isolated.md`

All three began at `527bef1`, passed `go test ./...` and `go vet ./...`, and found no production-code changes in the dogfood pass.

## Recommended product decisions

1. **Separate release-owned runtime assets from site-owned content.** Embed the default layout, required partials, and graph explorer CSS/JS in the binary using `go:embed`, while allowing explicit site overrides. This makes a fresh `init` + `build` complete without requiring agents to discover repository assets.
2. **Make the site root explicit.** Add a stable `--project-root` or `--config` contract, plus `--asset-dir` where needed. Preserve current-directory behavior as the default for compatibility, but document precedence: explicit flags > `config.yaml` > current directory/defaults.
3. **Add build identity.** Implement `la-famille --version` and a machine-readable form such as `--version --json`, populated by documented linker flags with semantic version, commit, build date, target, and Go version.
4. **Treat `public/` as the publish artifact.** Document and test the complete tree: clean-URL pages, taxonomies, graph files, discovery files, copied assets, and intentional raw files. Keep `.la-famille-cache.json` out of public output by default (or give it an explicit documented inclusion mode); correct the `.buildcache.json` documentation mismatch.
5. **Keep Pages artifact-based.** The current contract is `public/` contents uploaded to the Pages artifact/deploy-pages actions, not a `gh-pages` branch. State that plainly in the quickstart and make the workflow consume a pinned release binary once the release pipeline exists.
6. **Make RAG publishable without checkout side effects.** Add explicit project/content/output flags to `rag`, or a shared project-root/output contract, so Pages can write directly to `public/rag-archive` in a temporary build directory.

## Implementation phases

## Potential breaking changes

- The build cache will move out of `public/`; any workflow that currently
  relies on `.la-famille-cache.json` being present in the publish artifact must
  stop doing so.
- Relative paths supplied by the CLI will resolve from `--project-root` when
  that flag is present, rather than implicitly from the process CWD.
- The default initialized asset set will include release-owned fallback files;
  existing site files remain authoritative and are never overwritten by
  `init`.
- The Pages workflow will continue to publish the complete `public/` tree, but
  its preferred compiler path will become a verified release binary with a
  source-build fallback for development.

### Phase 0 — Contract tests and baseline

- Create one implementation branch from `origin/master` (for example `codex/release-binary-contract`).
- Add temporary-directory tests for version output, project-root/config resolution, init/build from outside the source tree, asset resolution, cache publication policy, and RAG output isolation.
- Freeze an output manifest for a representative site so future changes show whether `public/` drifted.

### Phase 1 — CLI/runtime boundary

- Add version/build-info variables, `--version`, and JSON output with stable exit codes.
- Add project-root/config/asset-dir resolution without breaking existing `--content`, `--output`, `--template`, and `--site-url` flags.
- Embed release-owned runtime assets and let site-owned assets override them explicitly.
- Update `init` so a default initialized site either contains every asset required for a successful graph build or disables graph explorer until assets are present. Prefer the embedded-asset path.
- Add RAG output/project-root flags and tests.

### Phase 2 — Static output and Pages contract

- Move or filter the build cache so it is not accidentally published; document the final policy and exact filename.
- Add a publisher-facing manifest/check command that verifies required relative references resolve.
- Update `.github/workflows/deploy.yml`, `action.yml`, and publishing docs to use the explicit artifact contract and, once available, a pinned binary with checksum verification.
- Keep a source-build fallback for development, clearly separated from the release path.

### Phase 3 — Release pipeline

- Produce versioned archives per supported OS/architecture, with `SHA256SUMS`, license/notice, and a short binary quickstart.
- Build in a clean official CI toolchain and fail the release if any advertised target cannot cross-build or pass its smoke test.
- Smoke-test each archive offline: `--version`, `--help`, invalid input, `init`, `new`, `build`, `check`, and generated local-asset reference validation.
- Decide whether CDN references in the stock template are acceptable; if not, bundle or document the required offline frontend assets.

### Phase 4 — Documentation and migration

- Align README, CLI, config, setup, publishing, Pages, and action docs with the final flags, output tree, cache policy, RAG behavior, and release archive layout.
- Add a short “source checkout vs released binary” decision table.
- Document that Pages uses an artifact/deploy workflow and that `gh-pages` is not implied.

## Retest protocol using the same branches

Do not create replacement dogfood branch names. After the implementation branch is ready, apply its implementation commits to the existing branches while preserving their original reports:

- `codex/dogfood-local-static-isolated`: append a retest section covering version, init/new/build/check, serve/watch, and all generated asset references.
- `codex/dogfood-github-pages-isolated`: append a retest section covering project-site URLs, artifact root shape, cache policy, RAG output isolation, and the pinned-binary workflow.
- `codex/dogfood-release-binary-isolated`: append a retest section covering an unpacked release archive run entirely outside the source tree, offline, plus every advertised target.

Each branch must remain a real worktree under `/private/tmp`, use system Git for metadata operations, and commit only its plan/report changes plus any explicitly assigned implementation cherry-pick. No pushes or Pages deployments are part of the retest.

## Acceptance criteria

- A released binary can be unpacked into an empty directory and identified with `--version` without Go, source, or network access.
- A new operator can initialize a site and produce a complete graph-capable build without discovering hidden repository assets.
- A binary launched outside the source tree can select its project root/config/assets explicitly; CWD is no longer a hidden required input.
- A Pages build can generate all intended output, including optional RAG content, without mutating the repository checkout.
- The public artifact policy for cache files, raw files, graph assets, and relative URLs is documented and covered by tests.
- Every supported release target builds and passes its offline smoke test.
- The three same-name dogfood branches complete their retest with no unresolved P1 findings.

## Copy/paste handoff prompt

> Implement `.agents/plans/release-binary-publishing-hardening-20260801.md` from `origin/master`. Start with Phase 0 contract tests, then implement the CLI/runtime and Pages changes in one reviewable branch. Do not alter the three existing dogfood reports; after the implementation is ready, apply the implementation commits to `codex/dogfood-local-static-isolated`, `codex/dogfood-github-pages-isolated`, and `codex/dogfood-release-binary-isolated` in their real `/private/tmp` worktrees and have the agents append retest sections. Keep all work local: no pushes, tags, PRs, or deployments. Run `go test ./...` and `go vet ./...` with system-level Git/cache/network handling as needed, and preserve exact report evidence.

## Status
Implementation complete on `codex/release-binary-contract`. The CLI/runtime
boundary, embedded fallback assets, cache/publication policy, artifact checker,
RAG output isolation, Pages/action workflows, release packaging, tests, and
operator documentation are committed. `go test ./...` and `go vet ./...`
pass. The same-name dogfood retests are complete: local static `512de18`,
GitHub Pages `e9ec01a`, and release binary `27c90db`; each branch preserved its
original report and added only its retest section plus the cherry-picked
implementation commit.
