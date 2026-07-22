package render

import (
	"os"
	"strings"
	"testing"
)

func TestDefaultTemplateContract(t *testing.T) {
	sourceBytes, err := os.ReadFile("../../templates/layout.html")
	if err != nil {
		t.Fatalf("read default template: %v", err)
	}
	source := string(sourceBytes)

	checks := map[string]string{
		"semantic header":       `<header class="navbar`,
		"navigation landmark":   `<nav id="site-navigation" aria-label="Main navigation"`,
		"skip target":           `href="#main-content"`,
		"conditional canonical": `{{if .CanonicalURL}}<link rel="canonical" href="{{.CanonicalURL}}">`,
		"conditional og url":    `<meta property="og:url" content="{{.CanonicalURL}}">{{end}}`,
		"empty site title":      `{{if .Site.SiteName}}{{.Title}} - {{.Site.SiteName}}{{else}}{{.Title}}{{end}}`,
	}
	for name, want := range checks {
		if !strings.Contains(source, want) {
			t.Errorf("default template missing %s: %q", name, want)
		}
	}
	if count := strings.Count(source, `<script src="/assets/js/search.js"></script>`); count != 1 {
		t.Errorf("expected one search script tag, found %d", count)
	}
}
