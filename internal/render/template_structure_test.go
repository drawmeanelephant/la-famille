package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAlternateLayoutsShareAccessibleDocumentStructure keeps themes visually
// distinct while ensuring every built-in alternate layout remains usable with
// keyboard and assistive technology.
func TestAlternateLayoutsShareAccessibleDocumentStructure(t *testing.T) {
	templates, err := filepath.Glob(filepath.Join("..", "..", "templates", "*.html"))
	if err != nil {
		t.Fatalf("glob templates: %v", err)
	}
	for _, path := range templates {
		if filepath.Base(path) == "layout.html" {
			continue
		}
		t.Run(filepath.Base(path), func(t *testing.T) {
			contents, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read template: %v", err)
			}
			html := string(contents)
			checks := map[string]string{
				"doctype":          "<!DOCTYPE html>",
				"language":         `lang=`,
				"viewport":         `viewport`,
				"title":            "<title>",
				"main landmark":    `<main id="main-content"`,
				"skip link":        `href="#main-content"`,
				"canonical opt-in": "{{if .CanonicalURL}}",
			}
			for name, marker := range checks {
				if !strings.Contains(html, marker) {
					t.Errorf("missing %s marker %q", name, marker)
				}
			}
			if count := strings.Count(html, "<h1"); count != 1 {
				t.Errorf("expected exactly one page h1, found %d", count)
			}
		})
	}
}
