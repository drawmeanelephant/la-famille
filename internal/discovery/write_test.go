package discovery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
)

func TestWrite(t *testing.T) {
	outputDir := t.TempDir()
	cfg := config.Config{OutputDir: outputDir, SiteURL: "https://example.com/docs"}

	if err := Write(cfg, []string{"guide/index.html", "", "index.html", "guide/index.html"}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	sitemap, err := os.ReadFile(filepath.Join(outputDir, "sitemap.xml"))
	if err != nil {
		t.Fatal(err)
	}
	wantSitemap := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://example.com/docs/</loc>
  </url>
  <url>
    <loc>https://example.com/docs/guide/</loc>
  </url>
</urlset>
`
	if string(sitemap) != wantSitemap {
		t.Fatalf("sitemap.xml =\n%s\nwant:\n%s", sitemap, wantSitemap)
	}

	robots, err := os.ReadFile(filepath.Join(outputDir, "robots.txt"))
	if err != nil {
		t.Fatal(err)
	}
	wantRobots := "User-agent: *\nAllow: /\n\nSitemap: https://example.com/docs/sitemap.xml\n"
	if string(robots) != wantRobots {
		t.Fatalf("robots.txt = %q, want %q", robots, wantRobots)
	}
}

func TestWriteWithoutSiteURL(t *testing.T) {
	outputDir := t.TempDir()
	if err := Write(config.Config{OutputDir: outputDir}, []string{"index.html"}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	sitemap, err := os.ReadFile(filepath.Join(outputDir, "sitemap.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(sitemap), "<loc>") {
		t.Fatalf("sitemap without site URL must not contain a location: %s", sitemap)
	}

	robots, err := os.ReadFile(filepath.Join(outputDir, "robots.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(robots), "Sitemap:") {
		t.Fatalf("robots without site URL must not contain a sitemap directive: %s", robots)
	}
}
