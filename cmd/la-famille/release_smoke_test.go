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

	// 2. Verify graph.json (includes rendered pages and render:false pages)
	graphPath := filepath.Join(outDir1, "graph.json")
	graphData, err := os.ReadFile(graphPath)
	if err != nil {
		t.Fatalf("missing graph.json: %v", err)
	}
	var gStruct struct {
		Nodes map[string]struct {
			Type   string `json:"type"`
			Render bool   `json:"render"`
		} `json:"nodes"`
		Edges [][2]string `json:"edges"`
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
	rawNode, exists := gStruct.Nodes["unrendered.md"]
	if !exists {
		t.Errorf("graph.json missing 'unrendered.md' node")
	} else if rawNode.Render {
		t.Errorf("expected 'unrendered.md' node to have render: false in graph.json")
	}

	// 3. Verify backlinks.json (includes references to/from unrendered pages)
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
	if refs, ok := backlinksMap["unrendered.md"]; !ok || len(refs) == 0 {
		t.Errorf("expected backlinks.json to record reference to unrendered page, got: %v", refs)
	}

	// 4. Verify meta.json (includes metadata for render:false pages)
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
	if unrenderedMeta, ok := metaMap["unrendered.md"]; !ok {
		t.Errorf("meta.json missing 'unrendered.md' entry")
	} else if renderVal, ok := unrenderedMeta["render"].(bool); !ok || renderVal {
		t.Errorf("meta.json 'unrendered.md' expected render: false, got: %v", unrenderedMeta["render"])
	}

	// 5. Verify search.json (excludes render:false pages)
	searchPath := filepath.Join(outDir1, "search.json")
	searchData, err := os.ReadFile(searchPath)
	if err != nil {
		t.Fatalf("missing search.json: %v", err)
	}
	var searchItems []struct {
		Title string   `json:"t"`
		URL   string   `json:"u"`
		Tags  []string `json:"g"`
	}
	if err := json.Unmarshal(searchData, &searchItems); err != nil {
		t.Fatalf("invalid search.json format: %v", err)
	}
	if len(searchItems) == 0 {
		t.Errorf("search.json is empty")
	}
	for _, item := range searchItems {
		if strings.Contains(item.URL, "unrendered") || item.Title == "Raw Data Notes" {
			t.Errorf("search.json unexpectedly contains render:false page: %+v", item)
		}
	}

	// 6. Verify taxonomy pages (excludes render:false pages)
	verifyHTMLPage(t, outDir1, filepath.Join("tags", "index.html"), []string{"Tags"})
	verifyHTMLPage(t, outDir1, filepath.Join("tags", "release", "index.html"), []string{"release"})
	verifyHTMLPage(t, outDir1, filepath.Join("categories", "index.html"), []string{"Categories"})
	verifyHTMLPage(t, outDir1, filepath.Join("categories", "general", "index.html"), []string{"general"})
	if _, err := os.Stat(filepath.Join(outDir1, "tags", "raw", "index.html")); !os.IsNotExist(err) {
		t.Errorf("render:false tag page 'tags/raw/index.html' should not be generated")
	}

	// 7. Verify RSS feed (feed.xml, excludes render:false pages)
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
	for _, item := range rssStruct.Channel.Items {
		if strings.Contains(item.Link, "unrendered") || item.Title == "Raw Data Notes" {
			t.Errorf("feed.xml unexpectedly contains render:false page: %+v", item)
		}
	}

	// 8. Verify sitemap.xml (excludes render:false pages)
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
		}
		if strings.Contains(u.Loc, "unrendered") {
			t.Errorf("sitemap.xml unexpectedly contains render:false location: %s", u.Loc)
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

	// 10. Verify unrendered file copied raw
	unrenderedRawPath := filepath.Join(outDir1, "unrendered.md")
	unrenderedData, err := os.ReadFile(unrenderedRawPath)
	if err != nil {
		t.Fatalf("missing unrendered raw output file %s: %v", unrenderedRawPath, err)
	}
	if !strings.Contains(string(unrenderedData), "render: false") || strings.Contains(string(unrenderedData), "<!DOCTYPE html>") {
		t.Errorf("unrendered file was not copied raw: %s", string(unrenderedData))
	}

	// 11. Verify static asset copying
	assetPath := filepath.Join(outDir1, "assets", "sample-image.png")
	assetData, err := os.ReadFile(assetPath)
	if err != nil {
		t.Fatalf("missing copied asset file %s: %v", assetPath, err)
	}
	if strings.TrimSpace(string(assetData)) != "fake image data for testing asset copying" {
		t.Errorf("copied asset content mismatch, got %q", string(assetData))
	}

	// 12. Contract verification when SiteURL is omitted (empty)
	outDirNoSiteURL := t.TempDir()
	cfgNoSiteURL := cfg
	cfgNoSiteURL.OutputDir = outDirNoSiteURL
	cfgNoSiteURL.SiteURL = ""

	if _, err := generator.Build(cfgNoSiteURL); err != nil {
		t.Fatalf("generator.Build with empty SiteURL failed: %v", err)
	}

	// 12a. robots.txt should omit Sitemap directive
	robotsNoURLData, err := os.ReadFile(filepath.Join(outDirNoSiteURL, "robots.txt"))
	if err != nil {
		t.Fatalf("missing robots.txt in no-SiteURL build: %v", err)
	}
	if strings.Contains(string(robotsNoURLData), "Sitemap:") {
		t.Errorf("robots.txt should omit Sitemap directive when SiteURL is empty, got:\n%s", string(robotsNoURLData))
	}

	// 12b. sitemap.xml locations should be root-relative
	sitemapNoURLData, err := os.ReadFile(filepath.Join(outDirNoSiteURL, "sitemap.xml"))
	if err != nil {
		t.Fatalf("missing sitemap.xml in no-SiteURL build: %v", err)
	}
	var sitemapNoURLStruct struct {
		XMLName xml.Name `xml:"urlset"`
		URLs    []struct {
			Loc string `xml:"loc"`
		} `xml:"url"`
	}
	if err := xml.Unmarshal(sitemapNoURLData, &sitemapNoURLStruct); err != nil {
		t.Fatalf("invalid sitemap.xml in no-SiteURL build: %v", err)
	}
	if len(sitemapNoURLStruct.URLs) == 0 {
		t.Errorf("sitemap.xml in no-SiteURL build is empty")
	} else if !strings.HasPrefix(sitemapNoURLStruct.URLs[0].Loc, "/") {
		t.Errorf("expected root-relative sitemap location, got: %q", sitemapNoURLStruct.URLs[0].Loc)
	}

	// 12c. HTML canonical URL tag omitted when SiteURL is empty
	indexHTMLNoSiteURL, err := os.ReadFile(filepath.Join(outDirNoSiteURL, "index.html"))
	if err != nil {
		t.Fatalf("missing index.html in no-SiteURL build: %v", err)
	}
	if strings.Contains(string(indexHTMLNoSiteURL), "rel=\"canonical\"") {
		t.Errorf("index.html should omit rel=\"canonical\" when SiteURL is empty")
	}

	// 13. Determinism Check across repeated runs
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
		"unrendered.md",
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
