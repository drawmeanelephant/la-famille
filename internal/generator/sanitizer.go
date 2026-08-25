package generator

import (
	"regexp"

	"github.com/microcosm-cc/bluemonday"
)

// checkboxInputType limits the <input> elements the sanitizer keeps to the
// disabled checkboxes emitted by GFM task lists.
var checkboxInputType = regexp.MustCompile(`^checkbox$`)

var (
	imgLoadingPolicy  = regexp.MustCompile(`^(lazy|eager)$`)
	imgDecodingPolicy = regexp.MustCompile(`^(async|sync|auto)$`)
)

// newContentSanitizer builds the policy applied to markdown-derived HTML.
// It starts from bluemonday's UGC baseline and re-enables exactly the
// elements and attributes this project's pipeline emits beyond that
// baseline: class hooks, inline SVG icons, GFM task-list checkboxes, and
// block-level figures with their loading hints.
func newContentSanitizer() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("class").Globally()
	p.AllowElements("svg", "path")
	p.AllowAttrs("xmlns", "fill", "viewBox", "stroke-linecap", "stroke-linejoin", "stroke-width", "d", "stroke", "class").OnElements("svg", "path")
	// GFM task lists carry their state in a disabled checkbox input. Stripping
	// it makes a completed item render exactly like a pending one, so allow
	// precisely that element and nothing else about <input>.
	p.AllowAttrs("type").Matching(checkboxInputType).OnElements("input")
	p.AllowAttrs("checked", "disabled").OnElements("input")

	// Standalone images are promoted to <figure> blocks by the markdown
	// pipeline; keep the semantic wrapper and the performance hints.
	p.AllowElements("figure", "figcaption")
	p.AllowAttrs("loading").Matching(imgLoadingPolicy).OnElements("img")
	p.AllowAttrs("decoding").Matching(imgDecodingPolicy).OnElements("img")

	return p
}
