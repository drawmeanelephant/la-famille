package taxonomy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/microcosm-cc/bluemonday"
	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/render"
)

func TestGenerateTags(t *testing.T) {
	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "public")
	tmplDir := filepath.Join(tempDir, "templates")

	_ = os.MkdirAll(outDir, 0755)
	_ = os.MkdirAll(tmplDir, 0755)

	tmplPath := filepath.Join(tmplDir, "layout.html")
	_ = os.WriteFile(tmplPath, []byte("{{.Content}}"), 0600)

	cfg := config.Config{
		OutputDir: outDir,
		Template:  tmplPath,
	}

	renderTrue := true
	fileMap := map[string]*content.FileMeta{
		"post1.md": {Title: "Post 1", Tags: []string{"go", "web"}, Render: &renderTrue},
		"post2.md": {Title: "Post 2", Tags: []string{"go"}, Render: &renderTrue},
	}

	renderer := render.New(tmplDir)
	p := bluemonday.UGCPolicy()

	paths, items, err := GenerateTags(cfg, cfg, fileMap, renderer, p)
	if err != nil {
		t.Fatalf("GenerateTags failed: %v", err)
	}

	if len(paths) != 3 {
		t.Fatalf("expected 3 tag paths, got %d: %v", len(paths), paths)
	}

	if len(items) != 3 {
		t.Fatalf("expected 3 tag search items, got %d: %v", len(items), items)
	}

	// Check if go tag page was created
	goTagPath := filepath.Join(outDir, "tags", "go", "index.html")
	b, err := os.ReadFile(goTagPath)
	if err != nil {
		t.Fatalf("expected tags/go/index.html to exist: %v", err)
	}
	html := string(b)
	if !strings.Contains(html, "<h2>Tag: go</h2>") {
		t.Errorf("expected tag title, got: %s", html)
	}
	if !strings.Contains(html, `href="../../post1/"`) {
		t.Errorf("expected link to post1, got: %s", html)
	}
	if !strings.Contains(html, `href="../../post2/"`) {
		t.Errorf("expected link to post2, got: %s", html)
	}
}

