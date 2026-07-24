// Package graphexplorer renders a static Knowledge Graph Explorer page at
// <outputDir>/graph/index.html. The page loads the existing generated
// graph.json / backlinks.json / meta.json artifacts at runtime via relative
// fetches so it stays fully static and respects siteurl=unset builds.
//
// The HTML scaffolding is bundled via go:embed (from ./assets/template.html)
// so the binary is self-contained. JavaScript and CSS are stored under the
// project's assets/graph/ directory and are copied into <outputDir>/assets/
// by the established asset pipeline. This matches the configuration option
// "graph_explorer: false" being the only way to disable the explorer, and
// follows the same path as every other CSS/JS bundle in La Famille.
package graphexplorer

import (
	"embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/pathutil"
)

// LargeSiteThreshold is the node count above which the explorer falls back to
// a search-first default view rather than attempting a dense all-node render.
// The threshold is intentionally generous; sites comfortably fit on screen
// below this size. The threshold is exposed in the page footer so it can be
// tuned without a code change.
const LargeSiteThreshold = 500

//go:embed assets/template.html
var templateFS embed.FS

// Result summarizes what Write produced. The fields are stable so tests can
// assert on the produced artifact locations.
type Result struct {
	IndexPath string
	NodeCount int
	Disabled  bool
}

// IndexPath returns the absolute output path Write produces for the explorer
// page. Centralized so tests don't depend on path concatenation internals.
func IndexPath(outputDir string) string {
	return filepath.Join(filepath.Clean(outputDir), "graph", "index.html")
}

// IndexRel is the slash-separated public URL of the explorer page.
func IndexRel() string { return "graph/index.html" }

// AssetRel is the relative URL the explorer page uses to reference its
// accompanying CSS / JS. Both files live next to each other on disk so a
// single relative path covers both stylesheet and runtime script.
func AssetRel() string { return "../assets/graph/explorer" }

// renderModel is the data passed to the embedded HTML template.
type renderModel struct {
	Title          string
	PageTitle      string
	Description    string
	CanonicalURL   string
	FooterNote     string
	SiteName       string
	AssetCSSURL    string
	AssetJSURL     string
	LargeThreshold int
}

// Write emits the graph explorer page at <outputDir>/graph/index.html when
// cfg.GraphExplorer is true. When the option is false, the function returns a
// Result with Disabled=true and writes nothing so callers can safely call it
// on every build.
//
// nodeCount is the number of normalized graph nodes from Graph.Nodes. The
// value drives the footer note describing large-site behavior.
func Write(cfg config.Config, nodeCount int) (Result, error) {
	res := Result{}
	if !cfg.GraphExplorer {
		res.Disabled = true
		return res, nil
	}

	tmplBytes, err := templateFS.ReadFile("assets/template.html")
	if err != nil {
		return res, fmt.Errorf("read graphexplorer template: %w", err)
	}
	tmpl, err := template.New("template.html").Parse(string(tmplBytes))
	if err != nil {
		return res, fmt.Errorf("parse graphexplorer template: %w", err)
	}

	outDirClean := filepath.Clean(cfg.OutputDir)
	indexPath := IndexPath(cfg.OutputDir)
	if !pathutil.IsSafePath(outDirClean, indexPath) {
		return res, fmt.Errorf("graphexplorer output escapes output directory: %s", indexPath)
	}
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o750); err != nil {
		return res, fmt.Errorf("create graphexplorer directory: %w", err)
	}

	siteName := cfg.SiteName
	if siteName == "" {
		siteName = "La Famille"
	}
	assetRel := AssetRel()
	model := renderModel{
		Title:          explorerTitle(siteName),
		PageTitle:      explorerTitle(siteName),
		Description:    explorerDescription(siteName),
		CanonicalURL:   cfg.URLForOutputPath(IndexRel()),
		FooterNote:     explorerFooter(siteName, nodeCount),
		SiteName:       siteName,
		AssetCSSURL:    assetRel + ".css",
		AssetJSURL:     assetRel + ".js",
		LargeThreshold: LargeSiteThreshold,
	}

	out, err := os.OpenFile(indexPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return res, fmt.Errorf("create graphexplorer file: %w", err)
	}
	defer out.Close()

	if err := tmpl.Execute(out, model); err != nil {
		return res, fmt.Errorf("render graphexplorer template: %w", err)
	}

	res.IndexPath = indexPath
	res.NodeCount = nodeCount
	return res, nil
}

func explorerTitle(siteName string) string {
	return "Knowledge Graph — " + siteName
}

func explorerDescription(siteName string) string {
	return "Interactive explorer of all " + siteName + " pages and their internal links."
}

func explorerFooter(siteName string, nodeCount int) string {
	parts := []string{
		"Knowledge Graph for " + siteName + ".",
		fmt.Sprintf("%d nodes.", nodeCount),
		fmt.Sprintf("Sites above %d nodes default to search-first so the visualization stays readable.", LargeSiteThreshold),
	}
	return joinSpaces(parts)
}

// joinSpaces joins with single space, trimming repeats for clean footer text.
func joinSpaces(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " "
		}
		out += p
	}
	return out
}
