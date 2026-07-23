package main

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/generator"
)

func TestArtisanalCeramicsBoutiqueExampleSite(t *testing.T) {
	fixtureContentDir, err := filepath.Abs(filepath.Join("..", "..", "assets", "testdata", "sites", "artisanal-ceramics", "content"))
	if err != nil {
		t.Fatalf("failed to resolve fixture content path: %v", err)
	}

	assetDir, err := filepath.Abs(filepath.Join("..", "..", "assets", "testdata", "sites", "artisanal-ceramics", "assets"))
	if err != nil {
		t.Fatalf("failed to resolve asset path: %v", err)
	}

	templatePath, err := filepath.Abs(filepath.Join("..", "..", "templates", "layout.html"))
	if err != nil {
		t.Fatalf("failed to resolve template path: %v", err)
	}

	outDir := t.TempDir()

	cfg := config.DefaultConfig()
	cfg.ContentDir = fixtureContentDir
	cfg.AssetDir = assetDir
	cfg.OutputDir = outDir
	cfg.Template = templatePath
	cfg.SiteURL = "https://kintsugi.example.com"
	cfg.SiteName = "Kintsugi & Co. Studio"

	res, err := generator.Build(cfg)
	if err != nil {
		t.Fatalf("generator.Build failed for artisanal-ceramics example site: %v", err)
	}

	if res.PageCount < 4 {
		t.Errorf("expected page count >= 4, got %d", res.PageCount)
	}

	// 1. Inspect HTML Outputs & Metadata Tags
	expectedHTMLPages := []string{
		"index.html",
		filepath.Join("collection", "wheel-thrown-vessels", "index.html"),
		filepath.Join("care-guide", "index.html"),
		filepath.Join("journal", "2026-07-15-glazing-techniques", "index.html"),
	}

	for _, pagePath := range expectedHTMLPages {
		fullPath := filepath.Join(outDir, pagePath)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			t.Errorf("missing HTML page %s: %v", pagePath, err)
			continue
		}
		htmlStr := string(data)
		if !strings.Contains(htmlStr, "<!DOCTYPE html>") {
			t.Errorf("page %s does not contain valid HTML5 doctype", pagePath)
		}
		if !strings.Contains(htmlStr, "Kintsugi &amp; Co.") && !strings.Contains(htmlStr, "Kintsugi & Co.") {
			t.Errorf("page %s does not contain site name", pagePath)
		}
	}

	// Verify unrendered file (render: false) is NOT emitted as standalone HTML
	unrenderedHTML := filepath.Join(outDir, "notes", "unrendered-formulas", "index.html")
	if _, err := os.Stat(unrenderedHTML); !os.IsNotExist(err) {
		t.Errorf("unrendered file with render:false was incorrectly output to %s", unrenderedHTML)
	}

	// 2. Inspect Search Index (search.json)
	searchPath := filepath.Join(outDir, "search.json")
	searchBytes, err := os.ReadFile(searchPath)
	if err != nil {
		t.Fatalf("missing search.json: %v", err)
	}
	var searchEntries []struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(searchBytes, &searchEntries); err != nil {
		t.Fatalf("failed to parse search.json: %v", err)
	}
	if len(searchEntries) < 5 {
		t.Errorf("expected at least 5 search entries (including unrendered notes), got %d", len(searchEntries))
	}

	// 3. Inspect Taxonomy (tags/ & categories/)
	tagsDir := filepath.Join(outDir, "tags")
	if _, err := os.Stat(filepath.Join(tagsDir, "index.html")); os.IsNotExist(err) {
		t.Errorf("missing tags taxonomy index at %s", filepath.Join(tagsDir, "index.html"))
	}
	categoriesDir := filepath.Join(outDir, "categories")
	if _, err := os.Stat(filepath.Join(categoriesDir, "index.html")); os.IsNotExist(err) {
		t.Errorf("missing categories taxonomy index at %s", filepath.Join(categoriesDir, "index.html"))
	}

	// 4. Inspect RSS Feed (feed.xml)
	feedPath := filepath.Join(outDir, "feed.xml")
	feedBytes, err := os.ReadFile(feedPath)
	if err != nil {
		t.Fatalf("missing feed.xml: %v", err)
	}
	var rssStruct struct {
		XMLName xml.Name `xml:"rss"`
		Channel struct {
			Title string `xml:"title"`
			Items []struct {
				Title string `xml:"title"`
				Link  string `xml:"link"`
			} `xml:"item"`
		} `xml:"channel"`
	}
	if err := xml.Unmarshal(feedBytes, &rssStruct); err != nil {
		t.Fatalf("failed to parse feed.xml: %v", err)
	}
	if len(rssStruct.Channel.Items) == 0 {
		t.Errorf("expected RSS feed items for dated posts, got 0")
	}

	// 5. Inspect Sitemap (sitemap.xml)
	sitemapPath := filepath.Join(outDir, "sitemap.xml")
	sitemapBytes, err := os.ReadFile(sitemapPath)
	if err != nil {
		t.Fatalf("missing sitemap.xml: %v", err)
	}
	if !strings.Contains(string(sitemapBytes), "<urlset") || !strings.Contains(string(sitemapBytes), "https://kintsugi.example.com") {
		t.Errorf("sitemap.xml missing urlset or site base URL")
	}

	// 6. Inspect Robots (robots.txt)
	robotsPath := filepath.Join(outDir, "robots.txt")
	robotsBytes, err := os.ReadFile(robotsPath)
	if err != nil {
		t.Fatalf("missing robots.txt: %v", err)
	}
	if !strings.Contains(string(robotsBytes), "User-agent:") || !strings.Contains(string(robotsBytes), "Sitemap:") {
		t.Errorf("robots.txt missing User-agent or Sitemap link")
	}

	// 7. Inspect Graph, Backlinks, Meta JSON files
	graphPath := filepath.Join(outDir, "graph.json")
	if _, err := os.Stat(graphPath); os.IsNotExist(err) {
		t.Errorf("missing graph.json at %s", graphPath)
	}
	backlinksPath := filepath.Join(outDir, "backlinks.json")
	if _, err := os.Stat(backlinksPath); os.IsNotExist(err) {
		t.Errorf("missing backlinks.json at %s", backlinksPath)
	}
	metaPath := filepath.Join(outDir, "meta.json")
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		t.Errorf("missing meta.json at %s", metaPath)
	}

	// 8. Inspect Copied Assets
	assetPath := filepath.Join(outDir, "assets", "ceramic-vase.png")
	assetData, err := os.ReadFile(assetPath)
	if err != nil {
		t.Errorf("missing copied asset %s: %v", assetPath, err)
	} else if !strings.Contains(string(assetData), "ceramic vase image data") {
		t.Errorf("copied asset content mismatch")
	}
}
