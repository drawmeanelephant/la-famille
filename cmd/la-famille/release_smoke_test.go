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

func TestReleaseSmoke(t *testing.T) {
	fixtureContentDir, err := filepath.Abs(filepath.Join("..", "..", "assets", "testdata", "sites", "release-smoke", "content"))
	if err != nil {
		t.Fatalf("failed to resolve fixture content path: %v", err)
	}

	templatePath, err := filepath.Abs(filepath.Join("..", "..", "templates", "layout.html"))
	if err != nil {
		t.Fatalf("failed to resolve template path: %v", err)
	}

	outDir1 := t.TempDir()

	cfg := config.DefaultConfig()
	cfg.ContentDir = fixtureContentDir
	cfg.AssetDir = filepath.Join(fixtureContentDir, "..", "assets")
	cfg.OutputDir = outDir1
	cfg.Template = templatePath
	cfg.SiteURL = "https://example.com"
	cfg.SiteName = "Release Smoke Test Site"

	res, err := generator.Build(cfg)
	if err != nil {
		t.Fatalf("generator.Build failed: %v", err)
	}

	if res.PageCount == 0 {
		t.Errorf("expected page count > 0, got %d", res.PageCount)
	}

	// 1. Verify HTML pages exist and contain canonical/OG metadata
	verifyHTMLPage(t, outDir1, "index.html", []string{
		`<title>Home Page - Release Smoke Test Site</title>`,
		`<meta name="description" content="Welcome to the release smoke test site.">`,
		`<meta property="og:title" content="Home Page">`,
		`<meta property="og:description" content="Welcome to the release smoke test site.">`,
		`<link rel="canonical" href="https://example.com/">`,
		`<meta property="og:url" content="https://example.com/">`,
	})

	verifyHTMLPage(t, outDir1, filepath.Join("about", "index.html"), []string{
		`<link rel="canonical" href="https://example.com/about/">`,
		`<meta property="og:url" content="https://example.com/about/">`,
		`About Us`,
	})

	verifyHTMLPage(t, outDir1, filepath.Join("posts", "first-post", "index.html"), []string{
		`<link rel="canonical" href="https://example.com/posts/first-post/">`,
		`First Release Post`,
	})

	verifyHTMLPage(t, outDir1, filepath.Join("posts", "second-post", "index.html"), []string{
		`<link rel="canonical" href="https://example.com/posts/second-post/">`,
		`Second Release Post`,
	})

	// 2. Verify graph.json
	graphPath := filepath.Join(outDir1, "graph.json")
	graphData, err := os.ReadFile(graphPath)
	if err != nil {
		t.Fatalf("missing graph.json: %v", err)
	}
	var gStruct struct {
		Nodes map[string]interface{} `json:"nodes"`
		Edges [][2]string            `json:"edges"`
	}
	if err := json.Unmarshal(graphData, &gStruct); err != nil {
		t.Fatalf("invalid graph.json format: %v", err)
	}
	if len(gStruct.Nodes) == 0 {
		t.Errorf("graph.json has empty nodes")
	}
	if len(gStruct.Edges) == 0 {
		t.Errorf("graph.json has empty edges")
	}

	// 3. Verify backlinks.json
	backlinksPath := filepath.Join(outDir1, "backlinks.json")
	backlinksData, err := os.ReadFile(backlinksPath)
	if err != nil {
		t.Fatalf("missing backlinks.json: %v", err)
	}
	var backlinksMap map[string][]string
	if err := json.Unmarshal(backlinksData, &backlinksMap); err != nil {
		t.Fatalf("invalid backlinks.json format: %v", err)
	}
	if len(backlinksMap) == 0 {
		t.Errorf("backlinks.json is empty")
	}

	// 4. Verify meta.json
	metaPath := filepath.Join(outDir1, "meta.json")
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("missing meta.json: %v", err)
	}
	var metaMap map[string]map[string]interface{}
	if err := json.Unmarshal(metaData, &metaMap); err != nil {
		t.Fatalf("invalid meta.json format: %v", err)
	}
	if _, ok := metaMap["index"]; !ok {
		t.Errorf("meta.json missing 'index' entry")
	}

	// 5. Verify search.json
	searchPath := filepath.Join(outDir1, "search.json")
	searchData, err := os.ReadFile(searchPath)
	if err != nil {
		t.Fatalf("missing search.json: %v", err)
	}
	var searchItems []struct {
		Title string   `json:"title"`
		URL   string   `json:"url"`
		Tags  []string `json:"tags"`
	}
	if err := json.Unmarshal(searchData, &searchItems); err != nil {
		t.Fatalf("invalid search.json format: %v", err)
	}
	if len(searchItems) == 0 {
		t.Errorf("search.json is empty")
	}

	// 6. Verify taxonomy pages
	verifyHTMLPage(t, outDir1, filepath.Join("tags", "index.html"), []string{"Tags"})
	verifyHTMLPage(t, outDir1, filepath.Join("tags", "release", "index.html"), []string{"release"})
	verifyHTMLPage(t, outDir1, filepath.Join("categories", "index.html"), []string{"Categories"})
	verifyHTMLPage(t, outDir1, filepath.Join("categories", "general", "index.html"), []string{"general"})

	// 7. Verify RSS feed (feed.xml)
	feedPath := filepath.Join(outDir1, "feed.xml")
	feedData, err := os.ReadFile(feedPath)
	if err != nil {
		t.Fatalf("missing feed.xml: %v", err)
	}
	var rssStruct struct {
		XMLName xml.Name `xml:"rss"`
		Channel struct {
			Title string `xml:"title"`
			Link  string `xml:"link"`
			Items []struct {
				Title   string `xml:"title"`
				Link    string `xml:"link"`
				PubDate string `xml:"pubDate"`
			} `xml:"item"`
		} `xml:"channel"`
	}
	if err := xml.Unmarshal(feedData, &rssStruct); err != nil {
		t.Fatalf("invalid feed.xml XML format: %v", err)
	}
	if rssStruct.Channel.Title != "Release Smoke Test Site" {
		t.Errorf("expected RSS channel title %q, got %q", "Release Smoke Test Site", rssStruct.Channel.Title)
	}
	if len(rssStruct.Channel.Items) == 0 {
		t.Errorf("RSS feed has no items")
	}

	// 8. Verify sitemap.xml
	sitemapPath := filepath.Join(outDir1, "sitemap.xml")
	sitemapData, err := os.ReadFile(sitemapPath)
	if err != nil {
		t.Fatalf("missing sitemap.xml: %v", err)
	}
	var sitemapStruct struct {
		XMLName xml.Name `xml:"urlset"`
		URLs    []struct {
			Loc string `xml:"loc"`
		} `xml:"url"`
	}
	if err := xml.Unmarshal(sitemapData, &sitemapStruct); err != nil {
		t.Fatalf("invalid sitemap.xml XML format: %v", err)
	}
	if len(sitemapStruct.URLs) == 0 {
		t.Errorf("sitemap.xml has no URLs")
	}
	hasHomepageLoc := false
	for _, u := range sitemapStruct.URLs {
		if u.Loc == "https://example.com/" {
			hasHomepageLoc = true
			break
		}
	}
	if !hasHomepageLoc {
		t.Errorf("sitemap.xml missing https://example.com/ entry")
	}

	// 9. Verify robots.txt
	robotsPath := filepath.Join(outDir1, "robots.txt")
	robotsData, err := os.ReadFile(robotsPath)
	if err != nil {
		t.Fatalf("missing robots.txt: %v", err)
	}
	robotsStr := string(robotsData)
	if !strings.Contains(robotsStr, "User-agent: *") || !strings.Contains(robotsStr, "Sitemap: https://example.com/sitemap.xml") {
		t.Errorf("robots.txt missing expected content, got:\n%s", robotsStr)
	}

	// 10. Verify static asset copying
	assetPath := filepath.Join(outDir1, "assets", "sample-image.png")
	assetData, err := os.ReadFile(assetPath)
	if err != nil {
		t.Fatalf("missing copied asset file %s: %v", assetPath, err)
	}
	if strings.TrimSpace(string(assetData)) != "fake image data for testing asset copying" {
		t.Errorf("copied asset content mismatch, got %q", string(assetData))
	}

	// 11. Determinism Check across repeated runs
	outDir2 := t.TempDir()
	cfg2 := cfg
	cfg2.OutputDir = outDir2

	if _, err := generator.Build(cfg2); err != nil {
		t.Fatalf("second generator.Build failed: %v", err)
	}

	filesToCompare := []string{
		"graph.json",
		"backlinks.json",
		"meta.json",
		"search.json",
		"feed.xml",
		"sitemap.xml",
		"robots.txt",
		"index.html",
	}

	for _, rel := range filesToCompare {
		data1, err1 := os.ReadFile(filepath.Join(outDir1, rel))
		data2, err2 := os.ReadFile(filepath.Join(outDir2, rel))
		if err1 != nil || err2 != nil {
			t.Fatalf("error reading file %s for comparison: run1 err=%v, run2 err=%v", rel, err1, err2)
		}
		if string(data1) != string(data2) {
			t.Errorf("non-deterministic build output for %s", rel)
		}
	}
}

func verifyHTMLPage(t *testing.T, baseDir, relPath string, expectedSubstrings []string) {
	t.Helper()
	p := filepath.Join(baseDir, relPath)
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("missing rendered HTML page %s: %v", relPath, err)
	}
	content := string(data)
	if len(content) == 0 {
		t.Errorf("rendered HTML page %s is empty", relPath)
	}
	for _, sub := range expectedSubstrings {
		if !strings.Contains(content, sub) {
			t.Errorf("HTML page %s missing expected substring %q", relPath, sub)
		}
	}
}
