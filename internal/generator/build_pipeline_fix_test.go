package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
)

// flatLayoutSite lays out a site whose content_dir IS the project root
// (content_dir: "."), so the build's own bookkeeping — the cache file, the
// output directory and the staging/previous swap directories — lives inside
// the hashed tree.
func flatLayoutSite(t *testing.T) config.Config {
	t.Helper()
	root := t.TempDir()
	templateDir := filepath.Join(root, "templates")
	assetDir := filepath.Join(root, "assets")
	for _, d := range []string{templateDir, assetDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	templatePath := filepath.Join(templateDir, "layout.html")
	if err := os.WriteFile(templatePath, []byte("<!DOCTYPE html><html><body>{{.Content}}</body></html>"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.md"), []byte("---\ntitle: Home\n---\nHello."), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "site.css"), []byte("body{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ProjectRoot = root
	cfg.ContentDir = root
	cfg.OutputDir = filepath.Join(root, "public")
	cfg.AssetDir = assetDir
	cfg.Template = templatePath
	cfg.SiteURL = "https://example.com"
	return cfg
}

// TestFlatLayoutCachesAcrossBuilds guards #633: on a flat layout the
// fingerprint used to hash the cache file it had just written (and any
// staging/previous sibling left behind), so every build hashed a different
// tree and the cache could never hit — which is what let serve --watch loop.
func TestFlatLayoutCachesAcrossBuilds(t *testing.T) {
	cfg := flatLayoutSite(t)

	res, err := Build(cfg)
	if err != nil {
		t.Fatalf("first Build() error: %v", err)
	}
	if res.CacheHit {
		t.Fatal("first build should be a cache miss")
	}

	// A transient directory left behind by an interrupted build must not
	// poison the fingerprint either.
	linger := filepath.Join(cfg.ProjectRoot, ".public.staging-linger123")
	if err := os.MkdirAll(linger, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(linger, "meta.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	res, err = Build(cfg)
	if err != nil {
		t.Fatalf("second Build() error: %v", err)
	}
	if !res.CacheHit {
		t.Fatal("unchanged flat-layout build should be a cache hit")
	}
	if err := os.RemoveAll(linger); err != nil {
		t.Fatal(err)
	}

	res, err = Build(cfg)
	if err != nil {
		t.Fatalf("third Build() error: %v", err)
	}
	if !res.CacheHit {
		t.Fatal("third unchanged build should stay a cache hit")
	}
}

// TestWatchModeChangesFingerprint guards #634: watch mode injects the
// livereload script into rendered pages, so flipping WatchMode must
// invalidate the cache. Excluding it meant `serve --watch` after `build`
// served pages without livereload, and `build` after `serve --watch` shipped
// the watch-only script in the production artifact.
func TestWatchModeChangesFingerprint(t *testing.T) {
	cfg, _ := setupTestSite(t)
	page := filepath.Join(cfg.OutputDir, "page1", "index.html")

	res, err := Build(cfg)
	if err != nil {
		t.Fatalf("build error: %v", err)
	}
	if res.CacheHit {
		t.Fatal("first build should be a cache miss")
	}
	if data, err := os.ReadFile(page); err != nil || strings.Contains(string(data), "EventSource") {
		t.Fatalf("production page must not contain the livereload script (err=%v)", err)
	}

	cfg.WatchMode = true
	res, err = Build(cfg)
	if err != nil {
		t.Fatalf("watch build error: %v", err)
	}
	if res.CacheHit {
		t.Fatal("build -> serve --watch must invalidate the cache: served pages have no livereload otherwise")
	}
	if data, err := os.ReadFile(page); err != nil || !strings.Contains(string(data), "EventSource") {
		t.Fatalf("watch-mode page must contain the livereload script (err=%v)", err)
	}

	res, err = Build(cfg)
	if err != nil {
		t.Fatalf("repeat watch build error: %v", err)
	}
	if !res.CacheHit {
		t.Fatal("unchanged watch-mode build should be a cache hit")
	}

	cfg.WatchMode = false
	res, err = Build(cfg)
	if err != nil {
		t.Fatalf("production build error: %v", err)
	}
	if res.CacheHit {
		t.Fatal("serve --watch -> build must invalidate the cache: the livereload script would ship in the artifact")
	}
	data, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "EventSource") {
		t.Fatal("production artifact contains the watch-mode livereload script")
	}
}

// TestPublishArtifactModesUniform guards #637: pages and copied assets were
// published 0644 while feed.xml, sitemap.xml, robots.txt, graph.json,
// meta.json, the manifest, the diff ledger and raw render:false markdown
// copies went out 0600 — unreadable to a web server running as another user.
// Everything the build publishes must carry one mode: 0644 as masked by the
// process umask, measured against a control file rather than assumed.
func TestPublishArtifactModesUniform(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	templateDir := filepath.Join(dir, "templates")
	assetDir := filepath.Join(dir, "assets")
	for _, d := range []string{contentDir, templateDir, assetDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	templatePath := filepath.Join(templateDir, "layout.html")
	if err := os.WriteFile(templatePath, []byte("<!DOCTYPE html><html><body>{{.Content}}</body></html>"), 0o600); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"index.md": "---\ntitle: Home\n---\n[a](a.md) [missing](missing.md) [b](b.md)\n",
		"a.md":     "---\ntitle: A\ndate: 2026-01-02\ntags: [x]\n---\nbody\n",
		"b.md":     "---\ntitle: B\nrender: false\n---\nraw\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(contentDir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	cfg := config.DefaultConfig()
	cfg.ProjectRoot = dir
	cfg.ContentDir = contentDir
	cfg.Template = templatePath
	cfg.AssetDir = assetDir
	cfg.OutputDir = filepath.Join(dir, "public")
	cfg.SiteURL = "https://example.com"

	if _, err := Build(cfg); err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	// A control file written 0644 through the same umask is the expected mode,
	// so the assertion holds under any runner umask.
	control := filepath.Join(cfg.OutputDir, "mode-control.tmp")
	// #nosec G306 -- the control deliberately measures the published 0644 mode
	if err := os.WriteFile(control, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(control)
	if err != nil {
		t.Fatal(err)
	}
	want := info.Mode().Perm()
	defer os.Remove(control)

	err = filepath.WalkDir(cfg.OutputDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !d.Type().IsRegular() || path == control {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if got := info.Mode().Perm(); got != want {
			t.Errorf("%s mode %04o, want %04o", path, got, want)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
