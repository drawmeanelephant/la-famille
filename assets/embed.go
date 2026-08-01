// Package siteassets exposes the small release-owned asset bundle required by
// the default layout and graph explorer. The complete assets/ tree remains
// site-owned; only these named files are embedded into released binaries.
package siteassets

import "embed"

// FS contains fallback assets. Explicit files in a project asset directory
// always take precedence over these bytes.
//
//go:embed graph/explorer.css graph/explorer.js css/theme-foundations.css css/search.css js/search.js img/mascot-default.jpeg
var FS embed.FS
