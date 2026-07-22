---
date: "2026-07-09"
title: "Search Implementation Guide"
author: "Jules"
---

# Client-Side Search Implementation

La Famille includes a built-in, lightning-fast client-side search functionality. This guide explains how to integrate it into your site's layouts.

## Overview

During the build process, the generator parses all markdown content and creates a highly compressed, deterministic `search.json` file in the root of the output directory. This file contains a minified array of all rendered pages (`render: false` pages are automatically excluded).

Each item in `search.json` includes:
*   `t`: Page title (or fallback filename)
*   `u`: Page relative URL (validated against slugs)
*   `g`: Combined taxonomy metadata terms (tags and categories)
*   `s`: Cleaned plaintext content excerpt snippet (up to 160 characters)
*   `h`: Extracted ATX heading titles (# to ######)

The client-side JavaScript (`assets/js/search.js`) fetches this file, caches it in memory, and provides instant, debounce-optimized search results as you type.

## Integration Steps

To add the search functionality to your site, follow these steps:

### 1. Include the JavaScript

Add the following script tag just before the closing `</body>` tag in your layout template (e.g., `templates/layout.html`):

```html
<script src="/assets/js/search.js"></script>
```

### 2. Add the Search Markup

Add the following HTML markup to your navigation bar or header to provide the search input and results dropdown. Note the `id` attributes and `aria` roles, which are required by the JavaScript logic:

```html
<div class="relative max-w-xs w-full">
    <form action="#" method="get" class="m-0" onsubmit="event.preventDefault();">
        <label for="site-search" class="sr-only">Search site</label>
        <input type="search" id="site-search" placeholder="Type / to search..."
               class="input input-bordered w-full pr-10 focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary"
               autocomplete="off" aria-autocomplete="list" aria-controls="search-results-list" />
    </form>
    <ul id="search-results-list" role="listbox" aria-label="Search results"
        class="absolute z-50 left-0 right-0 mt-2 max-h-60 overflow-y-auto hidden bg-base-200 border border-base-300 rounded-box shadow-xl p-2 text-sm space-y-1">
    </ul>
</div>
```

## How It Works

*   **Multi-Signal Matching:** The client filters across title (`t`), taxonomy metadata (`g`), content snippet (`s`), and headings (`h`).
*   **Rich UI Rendering:** Results display titles, snippets, matched heading section badges, and taxonomy tag badges.
*   **Keyboard Shortcut:** The search input can be quickly focused from anywhere on the page by pressing the `/` key.
*   **Lazy Loading:** To conserve bandwidth, `search.json` is only fetched the first time the search input receives focus. It is then cached in `window.LaFamilleSearchIndex`.
*   **Debouncing:** The search query is debounced by 50ms to prevent excessive filtering during rapid typing.
*   **Security:** All user input and search result snippets are carefully escaped before being inserted into the DOM to prevent Cross-Site Scripting (XSS) vulnerabilities.
