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
