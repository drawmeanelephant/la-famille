## 2025-02-23 - Skip-to-content and Semantic Navigation
**Learning:** Found that the default layout templates used a generic `div` for the navigation bar and lacked a skip-to-content link, which significantly degraded the keyboard navigation and screen reader experience.
**Action:** Added a `nav` element with `aria-label="Main Navigation"`, an `id="main-content"` on the `main` tag, and a visually hidden `Skip to content` link using Tailwind utilities (`sr-only focus:not-sr-only`) at the top of `<body>`. Will ensure future templates incorporate these semantic and accessible patterns by default.

## 2025-02-23 - Focus States and Decorative Icons in Custom Themes
**Learning:** The `cyberpunk.html` theme had custom hover states for sidebar navigation links (`hover:bg-secondary hover:text-secondary-content border-2 border-transparent hover:border-primary`) but lacked corresponding `focus-visible` utilities, making keyboard navigation hard to track. Furthermore, the inline SVGs lacked `aria-hidden="true"` and `focusable="false"`, adding noise for screen reader users.
**Action:** Always ensure that custom `hover` states have matching `focus-visible` states (e.g., `focus-visible:bg-secondary focus-visible:text-secondary-content focus-visible:border-primary focus-visible:outline-none`). Additionally, all decorative SVG icons used alongside text labels must include `aria-hidden="true"` and `focusable="false"` to optimize the screen reader experience.

## 2026-06-19 - Add missing focus-visible states to template links
**Learning:** Keyboard accessibility is compromised when custom `hover` states (like `hover:underline` or `hover:bg-primary/20`) are added without corresponding `focus-visible` states. Default browser focus rings might not provide sufficient contrast or might conflict with custom hover styling.
**Action:** Always pair interactive `hover` state utilities with matching `focus-visible` equivalents (e.g., `focus-visible:outline-none focus-visible:bg-primary/20`) to ensure custom styling is visible and functional for keyboard users.
