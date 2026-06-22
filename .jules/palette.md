## 2025-02-23 - Skip-to-content and Semantic Navigation
**Learning:** Found that the default layout templates used a generic `div` for the navigation bar and lacked a skip-to-content link, which significantly degraded the keyboard navigation and screen reader experience.
**Action:** Added a `nav` element with `aria-label="Main Navigation"`, an `id="main-content"` on the `main` tag, and a visually hidden `Skip to content` link using Tailwind utilities (`sr-only focus:not-sr-only`) at the top of `<body>`. Will ensure future templates incorporate these semantic and accessible patterns by default.

## 2025-02-23 - Focus States and Decorative Icons in Custom Themes
**Learning:** The `cyberpunk.html` theme had custom hover states for sidebar navigation links (`hover:bg-secondary hover:text-secondary-content border-2 border-transparent hover:border-primary`) but lacked corresponding `focus-visible` utilities, making keyboard navigation hard to track. Furthermore, the inline SVGs lacked `aria-hidden="true"` and `focusable="false"`, adding noise for screen reader users.
**Action:** Always ensure that custom `hover` states have matching `focus-visible` states (e.g., `focus-visible:bg-secondary focus-visible:text-secondary-content focus-visible:border-primary focus-visible:outline-none`). Additionally, all decorative SVG icons used alongside text labels must include `aria-hidden="true"` and `focusable="false"` to optimize the screen reader experience.

## 2026-06-19 - Add missing focus-visible states to template links
**Learning:** Keyboard accessibility is compromised when custom `hover` states (like `hover:underline` or `hover:bg-primary/20`) are added without corresponding `focus-visible` states. Default browser focus rings might not provide sufficient contrast or might conflict with custom hover styling.
**Action:** Always pair interactive `hover` state utilities with matching `focus-visible` equivalents (e.g., `focus-visible:outline-none focus-visible:bg-primary/20`) to ensure custom styling is visible and functional for keyboard users.

## 2026-06-20 - Default Layout Focus Accessibility
**Learning:** The default layout in `templates/layout.html` lacked keyboard focus visibility for main navigation, "skip to content" link, and article links, making the site difficult to navigate via keyboard despite having semantic HTML in place.
**Action:** When creating or maintaining layout templates, always explicitly define `focus-visible` states using Tailwind utilities (e.g., `focus-visible:ring-2`, `focus-visible:outline`, `focus-visible:prose-a:outline`) for all interactive elements, including utility links like "Skip to content".
## 2026-06-19 - Tailwind Typography State Modifier Ordering
**Learning:** When using state modifiers (like `hover:` or `focus-visible:`) in combination with Tailwind Typography element modifiers (like `prose-a:`), the state modifier must come *after* the element modifier (e.g., `prose-a:focus-visible:outline`). If the state modifier is placed first (e.g., `focus-visible:prose-a:outline`), the state variant is applied to the parent element (`.prose`) instead of the child element (`<a>`). Additionally, DaisyUI 4 removed `-focus` color modifier classes, so base color modifiers or opacities must be used for focus states instead.
**Action:** Always verify modifier order when applying interaction styles to typography children to ensure proper a11y focus states are visually rendering on the targeted element.

## 2026-06-21 - Dashboard Action Button Context
**Learning:** Action buttons in dense, utility-focused layouts like dashboards can lack context without labels or surrounding descriptions. Additionally, relying solely on custom CSS focus rings can result in inconsistent keyboard navigation experiences if not explicitly styled.
**Action:** When adding utility or action buttons (like Export or Share) to dashboard headers, wrap them in DaisyUI tooltip components (`<div class="tooltip" data-tip="...">`) to provide clear, immediate context to users. Always ensure these buttons also explicitly define `focus-visible` states matching the design system.
