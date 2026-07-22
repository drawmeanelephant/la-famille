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
	expectedSearchURLs := map[string]string{
		"/tags/index.html":            "Tags",
		"/tags/go/index.html":         "Tag: go",
		"/tags/web/index.html":        "Tag: web",
		"/categories/index.html":      "Categories",
		"/categories/news/index.html": "Category: news",
		"/categories/tech/index.html": "Category: tech",
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
