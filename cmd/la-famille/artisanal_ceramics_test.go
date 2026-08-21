package main

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/generator"
)

// TestArtisanalCeramicsBoutiqueExampleSite is the milestone publishing
// integration harness: one realistic fixture site exercised through
// generator.Build, asserting every publishing artifact end to end in both
// documented contract modes — with and without siteurl.
func TestArtisanalCeramicsBoutiqueExampleSite(t *testing.T) {
	modes := []struct {
		name    string
		siteURL string
	}{
		{name: "with siteurl", siteURL: "https://kintsugi.example.com"},
		{name: "without siteurl", siteURL: ""},
	}

	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			outDir := buildBoutiqueFixture(t, mode.siteURL)
			assertBoutiquePublishing(t, outDir, mode.siteURL)
		})
	}
}

func buildBoutiqueFixture(t *testing.T, siteURL string) string {
	t.Helper()

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
	cfg.SiteURL = siteURL
	cfg.SiteName = "Kintsugi & Co. Studio"

	res, err := generator.Build(cfg)
	if err != nil {
		t.Fatalf("generator.Build failed for artisanal-ceramics example site: %v", err)
	}

	if res.PageCount < 4 {
		t.Errorf("expected page count >= 4, got %d", res.PageCount)
	}
	return outDir
}

