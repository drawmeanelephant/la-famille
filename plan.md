## Task: Extracted search JS to standalone file and enhanced

1. Created a standalone vanilla JS file `assets/js/search.js` that implements the search features:
   - Global `/` keydown listener to focus `#site-search`.
   - `{once: true}` focus listener on `#site-search` to fetch `/search.json` and cache it.
   - Debounced (50ms) input listener on `#site-search` that filters the index based on `t`, `g`, and `s` properties.
   - Securely generates innerHTML using an HTML escaper to prevent DOM XSS on user content.
2. Updated layout files (`templates/layout.html`, `templates/layout-dashboard.html`) to include the standalone script and implement proper element IDs (`#site-search`, `#search-results-list`).
3. Added styling adjustments (e.g. `relative` wrapper class where necessary) in `templates/layout-dashboard.html` to properly support the absolute positioned dropdown.
4. Updated all required HTML expected output test fixtures under `assets/testdata/sites/*/expected` to account for the new layout structure.
5. Successfully ran all tests and visually verified functionality using Playwright screenshots.

### Potential Breaking Changes:
- Previously inline script logic was removed and relies on `assets/js/search.js`.
- Dropdown ID changed from `search-results` to `search-results-list`.
