---
title: Routine - Test Fixture Expansion Report
date: 2026-06-26
---

# Routine Completed: Test Fixture Expansion

**Status:** Success
**Date:** 2026-06-26

**Details:**
Added a new test fixture (`query-fragments`) to cover link handling edge cases where Markdown links contain query parameters and URL fragments (e.g., `page.md?q=1#section`). This ensures the `link_transformer` correctly rewrites the extensions to `.html` while preserving the URL queries and fragments. The test protects against regression in link parsing logic.
