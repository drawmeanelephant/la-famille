package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/publisher"
)

func TestRepositoryFlagshipSitePublishingContract(t *testing.T) {
	repo := filepath.Join("..", "..")
	cfg, err := config.Load(filepath.Join(repo, "website.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Template != "templates/layout-site.html" || cfg.SiteURL != "https://la-famille.filed.fyi" {
		t.Fatalf("unexpected flagship configuration: template=%q siteurl=%q", cfg.Template, cfg.SiteURL)
	}
	root := t.TempDir()
	for _, dir := range []string{"content", "templates", "assets"} {
		if err := os.CopyFS(filepath.Join(root, dir), os.DirFS(filepath.Join(repo, dir))); err != nil {
			t.Fatal(err)
		}
	}
	cfg.ProjectRoot = root
	cfg.ContentDir = filepath.Join(root, "content")
	cfg.Template = filepath.Join(root, cfg.Template)
	cfg.AssetDir = filepath.Join(root, "assets")
	cfg.OutputDir = filepath.Join(root, "public")
	if _, err := Build(cfg); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"index.html", "docs/index.html", "docs/setup/index.html",
		"showcase/index.html", "showcase/escapement/index.html",
		"showcase/zai/index.html", "showcase/templates/index.html",
		"assets/css/site.css", "assets/js/search.js", "graph/index.html",
	} {
		if _, err := os.Stat(filepath.Join(cfg.OutputDir, name)); err != nil {
			t.Errorf("missing publish artifact %s: %v", name, err)
		}
	}
	home := readOutput(t, cfg, "index.html")
	for _, want := range []string{
		"Small files.", "Big possibilities.", "escapement.filed.fyi",
		"z.filed.fyi", `href="https://la-famille.filed.fyi/"`,
	} {
		if !strings.Contains(home, want) {
			t.Errorf("homepage missing %q", want)
		}
	}
	if _, err := publisher.Check(cfg.OutputDir, cfg.BasePath()); err != nil {
		t.Fatalf("flagship artifact validation: %v", err)
	}
	sitemap := readOutput(t, cfg, "sitemap.xml")
	if !strings.Contains(sitemap, "https://la-famille.filed.fyi/") ||
		strings.Contains(sitemap, "la-famille-go.pages.dev") {
		t.Error("sitemap must use the canonical custom domain, not the Pages preview")
	}
	defaults, err := config.Load(filepath.Join(repo, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if defaults.SiteURL != "" || defaults.Template != config.DefaultLayoutPath {
		t.Error("flagship configuration must not change starter project defaults")
	}
}
