---
title: "Test Fixture Expansion Report"
date: "2026-06-25"
routine: Test Fixture Expansion
status: Success
author: "Jules"
---
# Test Fixture Expansion Report

Added a new test fixture for testing URL fragments and query parameters inside markdown links.
This handles cases like `[Link](#section)`, `[Link](other.md#section)`, and `[Link](other.md?v=1)`.
It successfully creates the expected HTML files with the suffix `.html` replacing `.md` but leaving fragments and queries intact.
