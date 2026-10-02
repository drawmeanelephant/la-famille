package pack

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/generator"
	"github.com/tbuddy/la-famille/internal/ragexport"
)

func TestArtisanalCeramicsRoundTrip(t *testing.T) {
	fixture, err := filepath.Abs("../../assets/testdata/sites/artisanal-ceramics")
	if err != nil {
		t.Fatal(err)
	}
	template, err := filepath.Abs("../../templates/layout.html")
	if err != nil {
		t.Fatal(err)
	}
	temp := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.ProjectRoot = fixture
	cfg.ContentDir = filepath.Join(fixture, "content")
	cfg.AssetDir = filepath.Join(fixture, "assets")
	cfg.Template = template
	cfg.OutputDir = filepath.Join(temp, "public")
	cfg.RagDir = filepath.Join(temp, "rag")
	cfg.SiteName = "Kintsugi & Co. Studio"
	cfg.SiteURL = "https://kintsugi.example.com"
	if _, err := generator.Build(cfg); err != nil {
		t.Fatal(err)
	}
	if err := ragexport.RunExport(cfg); err != nil {
		t.Fatal(err)
	}
	options := BuildOptions{
		OutputDir: cfg.OutputDir, RagDir: cfg.RagDir,
		Site:       Site{Name: cfg.SiteName, URL: cfg.SiteURL},
		Provenance: Provenance{Generator: "la-famille", Version: "dev"},
	}
	first, second := filepath.Join(temp, "first.tar"), filepath.Join(temp, "second.tar")
	m, err := Build(options, first)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Build(options, second); err != nil {
		t.Fatal(err)
	}
	a, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("artisanal-ceramics packs are not byte-identical")
	}
	verified, err := VerifyFile(first)
	if err != nil {
		t.Fatal(err)
	}
	if verified.Site != options.Site || verified.ContentRoot != m.ContentRoot {
		t.Fatalf("round trip manifest = %+v", verified)
	}
	paths := make(map[string]bool)
	for _, member := range verified.Members {
		paths[member.Path] = true
	}
	for _, name := range []string{"rag-content.md", "site-manifest.json", "graph/data.json", "tags/index.html", "categories/index.html"} {
		if !paths[name] {
			t.Errorf("fixture pack missing %s", name)
		}
	}
	for name := range paths {
		if strings.Contains(name, "rag-system") || strings.Contains(name, "rag-config") || strings.HasPrefix(name, "content/") {
			t.Errorf("pack crossed content-only boundary: %s", name)
		}
	}
}
