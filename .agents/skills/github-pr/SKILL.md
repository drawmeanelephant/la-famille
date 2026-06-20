---
name: github-pr-management
description: Handle Git syncing, committing, and GitHub PR creation, cleanup, and merging. Trigger when the user wants to "sync GitHub", "clear the litterbox", or "create a PR".
---

# GitHub PR Management & Syncing

This skill provides a standardized workflow for managing Git changes and GitHub Pull Requests within the `la-famille` project. Your goal is to handle these operations efficiently and silently, without requiring the user to spend excessive AI context.

## 1. Syncing and Committing (Local Changes)
When asked to sync or commit:
1. Check the current status: `git status`
2. If there are changes, add them: `git add .`
3. Commit with a concise, descriptive message: `git commit -m "Auto-commit: [brief description]"`
4. Pull latest remote changes (rebase to avoid merge commits if possible): `git pull origin main --rebase`
5. Push to the remote branch: `git push origin HEAD`

## 2. Creating a Pull Request
When asked to create a PR for current changes:
1. Ensure changes are committed and pushed to a new branch first:
   `git checkout -b feature/auto-update`
   `git push -u origin feature/auto-update`
2. Use the GitHub CLI (`gh`) to create the PR:
   `gh pr create --title "Automated Update: [Topic]" --body "This PR was generated automatically to [brief reason]."`
3. If `gh` is unavailable, use standard `git push` and inform the user to click the PR link.

## 3. "Clearing the Litterbox" (Cleaning up PRs)
When the user asks to "clear the litterbox" or manage existing PRs:
1. List open PRs: `gh pr list`
2. Identify stale, redundant, or easily mergeable automated PRs.
3. To merge an approved/safe PR: `gh pr merge <PR-number> --squash --delete-branch`
4. To close a redundant PR: `gh pr close <PR-number>`
5. Run `git fetch -p` to prune local tracking branches after remote cleanup.
6. Return to the `main` branch: `git checkout main` and `git pull`.

## Important Rules
- Keep output concise. Only report the final status to the user (e.g., "Changes pushed to PR #12" or "Litterbox cleared: 2 PRs closed, 1 merged").
- Always be cautious when force-pushing or closing PRs. If a PR has human comments or review requests, pause and ask the user before closing it.
