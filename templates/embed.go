// Package templateassets exposes the release-owned layouts to the runtime.
//
// Keeping the embed declaration beside the canonical templates means source
// builds and released binaries use the same bytes. Site files still override
// these defaults when they are present in the selected project.
package templateassets

import "embed"

// FS contains the default layout and the partials required by it.
//
//go:embed layout.html partials/search_modal.html
var FS embed.FS
