// Package graphexplorer renders a static Knowledge Graph Explorer page at
// <outputDir>/graph/index.html, alongside the payload it reads at
// <outputDir>/graph/data.json.
//
// The payload is assembled in Go from the same graph the build already
// computed, so link direction, node classification, and public URLs are
// resolved once, server-side, where they are covered by tests. The browser
// renders that payload rather than re-deriving it from the generic graph.json /
// meta.json / backlinks.json artifacts, which stay untouched for other
// consumers.
//
// The HTML scaffolding is bundled via go:embed (from ./assets/template.html) so
// the binary is self-contained. JavaScript and CSS are stored under the
// project's assets/graph/ directory and are copied into <outputDir>/assets/ by
// the established asset pipeline.
package graphexplorer

import (
	"embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/graph"
	"github.com/tbuddy/la-famille/internal/jsonutil"
	"github.com/tbuddy/la-famille/internal/pathutil"
)

// LargeSiteThreshold is the node count at or above which the explorer opens in
// search-first mode: the visualization is not drawn until the reader picks a
// page, so a large site does not pay for rendering every node up front. The
// threshold is surfaced in the page footer so it can be tuned here without
// hunting through the client code.
const LargeSiteThreshold = 500

//go:embed assets/template.html
var templateFS embed.FS

// Input is everything Write needs to assemble the explorer.
type Input struct {
	Config config.Config
	Graph  graph.Graph
	// Meta is the per-page frontmatter map the build collected, keyed by page id.
	Meta map[string]map[string]interface{}
	// PageOutputs maps a page id to the output-relative path its HTML was
	// written to, which is what makes emitted URLs slug-aware.
	PageOutputs map[string]string
}

// Result summarizes what Write produced. The fields are stable so tests can
// assert on the produced artifact locations.
type Result struct {
	IndexPath string
	DataPath  string
	NodeCount int
	Disabled  bool
}

// IndexPath returns the absolute output path Write produces for the explorer
// page. Centralized so tests don't depend on path concatenation internals.
func IndexPath(outputDir string) string {
	return filepath.Join(filepath.Clean(outputDir), "graph", "index.html")
}

// DataPath returns the absolute output path of the explorer's data payload.
func DataPath(outputDir string) string {
	return filepath.Join(filepath.Clean(outputDir), "graph", "data.json")
}

// IndexRel is the slash-separated public URL of the explorer page.
func IndexRel() string { return "graph/index.html" }

// DataRel is the explorer page's relative URL for its data payload.
func DataRel() string { return "data.json" }

// AssetRel is the relative URL the explorer page uses to reference its
// accompanying CSS / JS. Both files live next to each other on disk so a
// single relative path covers both stylesheet and runtime script.
func AssetRel() string { return "../assets/graph/explorer" }

// renderModel is the data passed to the embedded HTML template.
type renderModel struct {
	Title        string
	PageTitle    string
	Description  string
	CanonicalURL string
	FooterNote   string
	SiteName     string
	AssetCSSURL  string
	AssetJSURL   string
	DataURL      string
}

// Write emits the graph explorer page and its data payload when
// cfg.GraphExplorer is true. When the option is false the function returns a
// Result with Disabled=true and writes nothing, so callers can call it on every
// build.
func Write(in Input) (Result, error) {
	res := Result{}
	cfg := in.Config
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
	dataPath := DataPath(cfg.OutputDir)
	for _, path := range []string{indexPath, dataPath} {
		if !pathutil.IsSafePath(outDirClean, path) {
			return res, fmt.Errorf("graphexplorer output escapes output directory: %s", path)
		}
	}
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o750); err != nil {
		return res, fmt.Errorf("create graphexplorer directory: %w", err)
	}

	data := BuildData(cfg, in.Graph, in.Meta, in.PageOutputs)
	if err := jsonutil.WriteJSON(dataPath, data); err != nil {
		return res, fmt.Errorf("write graphexplorer data: %w", err)
	}

	siteName := cfg.SiteName
	if siteName == "" {
		siteName = "La Famille"
	}
	assetRel := AssetRel()
	model := renderModel{
		Title:        explorerTitle(siteName),
		PageTitle:    explorerTitle(siteName),
		Description:  explorerDescription(siteName),
		CanonicalURL: cfg.URLForOutputPath(IndexRel()),
		FooterNote:   explorerFooter(siteName, len(data.Nodes)),
		SiteName:     siteName,
		AssetCSSURL:  assetRel + ".css",
		AssetJSURL:   assetRel + ".js",
		DataURL:      DataRel(),
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
	res.DataPath = dataPath
	res.NodeCount = len(data.Nodes)
	return res, nil
}

func explorerTitle(siteName string) string {
	return "Knowledge Graph — " + siteName
}

func explorerDescription(siteName string) string {
	return "Interactive explorer of all " + siteName + " pages and their internal links."
}

func explorerFooter(siteName string, nodeCount int) string {
	return strings.Join([]string{
		"Knowledge Graph for " + siteName + ".",
		fmt.Sprintf("%d nodes.", nodeCount),
		fmt.Sprintf("Sites at or above %d nodes open in search-first mode so the visualization stays readable.", LargeSiteThreshold),
	}, " ")
}
