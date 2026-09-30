package generator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildOnlyPublishesUsedBundledThemeAssets(t *testing.T) {
	cfg, _ := setupTestSite(t)
	cfg.SiteURL = "https://example.com/blog"

	// The site has its own default layout, but one page opts into a theme
	// stylesheet through a per-page layout.
	// Metadata and Markdown may refer to bundled images independently of
	// the chosen layout.
	templateDir := filepath.Dir(cfg.Template)
	if err := os.WriteFile(cfg.Template,
		[]byte(`<!DOCTYPE html><html><head><link href="/assets/css/site.css"><meta property="og:image" content="{{.Image}}"></head><body>{{.Content}}</body></html>`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "editorial.html"),
		[]byte(`<!DOCTYPE html><html><head><link href="/assets/css/layout-editorial.css"></head><body>{{.Content}}</body></html>`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.ContentDir, "page1.md"),
		[]byte("---\nlayout: editorial\n---\n![Raoul](/assets/img/mascot-default.jpeg)"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg.DefaultOGImage = "/assets/img/u1f419_u1f354.png"

	if _, err := Build(cfg); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"css/layout-editorial.css", "img/mascot-default.jpeg", "img/u1f419_u1f354.png",
	} {
		if _, err := os.Stat(filepath.Join(cfg.OutputDir, "assets", filepath.FromSlash(name))); err != nil {
			t.Errorf("referenced bundled asset %s missing: %v", name, err)
		}
	}
	for _, name := range []string{
		"css/theme.css", "css/layout-midnight.css", "css/theme-foundations.css", "img/jules-logo.png",
	} {
		if _, err := os.Stat(filepath.Join(cfg.OutputDir, "assets", filepath.FromSlash(name))); !os.IsNotExist(err) {
			t.Errorf("unreferenced bundled asset %s was published: %v", name, err)
		}
	}

	// A rebuild must drop assets no longer used, not keep the old output.
	if err := os.WriteFile(filepath.Join(cfg.ContentDir, "page1.md"), []byte("# No bundled assets"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg.DefaultOGImage = ""
	if _, err := Build(cfg); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"css/layout-editorial.css", "img/mascot-default.jpeg", "img/u1f419_u1f354.png",
	} {
		if _, err := os.Stat(filepath.Join(cfg.OutputDir, "assets", filepath.FromSlash(name))); !os.IsNotExist(err) {
			t.Errorf("stale bundled asset %s survived rebuild: %v", name, err)
		}
	}
}