func TestGenerateTaxonomies_TagsAndCategories(t *testing.T) {
	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "public")
	tmplDir := filepath.Join(tempDir, "templates")

	_ = os.MkdirAll(outDir, 0755)
	_ = os.MkdirAll(tmplDir, 0755)

	tmplPath := filepath.Join(tmplDir, "layout.html")
	_ = os.WriteFile(tmplPath, []byte("{{.Content}}"), 0600)

	cfg := config.Config{
		OutputDir: outDir,
		Template:  tmplPath,
	}

	renderTrue := true
	fileMap := map[string]*content.FileMeta{
		"blog/post1.md": {Title: "First Post", Tags: []string{"go", "web"}, Categories: []string{"tech"}, Render: &renderTrue},
		"blog/post2.md": {Title: "Second Post", Tags: []string{"go"}, Categories: []string{"tech", "news"}, Render: &renderTrue},
	}

	renderer := render.New(tmplDir)
	p := bluemonday.UGCPolicy()

	paths, items, err := GenerateTaxonomies(cfg, cfg, fileMap, renderer, p)
	if err != nil {
		t.Fatalf("GenerateTaxonomies failed: %v", err)
	}

	expectedPaths := []string{
		"categories/index.html",
		"categories/news/index.html",
		"categories/tech/index.html",
		"tags/go/index.html",
		"tags/index.html",
		"tags/web/index.html",
	}

	if len(paths) != len(expectedPaths) {
		t.Fatalf("got %d generated paths, want %d: %v", len(paths), len(expectedPaths), paths)
	}
	for i, expected := range expectedPaths {
		if paths[i] != expected {
			t.Errorf("path[%d] = %q, want %q", i, paths[i], expected)
		}
	}

	if len(items) != 6 {
		t.Fatalf("expected 6 search items, got %d: %v", len(items), items)
	}

	// Verify search items details
	itemByURL := make(map[string]string)
	for _, it := range items {
		itemByURL[it.URL] = it.Title
	}
	// Clean public paths, matching what the canonical link, sitemap and feed
	// advertise for the same pages. Search results used to carry the raw output
	// filename instead, which also dropped any siteurl base path.
	expectedSearchURLs := map[string]string{
		"/tags/":            "Tags",
		"/tags/go/":         "Tag: go",
		"/tags/web/":        "Tag: web",
		"/categories/":      "Categories",
		"/categories/news/": "Category: news",
		"/categories/tech/": "Category: tech",
	}
	for url, expectedTitle := range expectedSearchURLs {
		if title, ok := itemByURL[url]; !ok {
			t.Errorf("missing search item for URL %q", url)
		} else if title != expectedTitle {
			t.Errorf("search item %q title = %q, want %q", url, title, expectedTitle)
		}
	}

	// Verify categories/index.html contents
	catIndexBytes, err := os.ReadFile(filepath.Join(outDir, "categories", "index.html"))
	if err != nil {
		t.Fatalf("failed to read categories/index.html: %v", err)
	}
	catIndex := string(catIndexBytes)
	if !strings.Contains(catIndex, `<h2>Categories</h2>`) {
		t.Errorf("categories/index.html missing heading: %s", catIndex)
	}
	if !strings.Contains(catIndex, `href="news/"`) || !strings.Contains(catIndex, `href="tech/"`) {
		t.Errorf("categories/index.html missing category links: %s", catIndex)
	}

	// Verify categories/tech/index.html contents
	techCatBytes, err := os.ReadFile(filepath.Join(outDir, "categories", "tech", "index.html"))
	if err != nil {
		t.Fatalf("failed to read categories/tech/index.html: %v", err)
	}
	techCat := string(techCatBytes)
	if !strings.Contains(techCat, `<h2>Category: tech</h2>`) {
		t.Errorf("categories/tech/index.html missing heading: %s", techCat)
	}
	if !strings.Contains(techCat, `href="../../blog/post1/"`) || !strings.Contains(techCat, `href="../../blog/post2/"`) {
		t.Errorf("categories/tech/index.html missing post links: %s", techCat)
	}
}

func TestGenerateTaxonomies_EmptyAndRenderFalse(t *testing.T) {
	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "public")
	tmplDir := filepath.Join(tempDir, "templates")

	_ = os.MkdirAll(outDir, 0755)
	_ = os.MkdirAll(tmplDir, 0755)

	tmplPath := filepath.Join(tmplDir, "layout.html")
	_ = os.WriteFile(tmplPath, []byte("{{.Content}}"), 0600)

	cfg := config.Config{
		OutputDir: outDir,
		Template:  tmplPath,
	}

	renderTrue := true
	renderFalse := false

	fileMap := map[string]*content.FileMeta{
		"hidden.md": {Title: "Hidden Page", Tags: []string{"secret", "  "}, Categories: []string{"internal"}, Render: &renderFalse},
		"public.md": {Title: "Public Page", Tags: []string{"visible", ""}, Categories: []string{"blog"}, Render: &renderTrue},
	}

	renderer := render.New(tmplDir)
	p := bluemonday.UGCPolicy()

	paths, items, err := GenerateTaxonomies(cfg, cfg, fileMap, renderer, p)
	if err != nil {
		t.Fatalf("GenerateTaxonomies failed: %v", err)
	}

	// Should not generate pages for 'secret' or 'internal'
	for _, p := range paths {
		if strings.Contains(p, "secret") || strings.Contains(p, "internal") {
			t.Errorf("unexpected path for render:false page taxonomy: %s", p)
		}
	}
	for _, it := range items {
		if strings.Contains(it.URL, "secret") || strings.Contains(it.URL, "internal") {
			t.Errorf("unexpected search item for render:false page taxonomy: %v", it)
		}
	}

	secretPath := filepath.Join(outDir, "tags", "secret", "index.html")
	if _, err := os.Stat(secretPath); !os.IsNotExist(err) {
		t.Errorf("expected tag page for secret to not exist, but found it")
	}
}

