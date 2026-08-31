// Package siteassets exposes the small release-owned asset bundle required by
// the default layout and graph explorer. The complete assets/ tree remains
// site-owned; only these named files are embedded into released binaries.
package siteassets

import "embed"

// FS contains fallback assets. Explicit files in a project asset directory
// always take precedence over these bytes.
//
//go:embed graph/explorer.css graph/explorer.js css/theme-foundations.css css/theme.css css/layout-editorial.css css/layout-midnight.css css/layout-terminal.css css/search.css js/search.js img/mascot-default.jpeg img/jules-logo.png img/u1f419_u1f354.png
var FS embed.FS
