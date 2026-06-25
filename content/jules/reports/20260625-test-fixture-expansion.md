---
Title: Test Fixture Expansion Report
Date: 2026-06-25
Routine: Test Fixture Expansion
Status: Success
---
# Test Fixture Expansion Report

Added a new test fixture for testing URL fragments and query parameters inside markdown links.
This handles cases like `[Link](#section)`, `[Link](other.md#section)`, and `[Link](other.md?v=1)`.
It successfully creates the expected HTML files with the suffix `.html` replacing `.md` but leaving fragments and queries intact.
