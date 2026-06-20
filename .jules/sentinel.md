## 2023-10-27 - [Prevent Path Traversal in Link Transformation]
**Vulnerability:** Arbitrary file write due to path traversal when generating missing file stubs.
**Learning:** The application parsed Markdown links and resolved relative paths to generate HTML stubs for missing files. However, it did not restrict paths to the output directory, allowing paths like `../../../tmp/hack.md` to break out and write `.html` files elsewhere on the system.
**Prevention:** Use `filepath.IsLocal` to validate all resolved relative paths before writing files or treating them as missing files, ensuring they do not escape the intended directory boundaries.

## 2023-10-28 - [Prevent XSS in Missing File Stubs]
**Vulnerability:** Cross-Site Scripting (XSS) in dynamically generated HTML for missing pages.
**Learning:** When generating HTML stubs for missing Markdown files, the filenames of the "parent" pages that linked to the missing page were injected directly into the HTML without sanitization. A maliciously crafted parent filename (e.g., `<script>alert(1)</script>.md`) could execute arbitrary JavaScript. Even seemingly benign data like filenames can act as XSS vectors if they are reflected into HTML.
**Prevention:** Always sanitize unsanitized or user-influenced strings, including filenames, using `html.EscapeString` before injecting them into HTML templates or string builders.
## 2026-06-19 - Missing Page Stub Generation XSS
**Vulnerability:** A Cross-Site Scripting (XSS) vulnerability was found in the generation of missing page stubs. Malicious URLs (e.g., `javascript:alert(1)`) linked from markdown files could result in dangerous `href` attributes in the generated missing page HTML stubs.
**Learning:** We were relying entirely on `html.EscapeString` when injecting `relParent` and `parent` URLs into anchor tags for back-links. `html.EscapeString` only escapes characters like `<`, `>`, `&`, `"`, and `'`. It DOES NOT validate the scheme (like preventing `javascript:`).
**Prevention:** In addition to contextual escaping, all HTML templates or string builders that output full HTML must be passed through our `bluemonday` UGCPolicy (`p.SanitizeBytes()`) prior to writing or converting to `template.HTML`, ensuring that malicious schemes are scrubbed globally.
