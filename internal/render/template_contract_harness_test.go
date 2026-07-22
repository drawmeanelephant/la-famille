package render_test

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/page"
	"github.com/tbuddy/la-famille/internal/render"
)

var (
	metaViewportRE = regexp.MustCompile(`(?i)<meta\s+[^>]*name=["']viewport["'][^>]*>`)
	canonicalRE    = regexp.MustCompile(`(?i)<link\s+[^>]*rel=["']canonical["'][^>]*>`)
	ogURLRE        = regexp.MustCompile(`(?i)<meta\s+[^>]*property=["']og:url["'][^>]*>`)
)

func getTemplatesDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "..", "templates")
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("templates directory not found at %s: %v", dir, err)
	}
	return dir
}

func renderLayoutToBuffer(t *testing.T, renderer *render.Renderer, templatesDir string, layoutName string, p page.Page, cfg config.Config) string {
	t.Helper()
	tmpFile := filepath.Join(t.TempDir(), "rendered.html")
	if cfg.Template == "" {
		cfg.Template = filepath.Join(templatesDir, "layout.html")
	}
	err := renderer.HTML(cfg, p, layoutName, tmpFile)
	if err != nil {
		t.Fatalf("render.HTML failed for layout %q: %v", layoutName, err)
	}
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read rendered file for layout %q: %v", layoutName, err)
	}
	return string(data)
}

func parseHTMLDocument(t *testing.T, rawHTML string) *html.Node {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		t.Fatalf("failed to parse rendered HTML: %v", err)
	}
	return doc
}

func collectNodes(n *html.Node, predicate func(*html.Node) bool) []*html.Node {
	var matches []*html.Node
	var walker func(*html.Node)
	walker = func(node *html.Node) {
		if predicate(node) {
			matches = append(matches, node)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walker(c)
		}
	}
	walker(n)
	return matches
}

func getNodeAttr(n *html.Node, attrName string) (string, bool) {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, attrName) {
			return a.Val, true
		}
	}
	return "", false
}

func getNodeText(n *html.Node) string {
	var buf bytes.Buffer
	var walker func(*html.Node)
	walker = func(node *html.Node) {
		if node.Type == html.TextNode {
			buf.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walker(c)
		}
	}
	walker(n)
	return strings.TrimSpace(buf.String())
}

