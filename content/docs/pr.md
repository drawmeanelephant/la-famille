---
date: "2026-07-09"
title: "Pull Request Management"
author: "Jules"
---

# Pull Request Management

La Famille includes a built-in command to help manage automated pull requests. This is especially useful for clearing out the "litterbox" of stale PRs created by automation agents.

## Automated PR Sync

The `pr sync` command fetches open pull requests created by automation agents, closes stale or conflicting ones, and merges passing ones. If there are local uncommitted changes, it can also create a new branch and open a new PR.

```bash
go run ./cmd/la-famille pr sync
```

### Requirements

To use the PR management features, you must have a GitHub personal access token exported in your environment variables:

```bash
export GITHUB_TOKEN="your_personal_access_token"
```

The token must have sufficient permissions to read and modify pull requests in your repository.

### Configuration

You can specify the base branch to target for new PRs using the `--base` flag.

*   `--base` (string): The base branch to target for new PRs. Defaults to `main`.

*Example:* `go run ./cmd/la-famille pr sync --base master`

**Note on branches:** Ensure you are targeting the correct primary branch for your project (e.g., `master` or `main`).
