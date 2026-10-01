package render_test

import (
	"html/template"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/page"
	"github.com/tbuddy/la-famille/internal/render"
)

func TestFlagshipLayoutsKeepContentAndAccessibleNavigation(t *testing.T) {
	dir := getTemplatesDir(t)
	layouts, err := render.DiscoverLayouts(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, layout := range []string{"layout-site", "layout-site-home"} {
		t.Run(layout, func(t *testing.T) {
			if !layouts[layout] {
				t.Fatalf("missing flagship layout %s", layout)
			}
			cfg := config.DefaultConfig()
			cfg.Template = filepath.Join(dir, "layout-site.html")
			p := page.Page{
				Title:        "Small files. Big possibilities.",
				Description:  "A local-first static site generator.",
				CanonicalURL: "https://la-famille.filed.fyi/",
				Content:      template.HTML(`<h2>Real Markdown content</h2><p>Keep this searchable.</p>`),
				Site:         cfg,
			}
			for _, base := range []string{"https://la-famille.filed.fyi", "https://example.com/la-famille"} {
				cfg.SiteURL = base
				raw := renderLayoutToBuffer(t, render.New(dir), dir, layout, p, cfg)
				for _, want := range []string{
					"Keep this searchable.", `id="main-content"`, `href="#main-content"`,
					`id="site-search"`, `aria-label="Main navigation"`,
					cfg.BasePath() + "/assets/css/site.css",
					cfg.BasePath() + "/assets/js/search.js",
					cfg.BasePath() + "/docs/", cfg.BasePath() + "/showcase/",
					`rel="canonical" href="https://la-famille.filed.fyi/"`,
				} {
					if !strings.Contains(raw, want) {
						t.Errorf("missing %q", want)
					}
				}
				doc := parseHTMLDocument(t, raw)
				h1s := collectNodes(doc, func(n *html.Node) bool {
					return n.Type == html.ElementNode && n.Data == "h1"
				})
				if len(h1s) != 1 {
					t.Errorf("expected one h1, got %d", len(h1s))
				}
				if strings.Contains(raw, `src="https://`) || strings.Contains(raw, `href="https://fonts.`) {
					t.Error("flagship layout must not fetch remote runtime assets")
				}
			}
		})
	}
}

func TestFlagshipDocumentationReusesExistingMarkdownTitle(t *testing.T) {
	dir := getTemplatesDir(t)
	cfg := config.DefaultConfig()
	cfg.Template = filepath.Join(dir, "layout-site.html")
	for _, content := range []template.HTML{
		`<h1>Existing guide title</h1><p>Original guide content.</p>`,
		`<h1 id="guide">Existing guide title</h1><p>Original guide content.</p>`,
		`<h2>A new guide</h2><p>Original guide content.</p>`,
	} {
		p := page.Page{Title: "Metadata title", Content: content, Site: cfg}
		raw := renderLayoutToBuffer(t, render.New(dir), dir, "layout-site", p, cfg)
		doc := parseHTMLDocument(t, raw)
		headings := collectNodes(doc, func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.Data == "h1"
		})
		if len(headings) != 1 {
			t.Errorf("expected one h1 for %q, got %d", content, len(headings))
		}
		if !strings.Contains(raw, "Original guide content.") {
			t.Error("existing content was lost")
		}
	}
}
