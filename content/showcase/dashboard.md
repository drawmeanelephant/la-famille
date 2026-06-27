---
title: "The Janitor's Command Center (Dashboard Layout Showcase)"
layout: layout-dashboard
---

## The Command Center

The Janitor's command center is where the meticulous cleaning of the repository is monitored. Every old branch artifact in the "litterbox" and every rogue `serve.log` file is tracked here.

### Daily Metrics

Monitoring the daily sweep:

- Swept items today
  - `serve.log` instances: 42
  - `.DS_Store` files: 15
  - `/tmp` debris: 120MB
- Litterbox status
  1. Branches pruned: 12
  2. Stale PRs closed: 3
  3. Merge conflicts resolved: 0

> "You call it a messy git history. I call it job security. But seriously, rebase your branches." — The Janitor

### Persona Tracking

| Persona | Responsibility | Associated Image Asset |
|---|---|---|
| The Janitor | Cleaning the repo, clearing branches | `Octopus_mascot_cleaning_litterbox_202606200817.jpeg` |
| The Skater | Shredding the CI/CD pipeline | `Octopus_riding_skateboard_holdin…_202606200817_2.jpeg` |
| The Maestro | Curating Flow Music | `Octopus_mascot_writing_music_dia…_202606200817.jpeg` |

### XSS Prevention

The dashboard must display dynamic data safely. The Janitor ensures URLs are sanitized:

```js
// Sanitize dynamic URLs in the dashboard
function renderDashboardLink(url, label) {
    const safeUrl = encodeURI(url);
    return `<a href="${safeUrl}">${label}</a>`;
}
```