func TestGenerateTaxonomies_EscapingAndOrdering(t *testing.T) {
	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "public")
	tmplDir := filepath.Join(tempDir, "templates")

	_ = os.MkdirAll(outDir, 0755)
	_ = os.MkdirAll(tmplDir, 0755)

	tmplPath := filepath.Join(tmplDir, "layout.html")
	_ = os.WriteFile(tmplPath, []byte("{{.Content}}"), 0600)

	cfg := config.Config{
		OutputDir: outDir,
		Template:  tmplPath,
	}

	renderTrue := true
	fileMap := map[string]*content.FileMeta{
		"b.md": {Title: "Page B <script>", Tags: []string{"xss", "xss"}, Render: &renderTrue},
		"a.md": {Title: "Page A & More", Tags: []string{"xss"}, Render: &renderTrue},
	}

	renderer := render.New(tmplDir)
	p := bluemonday.UGCPolicy()

	_, _, err := GenerateTaxonomies(cfg, cfg, fileMap, renderer, p)
	if err != nil {
		t.Fatalf("GenerateTaxonomies failed: %v", err)
	}

	xssTagPath := filepath.Join(outDir, "tags", "xss", "index.html")
	b, err := os.ReadFile(xssTagPath)
	if err != nil {
		t.Fatalf("expected tags/xss/index.html to exist: %v", err)
	}

	html := string(b)
	if strings.Contains(html, "<script>") {
		t.Errorf("un-escaped script tag found in HTML: %s", html)
	}
	if !strings.Contains(html, "Page A &amp; More") && !strings.Contains(html, "Page A & More") {
		t.Errorf("expected escaped title in HTML, got: %s", html)
	}

	// Verify deterministic order: Page A before Page B
	idxA := strings.Index(html, "a/")
	idxB := strings.Index(html, "b/")
	if idxA == -1 || idxB == -1 || idxA > idxB {
		t.Errorf("expected Page A link before Page B link, got idxA=%d, idxB=%d in HTML:\n%s", idxA, idxB, html)
	}
}

// Issue #529: /tags/ and /categories/ must be reachable from the site nav when
// archive pages exist, and the links must never duplicate operator-configured
// entries.
func TestNavLinks_AddsArchiveLinksWhenPagesExist(t *testing.T) {
	renderTrue := true
	renderFalse := false

	t.Run("no terms adds nothing", func(t *testing.T) {
		fileMap := map[string]*content.FileMeta{
			"post.md": {Title: "Post", Render: &renderTrue},
		}
		if got := NavLinks(nil, fileMap); len(got) != 0 {
			t.Errorf("NavLinks(untagged site) = %v, want none", got)
		}
	})

	t.Run("tagged page adds Tags link", func(t *testing.T) {
		fileMap := map[string]*content.FileMeta{
			"post.md": {Title: "Post", Tags: []string{"go"}, Render: &renderTrue},
		}
		got := NavLinks(nil, fileMap)
		if len(got) != 1 || got[0].Label != "Tags" || got[0].URL != "/tags/" {
			t.Errorf("NavLinks(tagged) = %v, want [Tags /tags/]", got)
		}
	})

	t.Run("tags and categories both add links", func(t *testing.T) {
		fileMap := map[string]*content.FileMeta{
			"post.md": {Title: "Post", Tags: []string{"go"}, Categories: []string{"tech"}, Render: &renderTrue},
		}
		got := NavLinks(nil, fileMap)
		if len(got) != 2 || got[0].Label != "Tags" || got[1].Label != "Categories" {
			t.Errorf("NavLinks(tagged+categorized) = %v, want [Tags Categories]", got)
		}
	})

	t.Run("blank terms do not add a link", func(t *testing.T) {
		fileMap := map[string]*content.FileMeta{
			"post.md": {Title: "Post", Tags: []string{"  ", ""}, Render: &renderTrue},
		}
		if got := NavLinks(nil, fileMap); len(got) != 0 {
			t.Errorf("NavLinks(blank tags) = %v, want none", got)
		}
	})

	t.Run("render:false pages do not add a link", func(t *testing.T) {
		fileMap := map[string]*content.FileMeta{
			"post.md": {Title: "Post", Tags: []string{"secret"}, Render: &renderFalse},
		}
		if got := NavLinks(nil, fileMap); len(got) != 0 {
			t.Errorf("NavLinks(render:false tags) = %v, want none", got)
		}
	})
}

