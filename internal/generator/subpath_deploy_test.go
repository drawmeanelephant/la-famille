package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/runtimeassets"
)

// TestSubpathDeployRebasesOutput guards Issue #528: a siteurl that points at a
// subpath (e.g. a GitHub Pages project site at https://<user>.github.io/<repo>)
// used to rebase only canonical/og:url while leaving every /assets/ link and
// the "/" home link root-absolute, so every subresource 404ed on deploy.
//
// The build must produce an artifact whose rendered HTML links, injected
// base-path meta, canonical tags, and machine-readable consumers all agree the
// site lives under /my-site — the base path a subpath server serves.
func TestSubpathDeployRebasesOutput(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	templateDir := filepath.Join(dir, "templates")
	assetDir := filepath.Join(dir, "assets")
	outputDir := filepath.Join(dir, "public")

	for _, d := range []string{contentDir, templateDir, assetDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}
	// The default layout references /assets/css/*, /assets/js/search.js and a
	// mascot image; lay those down so the build copies them and the rebase has
	// concrete links to prove. Exact filenames don't matter — only that a
	// root-relative asset makes it into the rendered HTML.
	for _, sub := range []string{"css", "js", "img"} {
		if err := os.MkdirAll(filepath.Join(assetDir, sub), 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(assetDir, "css", "theme.css"), []byte("body{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "js", "search.js"), []byte(""), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "img", "mascot-default.jpeg"), []byte("img"), 0600); err != nil {
		t.Fatal(err)
	}

	// A template shaped like the bundled themes: <head> with asset links, a
	// home "/" nav link, and a body the generator fills.
	tmpl, err := runtimeassets.DefaultTemplate()
	if err != nil {
		t.Fatalf("read embedded default template: %v", err)
	}
	templatePath := filepath.Join(templateDir, "layout.html")
	if err := os.WriteFile(templatePath, tmpl, 0600); err != nil {
		t.Fatal(err)
	}
	// The default layout invokes the search_modal partial, which ships embedded
	// with the binary and is installed into templates/partials by init.
	partials, err := runtimeassets.DefaultPartials()
	if err != nil {
		t.Fatalf("read embedded partials: %v", err)
	}
	for name, data := range partials {
		dest := filepath.Join(templateDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dest, data, 0600); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.WriteFile(filepath.Join(contentDir, "index.md"),
		[]byte("---\ntitle: Home\ndescription: d\n---\n# Home\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "blog.md"),
		[]byte("---\ntitle: A Post\ndescription: d\ndate: 2026-08-27\n---\n# A Post\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.Template = templatePath
	cfg.AssetDir = assetDir
	cfg.OutputDir = outputDir
	cfg.ProjectRoot = dir
	cfg.SiteURL = "https://user.github.io/my-site/"

	if _, err := Build(cfg); err != nil {
		t.Fatalf("Build with subpath siteurl failed: %v", err)
	}

	home, err := os.ReadFile(filepath.Join(outputDir, "index.html"))
	if err != nil {
		t.Fatalf("read index.html: %v", err)
	}
	post, err := os.ReadFile(filepath.Join(outputDir, "blog", "index.html"))
	if err != nil {
		t.Fatalf("read blog/index.html: %v", err)
	}

	homeStr := string(home)
	postStr := string(post)

	// Rendered HTML links and asset references must live under the base path.
	for _, want := range []string{
		`href="/my-site/assets/css/theme.css"`,
		`href="/my-site/"`, // home nav link
		`src="/my-site/assets/js/search.js"`,
	} {
		if !strings.Contains(homeStr, want) {
			t.Errorf("index.html missing %q (asset/home links not rebased):\n%s", want, homeStr)
		}
		if !strings.Contains(postStr, want) {
			t.Errorf("blog/index.html missing %q:\n%s", want, postStr)
		}
	}

	// A client-script hook must be injected so search.js etc. can fetch under
	// the base path, and it must land inside <head>.
	headIdx := strings.Index(homeStr, "</head>")
	metaIdx := strings.Index(homeStr, `name="la-famille-base-path" content="/my-site"`)
	if metaIdx < 0 || headIdx < 0 || metaIdx > headIdx {
		t.Errorf("base-path meta not injected into index.html <head>:\n%s", homeStr)
	}

	// Canonical and og:url keep the absolute public URL, subpath included —
	// the rebase must not double it, and a bare-root canonical is a 404 on a
	// project page.
	if !strings.Contains(postStr, `rel="canonical" href="https://user.github.io/my-site/blog/"`) {
		t.Errorf("canonical not absolute-with-base in blog/index.html:\n%s", postStr)
	}
	if strings.Contains(postStr, `href="https://user.github.io/my-site/my-site/`) {
		t.Errorf("canonical/rebase double-applied the base path:\n%s", postStr)
	}

	// Machine-readable consumers (search index, sitemap) agree on the base path,
	// which is exactly what a subpath server must serve.
	search, err := os.ReadFile(filepath.Join(outputDir, "search.json"))
	if err != nil {
		t.Fatalf("read search.json: %v", err)
	}
	if !strings.Contains(string(search), `"/my-site/blog/"`) {
		t.Errorf("search.json URL not under base path:\n%s", search)
	}
	sitemap, err := os.ReadFile(filepath.Join(outputDir, "sitemap.xml"))
	if err != nil {
		t.Fatalf("read sitemap.xml: %v", err)
	}
	if !strings.Contains(string(sitemap), "https://user.github.io/my-site/blog/") {
		t.Errorf("sitemap <loc> not absolute-with-base:\n%s", sitemap)
	}
}