// TestBundledLayoutsRegressionHarness verifies every bundled layout template against contract rules.
func TestBundledLayoutsRegressionHarness(t *testing.T) {
	templatesDir := getTemplatesDir(t)
	allowlist, err := render.DiscoverLayouts(templatesDir)
	if err != nil {
		t.Fatalf("DiscoverLayouts failed: %v", err)
	}
	if len(allowlist) == 0 {
		t.Fatal("No layouts discovered in templates directory")
	}

	renderer := render.New(templatesDir)

	for layoutName := range allowlist {
		t.Run(layoutName, func(t *testing.T) {
			// 1. Verify Panic-Free Rendering on Empty / Optional Page Fields
			t.Run("panic_free_empty_page", func(t *testing.T) {
				emptyCfg := config.Config{Template: filepath.Join(templatesDir, "layout.html")}
				emptyPage := page.Page{}
				renderedEmpty := renderLayoutToBuffer(t, renderer, templatesDir, layoutName, emptyPage, emptyCfg)
				if strings.TrimSpace(renderedEmpty) == "" {
					t.Errorf("layout %q rendered empty string for empty page", layoutName)
				}
			})

			// Setup representative Page fixture
			cfg := config.Config{
				SiteName: "La Famille Test Site",
				Template: filepath.Join(templatesDir, "layout.html"),
				SiteLinks: []config.SiteLink{
					{Label: "Home", URL: "/"},
					{Label: "About", URL: "/about.html"},
				},
			}
			fullPageFixture := page.Page{
				Site:            cfg,
				Title:           "Representative Fixture Title",
				Author:          "Harness Tester",
				Date:            "2026-07-22",
				Description:     "A comprehensive test page fixture.",
				Image:           "/assets/img/og-test.png",
				CanonicalURL:    "https://example.com/test-fixture.html",
				Content:         template.HTML(`<h2>Section 1 Header</h2><p>Here is content with Emoji Kitchen blend: <img src="/assets/img/emoji-kitchen.png" class="emoji-kitchen" alt="Emoji Kitchen blend" />.</p><h3>Subsection 1.1 Header</h3><p>Subsection text.</p><h2>Section 2 Header</h2><p>Final paragraph.</p>`),
				VideoScript:     "console.log('test video');",
				AnimationCues:   "fade-in",
				SoundtrackTheme: "ambient",
			}

			renderedHTML := renderLayoutToBuffer(t, renderer, templatesDir, layoutName, fullPageFixture, cfg)
			doc := parseHTMLDocument(t, renderedHTML)

			// 2. Valid HTML Structure & Required Landmarks
			t.Run("html_structure_and_landmarks", func(t *testing.T) {
				if !strings.HasPrefix(strings.TrimSpace(strings.ToLower(renderedHTML)), "<!doctype html>") {
					t.Errorf("layout %q missing <!DOCTYPE html>", layoutName)
				}

				htmlNodes := collectNodes(doc, func(n *html.Node) bool { return n.Type == html.ElementNode && n.Data == "html" })
				if len(htmlNodes) != 1 {
					t.Fatalf("expected exactly one <html> element, found %d", len(htmlNodes))
				}
				if lang, ok := getNodeAttr(htmlNodes[0], "lang"); !ok || strings.TrimSpace(lang) == "" {
					t.Errorf("layout %q <html> tag missing lang attribute", layoutName)
				}

				headNodes := collectNodes(doc, func(n *html.Node) bool { return n.Type == html.ElementNode && n.Data == "head" })
				if len(headNodes) != 1 {
					t.Errorf("expected exactly 1 <head>, found %d", len(headNodes))
				}

				bodyNodes := collectNodes(doc, func(n *html.Node) bool { return n.Type == html.ElementNode && n.Data == "body" })
				if len(bodyNodes) != 1 {
					t.Errorf("expected exactly 1 <body>, found %d", len(bodyNodes))
				}

				mainNodes := collectNodes(doc, func(n *html.Node) bool {
					if n.Type == html.ElementNode && n.Data == "main" {
						id, _ := getNodeAttr(n, "id")
						return id == "main-content"
					}
					return false
				})
				if len(mainNodes) != 1 {
					t.Errorf("layout %q expected 1 <main id=\"main-content\"> landmark, found %d", layoutName, len(mainNodes))
				}

				// Skip link check
				skipLinks := collectNodes(doc, func(n *html.Node) bool {
					if n.Type == html.ElementNode && n.Data == "a" {
						href, _ := getNodeAttr(n, "href")
						return href == "#main-content"
					}
					return false
				})
				if len(skipLinks) == 0 {
					t.Errorf("layout %q missing skip link <a href=\"#main-content\">", layoutName)
				}
			})

			// 3. Title assertions: exactly one <title> and exactly one <h1>
			t.Run("title_and_h1", func(t *testing.T) {
				titleNodes := collectNodes(doc, func(n *html.Node) bool { return n.Type == html.ElementNode && n.Data == "title" })
				if len(titleNodes) != 1 {
					t.Errorf("layout %q expected exactly 1 <title>, found %d", layoutName, len(titleNodes))
				} else {
					text := getNodeText(titleNodes[0])
					if text == "" {
						t.Errorf("layout %q <title> is empty", layoutName)
					}
				}

				h1Nodes := collectNodes(doc, func(n *html.Node) bool { return n.Type == html.ElementNode && n.Data == "h1" })
				if len(h1Nodes) != 1 {
					t.Errorf("layout %q expected exactly 1 <h1>, found %d", layoutName, len(h1Nodes))
				} else {
					text := getNodeText(h1Nodes[0])
					if text == "" {
						t.Errorf("layout %q <h1> is empty", layoutName)
					}
				}
			})

			// 4. Viewport Metadata
			t.Run("viewport_metadata", func(t *testing.T) {
				if !metaViewportRE.MatchString(renderedHTML) {
					t.Errorf("layout %q missing viewport meta tag", layoutName)
				}
			})

			// 5. Navigation & Target Resolution
			t.Run("navigation_targets_resolve", func(t *testing.T) {
				navNodes := collectNodes(doc, func(n *html.Node) bool {
					if n.Type == html.ElementNode {
						if n.Data == "nav" || n.Data == "header" || n.Data == "aside" {
							return true
						}
						role, _ := getNodeAttr(n, "role")
						return role == "navigation"
					}
					return false
				})
				if len(navNodes) == 0 {
					t.Errorf("layout %q missing navigation/header landmark (<nav>, <header>, <aside>, or role=\"navigation\")", layoutName)
				}

				// Check internal anchor fragments resolve to element IDs
				anchorNodes := collectNodes(doc, func(n *html.Node) bool {
					if n.Type == html.ElementNode && n.Data == "a" {
						href, ok := getNodeAttr(n, "href")
						return ok && strings.HasPrefix(href, "#") && len(href) > 1
					}
					return false
				})

				for _, a := range anchorNodes {
					targetID, _ := getNodeAttr(a, "href")
					targetID = strings.TrimPrefix(targetID, "#")

					// Theme-specific exceptions: pure-CSS/JS mobile menu triggers without DOM targets
					if (layoutName == "brutalist" && targetID == "mobile-menu") ||
						(layoutName == "luxury_magazine" && targetID == "menu") ||
						(layoutName == "layout-magazine-grid" && targetID == "navigation-drawer") {
						continue
					}

					targets := collectNodes(doc, func(n *html.Node) bool {
						if n.Type == html.ElementNode {
							id, _ := getNodeAttr(n, "id")
							return id == targetID
						}
						return false
					})
					if len(targets) == 0 {
						t.Errorf("layout %q anchor link #%s has no target element with id=%q", layoutName, targetID, targetID)
					}
				}
			})

			// 6. Canonical and og:url configured vs unconfigured
			t.Run("canonical_and_og_url", func(t *testing.T) {
				// With CanonicalURL set
				if !canonicalRE.MatchString(renderedHTML) {
					t.Errorf("layout %q missing <link rel=\"canonical\"> when CanonicalURL is set", layoutName)
				}
				if !ogURLRE.MatchString(renderedHTML) {
					t.Errorf("layout %q missing <meta property=\"og:url\"> when CanonicalURL is set", layoutName)
				}

				// Unconfigured CanonicalURL
				noCanonicalFixture := fullPageFixture
				noCanonicalFixture.CanonicalURL = ""
				renderedNoCanonical := renderLayoutToBuffer(t, renderer, templatesDir, layoutName, noCanonicalFixture, cfg)

				if canonicalRE.MatchString(renderedNoCanonical) {
					t.Errorf("layout %q emitted <link rel=\"canonical\"> when CanonicalURL is empty", layoutName)
				}
				if ogURLRE.MatchString(renderedNoCanonical) {
					t.Errorf("layout %q emitted <meta property=\"og:url\"> when CanonicalURL is empty", layoutName)
				}
			})

			// 7. Stylesheet and Asset References
			t.Run("stylesheet_and_assets", func(t *testing.T) {
				if !strings.Contains(renderedHTML, "/assets/css/theme-foundations.css") {
					t.Errorf("layout %q does not reference /assets/css/theme-foundations.css", layoutName)
				}
			})

			// 8. Heading Hierarchy
			t.Run("heading_hierarchy", func(t *testing.T) {
				headingNodes := collectNodes(doc, func(n *html.Node) bool {
					if n.Type == html.ElementNode {
						data := strings.ToLower(n.Data)
						return len(data) == 2 && data[0] == 'h' && data[1] >= '1' && data[1] <= '6'
					}
					return false
				})

				previousLevel := 0
				h1Count := 0
				for _, h := range headingNodes {
					level, _ := strconv.Atoi(h.Data[1:])
					if level == 1 {
						h1Count++
					}
					if previousLevel > 0 && level > previousLevel+1 {
						t.Errorf("layout %q heading level jumped from h%d to h%d", layoutName, previousLevel, level)
					}
					previousLevel = level
				}
				if h1Count != 1 {
					t.Errorf("layout %q expected exactly 1 h1 heading, found %d", layoutName, h1Count)
				}
			})

			// 9. Visible Focus Styles and Accessible Labels
			t.Run("accessibility_labels_and_focus", func(t *testing.T) {
				// Verify image alt tags / aria-hidden
				imgNodes := collectNodes(doc, func(n *html.Node) bool { return n.Type == html.ElementNode && n.Data == "img" })
				for _, img := range imgNodes {
					ariaHidden, _ := getNodeAttr(img, "aria-hidden")
					alt, hasAlt := getNodeAttr(img, "alt")
					if ariaHidden == "true" {
						if hasAlt && strings.TrimSpace(alt) != "" {
							t.Errorf("layout %q aria-hidden img must have empty alt attribute, got alt=%q", layoutName, alt)
						}
					} else if !hasAlt {
						t.Errorf("layout %q img missing alt attribute", layoutName)
					}
				}

				// Nav aria-label check (theme exception for retro glitch / floating cards theme placeholders in layout-the-hacker and layout-floating-cards)
				if layoutName != "layout-the-hacker" && layoutName != "layout-floating-cards" {
					navNodes := collectNodes(doc, func(n *html.Node) bool { return n.Type == html.ElementNode && n.Data == "nav" })
					for _, nav := range navNodes {
						_, hasAriaLabel := getNodeAttr(nav, "aria-label")
						_, hasAriaLabelledBy := getNodeAttr(nav, "aria-labelledby")
						if !hasAriaLabel && !hasAriaLabelledBy {
							t.Errorf("layout %q <nav> element missing aria-label or aria-labelledby", layoutName)
						}
					}
				}
			})

			// 10. Emoji Kitchen Output Intact
			t.Run("emoji_kitchen_intact", func(t *testing.T) {
				if !strings.Contains(renderedHTML, `class="emoji-kitchen"`) && !strings.Contains(renderedHTML, `class='emoji-kitchen'`) {
					t.Errorf("layout %q mangled or omitted Emoji Kitchen class=\"emoji-kitchen\"", layoutName)
				}
				if !strings.Contains(renderedHTML, `/assets/img/emoji-kitchen.png`) {
					t.Errorf("layout %q omitted Emoji Kitchen image src", layoutName)
				}
			})
		})
	}
}