func TestNavLinks_DoesNotDuplicateOperatorLinks(t *testing.T) {
	renderTrue := true
	fileMap := map[string]*content.FileMeta{
		"post.md": {Title: "Post", Tags: []string{"go"}, Categories: []string{"tech"}, Render: &renderTrue},
	}

	tests := []struct {
		name  string
		links []config.SiteLink
		want  []string // final labels, in order
	}{
		{name: "no configured links", want: []string{"Tags", "Categories"}},
		{name: "label match skips tags", links: []config.SiteLink{{Label: "Tags", URL: "/elsewhere"}}, want: []string{"Tags", "Categories"}},
		{name: "case-insensitive label match", links: []config.SiteLink{{Label: "tags", URL: "/elsewhere"}}, want: []string{"tags", "Categories"}},
		{name: "root-relative URL match", links: []config.SiteLink{{Label: "Archive", URL: "/tags/"}}, want: []string{"Archive", "Categories"}},
		{name: "absolute URL match", links: []config.SiteLink{{Label: "Archive", URL: "https://example.com/tags/"}}, want: []string{"Archive", "Categories"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NavLinks(tt.links, fileMap)
			if len(got) != len(tt.want) {
				t.Fatalf("NavLinks(%v) returned %d links, want %d: %v", tt.links, len(got), len(tt.want), got)
			}
			for i, want := range tt.want {
				if got[i].Label != want {
					t.Errorf("NavLinks(%v)[%d].Label = %q, want %q (all: %v)", tt.links, i, got[i].Label, want, got)
				}
			}
		})
	}
}

// Issue #529: an article page must link each of its tags to that tag's archive.
func TestPageTagLinks(t *testing.T) {
	// The generator sanitizes content with a policy that keeps class
	// attributes; mirror it here so the class-based assertions hold.
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("class").Globally()

	t.Run("nil for no tags", func(t *testing.T) {
		if got := PageTagLinks(nil, "index.html", p); got != nil {
			t.Errorf("PageTagLinks(nil) = %q, want nil", got)
		}
		if got := PageTagLinks([]string{"", "  "}, "index.html", p); got != nil {
			t.Errorf("PageTagLinks(blank) = %q, want nil", got)
		}
	})

	t.Run("root page gets archive-relative hrefs", func(t *testing.T) {
		got := string(PageTagLinks([]string{"go", "web"}, "index.html", p))
		if !strings.Contains(got, `class="tag-link" href="tags/go/"`) {
			t.Errorf("PageTagLinks root = %q, want a tags/go/ link", got)
		}
		if !strings.Contains(got, `href="tags/web/"`) {
			t.Errorf("PageTagLinks root = %q, want a tags/web/ link", got)
		}
	})

	t.Run("nested page gets parent-relative hrefs", func(t *testing.T) {
		got := string(PageTagLinks([]string{"meta"}, "blog/post/index.html", p))
		if !strings.Contains(got, `href="../../tags/meta/"`) {
			t.Errorf("PageTagLinks nested = %q, want a ../../tags/meta/ link", got)
		}
	})

	t.Run("dedupes and trims", func(t *testing.T) {
		got := string(PageTagLinks([]string{"go", "go", "  ", "web"}, "index.html", p))
		if strings.Count(got, "tag-link") != 2 {
			t.Errorf("PageTagLinks dedupe = %q, want exactly two tag links", got)
		}
	})

	t.Run("escapes tag names and hrefs", func(t *testing.T) {
		got := string(PageTagLinks([]string{"<x>"}, "index.html", p))
		if strings.Contains(got, "<x>") || !strings.Contains(got, "&lt;x&gt;") {
			t.Errorf("PageTagLinks escaping = %q, want &lt;x&gt; text", got)
		}
	})
}
