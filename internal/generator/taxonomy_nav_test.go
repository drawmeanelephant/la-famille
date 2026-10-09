package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/search"
)

// Issue #529: an article page links each of its tags to that tag's archive,
// and the site nav gains Tags/Categories links whenever archive pages exist,
// so /tags/ is reachable from every page rather than only from the sitemap.
func TestBuild_TaxonomyArchivesReachable(t *testing.T) {
	template := `<html><body><nav>{{range .Site.SiteLinks}}<a href="{{.URL}}">{{.Label}}</a>{{end}}</nav><main>{{.Content}}</main></body></html>`

	setup := func(t *testing.T, files map[string]string) config.Config {
		t.Helper()
		tempDir := t.TempDir()
		contentDir := filepath.Join(tempDir, "content")
		templateDir := filepath.Join(tempDir, "templates")
		templatePath := filepath.Join(templateDir, "layout.html")
		if err := os.MkdirAll(templateDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(templatePath, []byte(template), 0600); err != nil {
			t.Fatal(err)
		}
		for name, body := range files {
			path := filepath.Join(contentDir, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(body), 0600); err != nil {
				t.Fatal(err)
			}
		}
		cfg := config.DefaultConfig()
		cfg.ContentDir = contentDir
		cfg.OutputDir = filepath.Join(tempDir, "public")
		cfg.Template = templatePath
		cfg.ProjectRoot = tempDir
		return cfg
	}

	t.Run("tagged site surfaces archives", func(t *testing.T) {
		cfg := setup(t, map[string]string{
			"index.md":      "---\ntitle: Home\ntags:\n  - welcome\ncategories:\n  - start\n---\nHOME_BODY\n",
			"blog/hello.md": "---\ntitle: Hello\ntags:\n  - meta\n---\nHELLO_BODY\n",
		})
		if _, err := Build(cfg); err != nil {
			t.Fatalf("Build: %v", err)
		}

		// The archives themselves are generated.
		for _, rel := range []string{
			"tags/index.html", "tags/welcome/index.html", "tags/meta/index.html",
			"categories/index.html", "categories/start/index.html",
		} {
			if _, err := os.Stat(filepath.Join(cfg.OutputDir, filepath.FromSlash(rel))); err != nil {
				t.Errorf("expected %s to exist: %v", rel, err)
			}
		}

		// The nav gains Tags and Categories links on the root page.
		home := readOutput(t, cfg, "index.html")
		for _, want := range []string{`href="/tags/"`, `>Tags</a>`, `href="/categories/"`, `>Categories</a>`} {
			if !strings.Contains(home, want) {
				t.Errorf("home nav missing %q in: %s", want, home)
			}
		}

		// The root article links its own tag with an archive-relative href.
		if !strings.Contains(home, `class="tag-link" href="tags/welcome/"`) {
			t.Errorf("home article missing tag link: %s", home)
		}

		// A nested article links its tag with parent-relative hrefs and still
		// carries the nav link.
		post := readOutput(t, cfg, "blog/hello/index.html")
		if !strings.Contains(post, `class="tag-link" href="../../tags/meta/"`) {
			t.Errorf("nested article missing tag link: %s", post)
		}
		if !strings.Contains(post, `href="/tags/"`) {
			t.Errorf("nested article nav missing Tags link: %s", post)
		}

		// search.json carries an archive URL for every taxonomy badge — tags
		// and categories alike — so the search modal can link each badge to a
		// real page instead of guessing /tags/ for everything.
		var searchIndex []search.Item
		searchBytes, err := os.ReadFile(filepath.Join(cfg.OutputDir, "search.json"))
		if err != nil {
			t.Fatalf("read search.json: %v", err)
		}
		if err := json.Unmarshal(searchBytes, &searchIndex); err != nil {
			t.Fatalf("parse search.json: %v", err)
		}
		byURL := make(map[string]search.Item, len(searchIndex))
		for _, item := range searchIndex {
			byURL[item.URL] = item
		}
		if homeItem := byURL["/"]; !slices.Equal(homeItem.Tags, []string{"welcome", "start"}) ||
			!slices.Equal(homeItem.TagURLs, []string{"/tags/welcome/", "/categories/start/"}) {
			t.Errorf("search entry for / has g/gu = %v/%v, want welcome/start and their archive URLs", homeItem.Tags, homeItem.TagURLs)
		}
		if postItem := byURL["/blog/hello/"]; !slices.Equal(postItem.Tags, []string{"meta"}) ||
			!slices.Equal(postItem.TagURLs, []string{"/tags/meta/"}) {
			t.Errorf("search entry for /blog/hello/ has g/gu = %v/%v, want meta and /tags/meta/", postItem.Tags, postItem.TagURLs)
		}
		// The tag archive page's own search entry links its badge to itself.
		if tagItem := byURL["/tags/meta/"]; !slices.Equal(tagItem.TagURLs, []string{"/tags/meta/"}) {
			t.Errorf("search entry for /tags/meta/ has gu = %v, want itself", tagItem.TagURLs)
		}
	})

	t.Run("untagged site gains nothing", func(t *testing.T) {
		cfg := setup(t, map[string]string{
			"index.md": "---\ntitle: Home\n---\nHOME_BODY\n",
		})
		if _, err := Build(cfg); err != nil {
			t.Fatalf("Build: %v", err)
		}

		home := readOutput(t, cfg, "index.html")
		for _, unwanted := range []string{"Tags", "Categories", "tag-link"} {
			if strings.Contains(home, unwanted) {
				t.Errorf("untagged home must not contain %q: %s", unwanted, home)
			}
		}
		if _, err := os.Stat(filepath.Join(cfg.OutputDir, "tags")); !os.IsNotExist(err) {
			t.Errorf("untagged site must not generate a tags/ directory")
		}
	})
}

// Issue #653: a term used as both tag and category must contribute both
// archive URLs to the page's search item. The shared dedupe set used to drop
// the category's URL, leaving /categories/go/ generated but unreferenced.
func TestBuild_SearchIndexLinksTagAndCategoryArchives(t *testing.T) {
	cfg := setupCollisionSite(t, map[string]string{
		"gopage.md": "---\ntitle: Go Page\ntags: [go]\ncategories: [go]\n---\nGO_BODY\n",
	})
	if _, err := Build(cfg); err != nil {
		t.Fatalf("Build: %v", err)
	}

	for _, rel := range []string{"tags/go/index.html", "categories/go/index.html"} {
		if _, err := os.Stat(filepath.Join(cfg.OutputDir, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("expected %s to exist: %v", rel, err)
		}
	}

	searchBytes, err := os.ReadFile(filepath.Join(cfg.OutputDir, "search.json"))
	if err != nil {
		t.Fatalf("read search.json: %v", err)
	}
	var items []search.Item
	if err := json.Unmarshal(searchBytes, &items); err != nil {
		t.Fatalf("parse search.json: %v", err)
	}
	for _, item := range items {
		if item.URL != "/gopage/" {
			continue
		}
		if !slices.Equal(item.Tags, []string{"go", "go"}) ||
			!slices.Equal(item.TagURLs, []string{"/tags/go/", "/categories/go/"}) {
			t.Fatalf("search entry for /gopage/ has g/gu = %v/%v, want both archive URLs", item.Tags, item.TagURLs)
		}
		return
	}
	t.Fatal("search.json has no item for /gopage/")
}

// Issue #649: NFC "café" and NFD "café" (e + combining acute) are one term.
// Kept byte-verbatim they produced two byte-distinct archive paths that
// resolve to a single file on normalization-insensitive filesystems — one
// archive on disk, two sitemap URLs, and one page's listing silently lost.
// Assertions inspect search.json and the archive contents, never the
// filesystem's normalization behavior, so they hold on every host.
func TestBuild_NormalizationEquivalentTermsShareArchive(t *testing.T) {
	cfg := setupCollisionSite(t, map[string]string{
		"a.md": "---\ntitle: Page A\ntags: [\"caf\u00e9\"]\n---\nA_BODY\n",
		"b.md": "---\ntitle: Page B\ntags: [\"cafe\u0301\"]\n---\nB_BODY\n",
	})
	if _, err := Build(cfg); err != nil {
		t.Fatalf("Build: %v", err)
	}

	// The merged archive lists both pages.
	archive := readOutput(t, cfg, "tags/caf\u00e9/index.html")
	for _, want := range []string{"Page A", "Page B"} {
		if !strings.Contains(archive, want) {
			t.Errorf("tags/caf\u00e9 archive missing %q: %s", want, archive)
		}
	}

	searchBytes, err := os.ReadFile(filepath.Join(cfg.OutputDir, "search.json"))
	if err != nil {
		t.Fatalf("read search.json: %v", err)
	}
	var items []search.Item
	if err := json.Unmarshal(searchBytes, &items); err != nil {
		t.Fatalf("parse search.json: %v", err)
	}
	// NFC café escapes as caf%C3%A9 and NFD as cafe%CC%81, so two distinct
	// /tags/caf URLs mean the term is still split.
	var archiveURLs []string
	seenPages := 0
	for _, item := range items {
		if strings.HasPrefix(item.URL, "/tags/caf") {
			archiveURLs = append(archiveURLs, item.URL)
		}
		if item.URL == "/a/" || item.URL == "/b/" {
			seenPages++
			if !slices.Equal(item.TagURLs, []string{"/tags/caf%C3%A9/"}) {
				t.Errorf("search entry %s has gu = %v, want the single NFC archive URL", item.URL, item.TagURLs)
			}
		}
	}
	if len(archiveURLs) != 1 || archiveURLs[0] != "/tags/caf%C3%A9/" {
		t.Errorf("tag archive URLs = %v, want exactly [/tags/caf%%C3%%A9/]", archiveURLs)
	}
	if seenPages != 2 {
		t.Errorf("search.json indexed %d of the two pages", seenPages)
	}

	sitemap := readOutput(t, cfg, "sitemap.xml")
	if n := strings.Count(sitemap, "/tags/caf"); n != 1 {
		t.Errorf("sitemap advertises %d caf\u00e9 archive URLs, want 1: %s", n, sitemap)
	}
}

func TestBuild_NativeLanguageTaxonomyArchives(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	if err := os.MkdirAll(filepath.Join(contentDir, "zh"), 0755); err != nil {
		t.Fatal(err)
	}
	doc := "---\ntitle: 中文\ndescription: Localized page\ntags: [起始]\ncategories: [说明]\n---\n正文\n"
	if err := os.WriteFile(filepath.Join(contentDir, "zh", "page.md"), []byte(doc), 0600); err != nil {
		t.Fatal(err)
	}
	templatePath := filepath.Join(dir, "layout.html")
	if err := os.WriteFile(templatePath, []byte(`<html><head><link rel="canonical" href="{{.CanonicalURL}}"></head><body>{{.Content}}</body></html>`), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.OutputDir = filepath.Join(dir, "public")
	cfg.Template = templatePath
	cfg.ProjectRoot = dir
	cfg.SiteURL = "https://example.com/site"
	result, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("native-language taxonomy terms produced warnings: %v", result.Warnings)
	}

	for _, tc := range []struct{ output, label, href string }{
		{"tags/起始/index.html", "起始", "https://example.com/site/tags/%E8%B5%B7%E5%A7%8B/"},
		{"categories/说明/index.html", "说明", "https://example.com/site/categories/%E8%AF%B4%E6%98%8E/"},
	} {
		archive := readOutput(t, cfg, tc.output)
		if !strings.Contains(archive, `href="`+tc.href+`"`) || !strings.Contains(archive, tc.label) {
			t.Errorf("%s missing canonical URL or label: %s", tc.output, archive)
		}
	}
	post := readOutput(t, cfg, "zh/page/index.html")
	if !strings.Contains(post, `href="../../tags/%E8%B5%B7%E5%A7%8B/"`) {
		t.Errorf("article missing encoded tag archive link: %s", post)
	}
	for _, tc := range []struct{ output, href string }{
		{"tags/index.html", `href="%E8%B5%B7%E5%A7%8B/"`},
		{"categories/index.html", `href="%E8%AF%B4%E6%98%8E/"`},
	} {
		index := readOutput(t, cfg, tc.output)
		if !strings.Contains(index, tc.href) {
			t.Errorf("%s missing encoded archive link %q: %s", tc.output, tc.href, index)
		}
	}

	searchBytes, err := os.ReadFile(filepath.Join(cfg.OutputDir, "search.json"))
	if err != nil {
		t.Fatal(err)
	}
	var items []search.Item
	if err := json.Unmarshal(searchBytes, &items); err != nil {
		t.Fatal(err)
	}
	var foundPost, foundTag, foundCategory bool
	for _, item := range items {
		switch item.URL {
		case "/site/zh/page/":
			foundPost = slices.Equal(item.Tags, []string{"起始", "说明"}) &&
				slices.Equal(item.TagURLs, []string{"/site/tags/%E8%B5%B7%E5%A7%8B/", "/site/categories/%E8%AF%B4%E6%98%8E/"})
		case "/site/tags/%E8%B5%B7%E5%A7%8B/":
			foundTag = item.Title == "Tag: 起始"
		case "/site/categories/%E8%AF%B4%E6%98%8E/":
			foundCategory = item.Title == "Category: 说明"
		}
	}
	if !foundPost || !foundTag || !foundCategory {
		t.Errorf("search index missing native-language archive URLs: post=%v tag=%v category=%v", foundPost, foundTag, foundCategory)
	}
	sitemapBytes, err := os.ReadFile(filepath.Join(cfg.OutputDir, "sitemap.xml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"https://example.com/site/tags/%E8%B5%B7%E5%A7%8B/", "https://example.com/site/categories/%E8%AF%B4%E6%98%8E/"} {
		if !strings.Contains(string(sitemapBytes), want) {
			t.Errorf("sitemap missing %s: %s", want, sitemapBytes)
		}
	}
}
