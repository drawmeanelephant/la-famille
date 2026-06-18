## 2023-10-27 - [Prevent Path Traversal in Link Transformation]
**Vulnerability:** Arbitrary file write due to path traversal when generating missing file stubs.
**Learning:** The application parsed Markdown links and resolved relative paths to generate HTML stubs for missing files. However, it did not restrict paths to the output directory, allowing paths like `../../../tmp/hack.md` to break out and write `.html` files elsewhere on the system.
**Prevention:** Use `filepath.IsLocal` to validate all resolved relative paths before writing files or treating them as missing files, ensuring they do not escape the intended directory boundaries.
