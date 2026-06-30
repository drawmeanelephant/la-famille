1.  **Replace Mobile Drawer SVG with 🍔 Emoji**
    *   Target templates: `layout-dashboard.html`, `layout-sidebar.html`, `layout-drawer.html`, `layout-documentation.html`, `layout-floating-cards.html` (any that use the `d="M4 6h16M4 12h16M4 18h16"` or similar drawer icon SVG).
    *   Find the SVG block wrapping `<path d="M4 6h16..."></path></svg>`.
    *   Replace it with `<span aria-hidden="true" class="text-2xl">🍔</span>`. Ensure the parent button retains `aria-label="Open Sidebar"` (or similar).

2.  **Add Compliance Modal Support**
    *   **Backend:** Add `ComplianceModal string` to `internal/page/Page`, `FileMeta` in `internal/content/metadata.go`, and the anonymous struct in `GatherMetadata`. Also, update `internal/generator/generator.go` to assign `ComplianceModal: meta.ComplianceModal`.
    *   **Frontend (`templates/layout.html`):** Add the DaisyUI modal structure at the end of the `<body>`:
        ```html
        {{if .ComplianceModal}}
        <input type="checkbox" id="compliance-modal" class="modal-toggle" checked />
        <div class="modal" role="dialog">
            <div class="modal-box">
                <h3 class="font-bold text-lg">Compliance Required</h3>
                <p class="py-4">{{.ComplianceModal}}</p>
                <div class="modal-action">
                    <label for="compliance-modal" class="btn btn-primary">I Reluctantly Agree</label>
                </div>
            </div>
        </div>
        {{end}}
        ```

3.  **Update Test Data/Frontmatter**
    *   Update `content/showcase/layout.md` frontmatter to include `compliance_modal: "By reading this demo, you agree to the cookie treaty of 1842."`.

4.  **Testing and Verification**
    *   Run `go test ./...` and `go vet ./...`.
    *   Run the site generation `go run ./cmd/la-famille build`.

5.  **Write Routine Log**
    *   Create a markdown file in `content/jules/reports/` describing the completion of the routine.

6.  **Pre-commit checks**
    *   Complete pre-commit steps to ensure proper testing, verification, review, and reflection are done.
