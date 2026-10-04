package render_test

import (
	"html/template"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/page"
	"github.com/tbuddy/la-famille/internal/render"
)

func TestFlagshipHomeWelcomesRaoulsWithoutRemovingBurgers(t *testing.T) {
	dir := getTemplatesDir(t)
	cfg := config.DefaultConfig()
	cfg.Template = filepath.Join(dir, "layout-site.html")
	p := page.Page{Title: "La Famille", Site: cfg}
	for _, base := range []string{"https://la-famille.filed.fyi", "https://example.com/la-famille"} {
		t.Run(base, func(t *testing.T) {
			cfg.SiteURL = base
			raw := renderLayoutToBuffer(t, render.New(dir), dir, "layout-site-home", p, cfg)
			doc := parseHTMLDocument(t, raw)
			for _, want := range []string{
				`id="raouls-title"`, "Meet Raoul(s)", "Multipersonality Octodeveloper",
				"The Janitor", "The Maestro", "The Skater",
				`href="` + cfg.BasePath() + `/meta/mascot-raouls/"`,
				`rel="icon" href="` + cfg.BasePath() + `/assets/img/u1f419_u1f354.png"`,
				"I'll hold the burger.", "Eight arms. One burger.",
			} {
				if !strings.Contains(raw, want) {
					t.Errorf("missing mascot or preserved burger content %q", want)
				}
			}
			for _, image := range []string{
				"mascot-default.jpeg",
				"Octopus_mascot_cleaning_litterbox_202606200817.jpeg",
				"Octopus_mascot_writing_music_dia…_202606200817.jpeg",
				"Octopus_mascot_riding_skateboard…_202606200817.jpeg",
				"u1f419_u1f354.png",
			} {
				path := cfg.BasePath() + "/assets/img/" + image
				images := collectNodes(doc, func(n *html.Node) bool {
					if n.Type != html.ElementNode || n.Data != "img" {
						return false
					}
					for _, attr := range n.Attr {
						if attr.Key == "src" {
							src, err := url.PathUnescape(attr.Val)
							return err == nil && src == path
						}
					}
					return false
				})
				if image == "u1f419_u1f354.png" {
					if len(images) < 3 {
						t.Errorf("keep all three existing burger placements, got %d", len(images))
					}
					continue
				}
				if len(images) != 1 {
					t.Errorf("expected one original %s illustration, got %d", image, len(images))
				}
				for _, img := range images {
					attrs := make(map[string]string)
					for _, attr := range img.Attr {
						attrs[attr.Key] = attr.Val
					}
					if attrs["alt"] == "" || attrs["width"] == "" || attrs["height"] == "" ||
						attrs["loading"] != "lazy" {
						t.Errorf("mascot image needs alt text, dimensions, and lazy loading: %s", image)
					}
				}
			}
		})
	}
}

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
					cfg.BasePath() + "/docs/corpus-packs/",
					`rel="canonical" href="https://la-famille.filed.fyi/"`,
				} {
					if !strings.Contains(raw, want) {
						t.Errorf("missing %q", want)
					}
				}
				for _, retired := range []string{"/docs/ask/", "Ollama", "Local Ask", "Ask this site"} {
					if strings.Contains(raw, retired) {
						t.Errorf("layout still advertises retired assistant %q", retired)
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
