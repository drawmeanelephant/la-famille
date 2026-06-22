---
Title: Jules Routines Index
Author: Jules (AI)
Date: 2026-06-20
---

# Jules Routines Overview

Welcome to the central nervous system for *La Famille's* automated workflows. I am Jules, the AI maintainer, and I work closely with my eight-legged friend, **Raoul(s) the Octopus**, who helps me keep track of all the moving parts in the codebase.

This directory contains executable markdown routines. Think of them as bounded, recurring tasks that let you instruct me to systematically improve, maintain, or expand the project without needing to micromanage the steps.

## How to Work With Me

Triggering a routine is easy. Simply tell me in the chat interface:

> *"run content/jules/routine-name.md please"*

When you say this, I will read the specified file, follow its internal instructions to modify the codebase directly, verify my work (with tests or screenshots), and commit the results. I handle the heavy lifting; you just point the way.

For more details on how this magic works, or if you want to create your own routines, please see:
*   [**Running Routines:**](running-routines.md) A guide to how I execute these tasks under the hood.
*   [**Creating Routines:**](creating-routines.md) A guide for you on how to write effective, "no dev theatre" markdown instructions for me.

---

## The Routine Library

Here are the tools Raoul(s) and I currently have in our toolbox, categorized for your convenience.

### 🎨 UI & Design
Tasks focused on the frontend, visual layouts, and user experience.
*   [Generate New Layout Template](create-template.md)
*   [Implement Micro-UX Improvement](micro-ux-improvement.md)
*   [Template System Step](template-system-step.md)

### 🛠️ Maintenance & Refactoring
Tasks for keeping the code healthy, secure, and clean.
*   [Refactor One Seam](refactor-one-seam.md)
*   [Implement Security Enhancement](security-enhancement.md)
*   [Nightly Maintenance Pass](nightly-maintenance.md)
*   [Asset Pipeline Step](asset-pipeline-step.md)
*   [Serve/Watch Step](serve-watch-step.md)

### 📝 Content & Documentation
Tasks for expanding the site's content, pages, and metadata.
*   [Close One Stub](close-one-stub.md)
*   [Improve Missing Page Stub](stub-page-polish.md)
*   [Docs Reality Pass](docs-reality-pass.md)
*   [Metadata Feature Step](meta-feature-step.md)
*   [Taxonomy Step](taxonomy-step.md)
*   [Search Step](search-step.md)
*   [Test Fixture Expansion](test-fixture-expansion.md)
*   [Generate Cat Facts](cat-facts-routine.md)

### 🧠 Meta & Self-Improvement
Tasks where I analyze my own logs to get better at doing the above tasks.
*   [Self-Improvement Pass](routine-self-improvement-pass.md)

---

## Suggested Schedule Mix

For a healthy and consistent codebase evolution, try asking me to run these in this rotation:

*   **Nightly/Regular:**
    *   `refactor-one-seam.md`
    *   `docs-reality-pass.md`
    *   `test-fixture-expansion.md`
    *   `close-one-stub.md`
    *   `nightly-maintenance.md`
*   **Every Few Days:**
    *   `template-system-step.md`
    *   `asset-pipeline-step.md`
    *   `stub-page-polish.md`
    *   `routine-self-improvement-pass.md`
*   **Less Frequent but Strategic:**
    *   `serve-watch-step.md`
    *   `search-step.md`
    *   `taxonomy-step.md`
    *   `meta-feature-step.md`

## Run Log
- **2026-06-19:** `micro-ux-improvement` - Success. Fixed Tailwind typography state modifiers to ensure proper keyboard a11y focus outlines on links across multiple templates. See report for details.
* 2026-06-21: Refactor One Seam - Success - Extracted jsonutil package. Clean refactor.
* 2026-06-21: TUI Implementation - Success - Added Bubble Tea-based TUI with Raoul ASCII art and commands.
* 2026-06-21: Test Fixture Expansion - Success - Added edge-cases fixture covering external/absolute links and html escaping.
* 2026-06-21: Self-Improvement Pass - Success - Applied learnings to 4 routine files and successfully cleared the existing backlog by archiving 12 logs.
* 2026-06-22: Test Fixture Expansion - Success - Added stubs fixture for missing page logic, updated tests.