func assertBoutiquePublishing(t *testing.T, outDir, siteURL string) {
	t.Helper()

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

	// Canonical tags follow the siteurl contract: emitted with the absolute
	// public URL when siteurl is configured, omitted entirely when it is not.
	indexHTMLBytes, err := os.ReadFile(filepath.Join(outDir, "index.html"))
	if err != nil {
		t.Fatalf("missing index.html: %v", err)
	}
	indexHasCanonical := strings.Contains(string(indexHTMLBytes), `rel="canonical"`)
	if siteURL != "" && !indexHasCanonical {
		t.Errorf("index.html should contain rel=canonical when siteurl is set")
	}
	if siteURL == "" && indexHasCanonical {
		t.Errorf("index.html should omit rel=canonical when siteurl is empty")
	}

	// robots.txt always allows crawlers; the Sitemap directive appears only
	// when siteurl gives the sitemap an absolute location.
	robotsBytes, err := os.ReadFile(filepath.Join(outDir, "robots.txt"))
	if err != nil {
		t.Fatalf("missing robots.txt: %v", err)
	}
	robots := string(robotsBytes)
	if !strings.Contains(robots, "User-agent: *") || !strings.Contains(robots, "Allow: /") {
		t.Errorf("robots.txt missing User-agent/Allow rules, got:\n%s", robots)
	}
	hasSitemapDirective := strings.Contains(robots, "Sitemap:")
	if siteURL != "" && !hasSitemapDirective {
		t.Errorf("robots.txt should include Sitemap directive when siteurl is set, got:\n%s", robots)
	}
	if siteURL == "" && hasSitemapDirective {
		t.Errorf("robots.txt should omit Sitemap directive when siteurl is empty, got:\n%s", robots)
	}

	// 2. Inspect Search Index (search.json) using the real wire schema
	// t/u/g/s/h from internal/search. render:false pages are excluded from the
	// index; rendered pages and generated taxonomy pages are included.
	searchPath := filepath.Join(outDir, "search.json")
	searchBytes, err := os.ReadFile(searchPath)
	if err != nil {
		t.Fatalf("missing search.json: %v", err)
	}
	var searchEntries []struct {
		Title    string   `json:"t"`
		URL      string   `json:"u"`
		Tags     []string `json:"g"`
		Snippet  string   `json:"s"`
		Headings []string `json:"h"`
	}
	if err := json.Unmarshal(searchBytes, &searchEntries); err != nil {
		t.Fatalf("failed to parse search.json: %v", err)
	}

	urlsByID := make(map[string]string, len(searchEntries))
	for _, entry := range searchEntries {
		if entry.Title == "" {
			t.Errorf("search.json entry with empty title: %s", entry.URL)
		}
		if entry.URL == "" {
			t.Errorf("search.json entry with empty URL: %s", entry.Title)
		}
		urlsByID[entry.URL] = entry.Title
	}

	for _, wantURLSuffix := range []string{
		"/collection/wheel-thrown-vessels/",
		"/care-guide/",
		"/journal/2026-07-15-glazing-techniques/",
	} {
		found := false
		for url := range urlsByID {
			if strings.HasSuffix(url, wantURLSuffix) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("search.json missing rendered page entry %q", wantURLSuffix)
		}
	}
	hasHomepageEntry := false
	for url := range urlsByID {
		if url == "/" || url == "/index.html" {
			hasHomepageEntry = true
		}
		if strings.Contains(url, "unrendered-formulas") {
			t.Errorf("search.json unexpectedly contains render:false page: %s", url)
		}
	}
	if !hasHomepageEntry {
		t.Errorf("search.json missing homepage entry")
	}

	// 3. Inspect Taxonomy (tags/ & categories/): index pages and term pages
	tagsDir := filepath.Join(outDir, "tags")
	if _, err := os.Stat(filepath.Join(tagsDir, "index.html")); os.IsNotExist(err) {
		t.Errorf("missing tags taxonomy index at %s", filepath.Join(tagsDir, "index.html"))
	}
	categoriesDir := filepath.Join(outDir, "categories")
	if _, err := os.Stat(filepath.Join(categoriesDir, "index.html")); os.IsNotExist(err) {
		t.Errorf("missing categories taxonomy index at %s", filepath.Join(categoriesDir, "index.html"))
	}
	expectedTermPages := []string{
		filepath.Join(tagsDir, "ceramics", "index.html"),
		filepath.Join(tagsDir, "glazing", "index.html"),
		filepath.Join(categoriesDir, "crafts", "index.html"),
		filepath.Join(categoriesDir, "journal", "index.html"),
	}
	for _, termPage := range expectedTermPages {
		if _, err := os.Stat(termPage); os.IsNotExist(err) {
			t.Errorf("missing taxonomy term page %s", termPage)
		}
	}

	// 4. Inspect RSS Feed (feed.xml): all four rendered pages are dated, so
	// the feed must contain exactly those items, newest first, RFC1123Z dates.
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
				Title   string `xml:"title"`
				Link    string `xml:"link"`
				PubDate string `xml:"pubDate"`
			} `xml:"item"`
		} `xml:"channel"`
	}
	if err := xml.Unmarshal(feedBytes, &rssStruct); err != nil {
		t.Fatalf("failed to parse feed.xml: %v", err)
	}
	if len(rssStruct.Channel.Items) != 4 {
		t.Errorf("expected RSS feed items for the 4 dated rendered pages, got %d", len(rssStruct.Channel.Items))
	}
	if len(rssStruct.Channel.Items) > 0 {
		newest := rssStruct.Channel.Items[0]
		if !strings.Contains(newest.Link, "2026-07-15-glazing-techniques") {
			t.Errorf("feed.xml newest item should be the 2026-07-15 journal entry, got link %q", newest.Link)
		}
		if siteURL != "" && !strings.HasPrefix(newest.Link, siteURL+"/") {
			t.Errorf("feed.xml item links should be absolute with siteurl, got %q", newest.Link)
		}
		if siteURL == "" && !strings.HasPrefix(newest.Link, "/") {
			t.Errorf("feed.xml item links should be root-relative without siteurl, got %q", newest.Link)
		}
	}
	pubDateRE := regexp.MustCompile(`^[A-Z][a-z]{2}, \d{2} [A-Z][a-z]{2} \d{4} \d{2}:\d{2}:\d{2} [+-]\d{4}$`)
	itemDates := make([]time.Time, 0, len(rssStruct.Channel.Items))
	for i, item := range rssStruct.Channel.Items {
		if !pubDateRE.MatchString(item.PubDate) {
			t.Errorf("feed.xml item %d pubDate %q is not RFC1123Z", i, item.PubDate)
			continue
		}
		parsed, err := time.Parse(time.RFC1123Z, item.PubDate)
		if err != nil {
			t.Errorf("feed.xml item %d pubDate %q failed to parse: %v", i, item.PubDate, err)
			continue
		}
		itemDates = append(itemDates, parsed)
	}
	for i := 1; i < len(itemDates); i++ {
		if itemDates[i-1].Before(itemDates[i]) {
			t.Errorf("feed.xml items not sorted newest-first: %q before %q", rssStruct.Channel.Items[i-1].PubDate, rssStruct.Channel.Items[i].PubDate)
		}
	}

	// 5. Inspect Sitemap (sitemap.xml): unique absolute URLs covering rendered
	// pages and taxonomy term pages, excluding render:false pages.
	sitemapPath := filepath.Join(outDir, "sitemap.xml")
	sitemapBytes, err := os.ReadFile(sitemapPath)
	if err != nil {
		t.Fatalf("missing sitemap.xml: %v", err)
	}
	var sitemapStruct struct {
		URLs []struct {
			Loc string `xml:"loc"`
		} `xml:"url"`
	}
	if err := xml.Unmarshal(sitemapBytes, &sitemapStruct); err != nil {
		t.Fatalf("failed to parse sitemap.xml: %v", err)
	}
	seenLocs := make(map[string]bool, len(sitemapStruct.URLs))
	sitemapHas := func(suffix string) bool {
		for loc := range seenLocs {
			if strings.HasSuffix(loc, suffix) {
				return true
			}
		}
		return false
	}
	for _, u := range sitemapStruct.URLs {
		loc := strings.TrimSpace(u.Loc)
		if loc == "" {
			t.Errorf("sitemap.xml contains an empty <loc>")
			continue
		}
		if siteURL != "" && !strings.HasPrefix(loc, siteURL+"/") {
			t.Errorf("sitemap.xml <loc> %q should be absolute with siteurl", loc)
		}
		if siteURL == "" && !strings.HasPrefix(loc, "/") {
			t.Errorf("sitemap.xml <loc> %q should be root-relative without siteurl", loc)
		}
		if seenLocs[loc] {
			t.Errorf("sitemap.xml contains duplicate <loc> %q", loc)
		}
		seenLocs[loc] = true
	}
	for _, want := range []string{
		"collection/wheel-thrown-vessels/",
		"tags/ceramics/",
		"categories/journal/",
	} {
		if !sitemapHas(want) {
			t.Errorf("sitemap.xml missing expected URL %q", want)
		}
	}
	if sitemapHas("unrendered-formulas") {
		t.Errorf("sitemap.xml unexpectedly contains render:false page notes/unrendered-formulas")
	}

	// 6. Inspect Graph, Backlinks, Meta JSON files: parse and assert content,
	// not just existence.
	graphBytes, err := os.ReadFile(filepath.Join(outDir, "graph.json"))
	if err != nil {
		t.Fatalf("missing graph.json: %v", err)
	}
	var graphStruct struct {
		Nodes map[string]struct {
			Type   string `json:"type"`
			Render bool   `json:"render"`
		} `json:"nodes"`
		Edges [][2]string `json:"edges"`
	}
	if err := json.Unmarshal(graphBytes, &graphStruct); err != nil {
		t.Fatalf("failed to parse graph.json: %v", err)
	}
	if len(graphStruct.Nodes) < 5 {
		t.Errorf("graph.json expected >= 5 nodes (4 rendered + 1 unrendered), got %d", len(graphStruct.Nodes))
	}
	unrenderedNode, ok := graphStruct.Nodes["notes/unrendered-formulas.md"]
	if !ok {
		t.Errorf("graph.json missing node for unrendered note notes/unrendered-formulas.md")
	} else if unrenderedNode.Render {
		t.Errorf("graph.json node notes/unrendered-formulas.md should have render:false")
	}
	if len(graphStruct.Edges) == 0 {
		t.Errorf("graph.json has no edges although index.md links other fixture pages")
	}

	backlinksBytes, err := os.ReadFile(filepath.Join(outDir, "backlinks.json"))
	if err != nil {
		t.Fatalf("missing backlinks.json: %v", err)
	}
	var backlinks map[string][]string
	if err := json.Unmarshal(backlinksBytes, &backlinks); err != nil {
		t.Fatalf("failed to parse backlinks.json: %v", err)
	}
	vesselBacklinks := backlinks["collection/wheel-thrown-vessels"]
	hasIndexBacklink := false
	for _, parent := range vesselBacklinks {
		if parent == "index" {
			hasIndexBacklink = true
		}
	}
	if !hasIndexBacklink {
		t.Errorf("backlinks.json missing index -> collection/wheel-thrown-vessels backlink, got %v", vesselBacklinks)
	}

	metaBytes, err := os.ReadFile(filepath.Join(outDir, "meta.json"))
	if err != nil {
		t.Fatalf("missing meta.json: %v", err)
	}
	var metaData map[string]struct {
		Title  string `json:"title"`
		Render bool   `json:"render"`
		URL    string `json:"url"`
	}
	if err := json.Unmarshal(metaBytes, &metaData); err != nil {
		t.Fatalf("failed to parse meta.json: %v", err)
	}
	indexMeta, ok := metaData["index"]
	if !ok {
		t.Errorf("meta.json missing entry for index")
	} else if !indexMeta.Render || indexMeta.Title == "" {
		t.Errorf("meta.json index entry unexpected: %+v", indexMeta)
	}
	noteMeta, ok := metaData["notes/unrendered-formulas.md"]
	if !ok {
		t.Errorf("meta.json missing entry for unrendered note")
	} else if noteMeta.Render {
		t.Errorf("meta.json entry for unrendered note should have render:false")
	}

	// 7. Inspect Copied Assets
	assetPath := filepath.Join(outDir, "assets", "ceramic-vase.png")
	assetData, err := os.ReadFile(assetPath)
	if err != nil {
		t.Errorf("missing copied asset %s: %v", assetPath, err)
	} else if !strings.Contains(string(assetData), "ceramic vase image data") {
		t.Errorf("copied asset content mismatch")
	}
}
