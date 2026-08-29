package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
)

func setupTestSite(t *testing.T) (config.Config, string) {
	t.Helper()
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	templateDir := filepath.Join(tempDir, "templates")
	assetDir := filepath.Join(tempDir, "assets")
	outputDir := filepath.Join(tempDir, "public")

	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(assetDir, 0755); err != nil {
		t.Fatal(err)
	}

	templatePath := filepath.Join(templateDir, "layout.html")
	if err := os.WriteFile(templatePath, []byte("<!DOCTYPE html><html><body>{{.Content}}</body></html>"), 0600); err != nil {
		t.Fatal(err)
	}

	page1Path := filepath.Join(contentDir, "page1.md")
	if err := os.WriteFile(page1Path, []byte("# Page One\nInitial content."), 0600); err != nil {
		t.Fatal(err)
	}

	page2Path := filepath.Join(contentDir, "page2.md")
	if err := os.WriteFile(page2Path, []byte("# Page Two\nSecond page content."), 0600); err != nil {
		t.Fatal(err)
	}

	stylePath := filepath.Join(assetDir, "style.css")
	if err := os.WriteFile(stylePath, []byte("body { color: black; }"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.Template = templatePath
	cfg.AssetDir = assetDir
	cfg.OutputDir = outputDir
	cfg.ProjectRoot = tempDir
	cfg.SiteURL = "https://example.com"
	cfg.SiteName = "Test Site"

	return cfg, tempDir
}

func TestCacheInvalidationMatrix(t *testing.T) {
	t.Run("1_UnchangedMarkdown_ProducesCacheHit", func(t *testing.T) {
		cfg, _ := setupTestSite(t)

		res1, err := Build(cfg)
		if err != nil {
			t.Fatalf("Initial build failed: %v", err)
		}
		if res1.CacheHit {
			t.Errorf("Initial build should be cache miss, got cache hit")
		}

		res2, err := Build(cfg)
		if err != nil {
			t.Fatalf("Second build failed: %v", err)
		}
		if !res2.CacheHit {
			t.Errorf("Unchanged site build should be cache hit, got cache miss")
		}
	})

	t.Run("2_ChangedMarkdown_TriggersRebuild", func(t *testing.T) {
		cfg, _ := setupTestSite(t)

		res1, err := Build(cfg)
		if err != nil || res1.CacheHit {
			t.Fatalf("Initial build failed: err=%v, cacheHit=%v", err, res1.CacheHit)
		}

		page1Path := filepath.Join(cfg.ContentDir, "page1.md")
		if err := os.WriteFile(page1Path, []byte("# Page One\nUpdated content modification."), 0600); err != nil {
			t.Fatal(err)
		}

		res2, err := Build(cfg)
		if err != nil {
			t.Fatalf("Rebuild after markdown edit failed: %v", err)
		}
		if res2.CacheHit {
			t.Errorf("Changed markdown should trigger rebuild (cache miss), got cache hit")
		}

		outHTML, err := os.ReadFile(filepath.Join(cfg.OutputDir, "page1", "index.html"))
		if err != nil {
			t.Fatalf("Failed to read generated page: %v", err)
		}
		if !strings.Contains(string(outHTML), "Updated content modification") {
			t.Errorf("Generated output does not contain updated markdown text")
		}
	})

	t.Run("3_DeletedMarkdown_RemovesGeneratedPage", func(t *testing.T) {
		cfg, _ := setupTestSite(t)

		res1, err := Build(cfg)
		if err != nil || res1.CacheHit {
			t.Fatalf("Initial build failed: err=%v, cacheHit=%v", err, res1.CacheHit)
		}

		page2Out := filepath.Join(cfg.OutputDir, "page2", "index.html")
		if _, err := os.Stat(page2Out); err != nil {
			t.Fatalf("Expected generated page2 to exist: %v", err)
		}

		// Delete page2.md
		page2Path := filepath.Join(cfg.ContentDir, "page2.md")
		if err := os.Remove(page2Path); err != nil {
			t.Fatal(err)
		}

		res2, err := Build(cfg)
		if err != nil {
			t.Fatalf("Rebuild after deleting markdown failed: %v", err)
		}
		if res2.CacheHit {
			t.Errorf("Deleting markdown should trigger rebuild (cache miss), got cache hit")
		}

		// Verify page2 output is gone
		if _, err := os.Stat(page2Out); !os.IsNotExist(err) {
			t.Errorf("Deleted markdown output page still exists after build: %v", err)
		}

		// Verify search.json does not reference page2
		searchJSON, err := os.ReadFile(filepath.Join(cfg.OutputDir, "search.json"))
		if err != nil {
			t.Fatalf("Failed to read search.json: %v", err)
		}
		if strings.Contains(string(searchJSON), "page2") || strings.Contains(string(searchJSON), "Page Two") {
			t.Errorf("Deleted page still present in search.json: %s", string(searchJSON))
		}

		// Verify taxonomy/gen pages do not retain deleted content: search should only contain page1 entries
		if strings.Contains(string(searchJSON), "Second page content") {
			t.Errorf("Deleted page content still present in search.json")
		}
	})

	t.Run("4_ChangedTemplates_TriggersRebuild", func(t *testing.T) {
		cfg, _ := setupTestSite(t)

		res1, err := Build(cfg)
		if err != nil || res1.CacheHit {
			t.Fatalf("Initial build failed: err=%v, cacheHit=%v", err, res1.CacheHit)
		}

		// Modify template
		newTmpl := []byte(`<!DOCTYPE html><html class="theme-v2"><body><main>{{.Content}}</main></body></html>`)
		if err := os.WriteFile(cfg.Template, newTmpl, 0600); err != nil {
			t.Fatal(err)
		}

		res2, err := Build(cfg)
		if err != nil {
			t.Fatalf("Rebuild after template edit failed: %v", err)
		}
		if res2.CacheHit {
			t.Errorf("Changed template should trigger rebuild (cache miss), got cache hit")
		}

		outHTML, err := os.ReadFile(filepath.Join(cfg.OutputDir, "page1", "index.html"))
		if err != nil {
			t.Fatalf("Failed to read generated page: %v", err)
		}
		if !strings.Contains(string(outHTML), "theme-v2") {
			t.Errorf("Generated output does not reflect template modifications")
		}
		// Verify all pages re-rendered with new template
		out2, err := os.ReadFile(filepath.Join(cfg.OutputDir, "page2", "index.html"))
		if err != nil {
			t.Fatalf("Failed to read page2: %v", err)
		}
		if !strings.Contains(string(out2), "theme-v2") {
			t.Errorf("Second page not re-rendered with new template")
		}
	})

	t.Run("5_ChangedAssets_TriggersExpectedOutputUpdate", func(t *testing.T) {
		cfg, _ := setupTestSite(t)

		res1, err := Build(cfg)
		if err != nil || res1.CacheHit {
			t.Fatalf("Initial build failed: err=%v, cacheHit=%v", err, res1.CacheHit)
		}

		assetOut := filepath.Join(cfg.OutputDir, "assets", "style.css")
		styleData, err := os.ReadFile(assetOut)
		if err != nil || !strings.Contains(string(styleData), "color: black") {
			t.Fatalf("Initial asset copy missing or incorrect: %v", err)
		}

		// 5a. Modify asset
		stylePath := filepath.Join(cfg.AssetDir, "style.css")
		if err := os.WriteFile(stylePath, []byte("body { color: red; }"), 0600); err != nil {
			t.Fatal(err)
		}

		res2, err := Build(cfg)
		if err != nil {
			t.Fatalf("Rebuild after asset edit failed: %v", err)
		}
		if res2.CacheHit {
			t.Errorf("Changed asset should trigger rebuild (cache miss), got cache hit")
		}

		styleDataUpdated, err := os.ReadFile(assetOut)
		if err != nil || !strings.Contains(string(styleDataUpdated), "color: red") {
			t.Errorf("Asset output was not updated: %v", err)
		}

		// 5b. Add new asset
		scriptPath := filepath.Join(cfg.AssetDir, "app.js")
		if err := os.WriteFile(scriptPath, []byte("console.log('test');"), 0600); err != nil {
			t.Fatal(err)
		}

		res3, err := Build(cfg)
		if err != nil || res3.CacheHit {
			t.Fatalf("Rebuild after adding asset failed: err=%v, cacheHit=%v", err, res3.CacheHit)
		}
		scriptOut := filepath.Join(cfg.OutputDir, "assets", "app.js")
		if _, err := os.Stat(scriptOut); err != nil {
			t.Errorf("Newly added asset missing from output: %v", err)
		}
		scriptData, err := os.ReadFile(scriptOut)
		if err != nil || !strings.Contains(string(scriptData), "console.log") {
			t.Errorf("New asset content incorrect: %v", err)
		}

		// 5c. Delete asset
		if err := os.Remove(scriptPath); err != nil {
			t.Fatal(err)
		}
		res4, err := Build(cfg)
		if err != nil || res4.CacheHit {
			t.Fatalf("Rebuild after deleting asset failed: err=%v, cacheHit=%v", err, res4.CacheHit)
		}
		if _, err := os.Stat(scriptOut); !os.IsNotExist(err) {
			t.Errorf("Deleted asset still exists in output: %v", err)
		}
		// Original asset should still be present
		if _, err := os.Stat(assetOut); err != nil {
			t.Errorf("Original asset missing after deleting added asset: %v", err)
		}
	})

	t.Run("6_ChangedConfiguration_TriggersRebuild", func(t *testing.T) {
		cfg, _ := setupTestSite(t)

		res1, err := Build(cfg)
		if err != nil || res1.CacheHit {
			t.Fatalf("Initial build failed: err=%v, cacheHit=%v", err, res1.CacheHit)
		}

		// Initial graph explorer files must exist (default true)
		for _, rel := range []string{"graph/index.html", "graph/data.json"} {
			p := filepath.Join(cfg.OutputDir, filepath.FromSlash(rel))
			if _, err := os.Stat(p); err != nil {
				t.Fatalf("Expected %s to exist with GraphExplorer enabled: %v", rel, err)
			}
		}
		// Initial search.json should contain /page1/ without extra base path
		initialSearch, err := os.ReadFile(filepath.Join(cfg.OutputDir, "search.json"))
		if err != nil {
			t.Fatalf("read initial search.json: %v", err)
		}
		if !strings.Contains(string(initialSearch), "/page1/") {
			t.Fatalf("initial search.json missing /page1/: %s", string(initialSearch))
		}

		// 6a. SiteName change
		cfg.SiteName = "Renamed Site Title"
		res2, err := Build(cfg)
		if err != nil {
			t.Fatalf("Rebuild after SiteName edit failed: %v", err)
		}
		if res2.CacheHit {
			t.Errorf("Changed configuration (SiteName) should trigger rebuild, got cache hit")
		}
		graphIndex, err := os.ReadFile(filepath.Join(cfg.OutputDir, "graph", "index.html"))
		if err != nil {
			t.Fatalf("graph/index.html missing after SiteName change: %v", err)
		}
		if !strings.Contains(string(graphIndex), "Renamed Site Title") {
			t.Errorf("SiteName change not reflected in graph/index.html: %s", string(graphIndex)[:800])
		}

		// 6b. Theme change
		cfg.Theme = "dark"
		res3, err := Build(cfg)
		if err != nil {
			t.Fatalf("Rebuild after Theme edit failed: %v", err)
		}
		if res3.CacheHit {
			t.Errorf("Changed configuration (Theme) should trigger rebuild, got cache hit")
		}

		// 6c. SiteURL change — verify search.json and sitemap.xml reflect new base
		cfg.SiteURL = "https://new.example.com/subpath"
		res4, err := Build(cfg)
		if err != nil {
			t.Fatalf("Rebuild after SiteURL edit failed: %v", err)
		}
		if res4.CacheHit {
			t.Errorf("Changed configuration (SiteURL) should trigger rebuild, got cache hit")
		}
		searchAfterURL, err := os.ReadFile(filepath.Join(cfg.OutputDir, "search.json"))
		if err != nil {
			t.Fatalf("read search.json after SiteURL change: %v", err)
		}
		sSearch := string(searchAfterURL)
		if !strings.Contains(sSearch, "/subpath/page1/") {
			t.Errorf("search.json URLs do not reflect new SiteURL base /subpath/: %s", sSearch)
		}
		if !strings.Contains(sSearch, "/subpath/page2/") {
			t.Errorf("search.json missing /subpath/page2/ after SiteURL change: %s", sSearch)
		}
		sitemap, err := os.ReadFile(filepath.Join(cfg.OutputDir, "sitemap.xml"))
		if err != nil {
			t.Fatalf("read sitemap.xml: %v", err)
		}
		sSitemap := string(sitemap)
		if !strings.Contains(sSitemap, "https://new.example.com/subpath/page1/") {
			t.Errorf("sitemap.xml does not reflect new SiteURL base: %s", sSitemap)
		}
		if !strings.Contains(sSitemap, "https://new.example.com/subpath/page2/") {
			t.Errorf("sitemap.xml missing page2 with new base: %s", sSitemap)
		}
		// Ensure old base not lingering as absolute URL (e.g., https://example.com/page1/)
		if strings.Contains(sSitemap, "https://example.com/page1/") {
			t.Errorf("sitemap.xml still contains old SiteURL: %s", sSitemap)
		}

		// 6d. GraphExplorer toggle off -> files disappear
		cfg.GraphExplorer = false
		res5, err := Build(cfg)
		if err != nil {
			t.Fatalf("Rebuild after disabling GraphExplorer failed: %v", err)
		}
		if res5.CacheHit {
			t.Errorf("GraphExplorer disable should trigger rebuild, got cache hit")
		}
		for _, rel := range []string{"graph/index.html", "graph/data.json"} {
			p := filepath.Join(cfg.OutputDir, filepath.FromSlash(rel))
			if _, err := os.Stat(p); !os.IsNotExist(err) {
				t.Errorf("GraphExplorer disabled but %s still exists: %v", rel, err)
			}
		}

		// 6e. GraphExplorer toggle on -> files reappear
		cfg.GraphExplorer = true
		res6, err := Build(cfg)
		if err != nil {
			t.Fatalf("Rebuild after re-enabling GraphExplorer failed: %v", err)
		}
		if res6.CacheHit {
			t.Errorf("GraphExplorer re-enable should trigger rebuild, got cache hit")
		}
		for _, rel := range []string{"graph/index.html", "graph/data.json"} {
			p := filepath.Join(cfg.OutputDir, filepath.FromSlash(rel))
			if _, err := os.Stat(p); err != nil {
				t.Errorf("GraphExplorer re-enabled but %s missing: %v", rel, err)
			}
		}
		// Verify graph/data.json is valid JSON and contains nodes
		graphData, err := os.ReadFile(filepath.Join(cfg.OutputDir, "graph", "data.json"))
		if err != nil {
			t.Fatalf("read graph/data.json: %v", err)
		}
		if !strings.Contains(string(graphData), "\"nodes\"") {
			t.Errorf("graph/data.json missing nodes: %s", string(graphData)[:500])
		}
	})

	t.Run("7_RemovedGeneratedArtifacts_DoNotSurviveLaterBuild", func(t *testing.T) {
		cfg, _ := setupTestSite(t)

		res1, err := Build(cfg)
		if err != nil || res1.CacheHit {
			t.Fatalf("Initial build failed: err=%v, cacheHit=%v", err, res1.CacheHit)
		}

		searchPath := filepath.Join(cfg.OutputDir, "search.json")
		if _, err := os.Stat(searchPath); err != nil {
			t.Fatalf("Expected search.json in output: %v", err)
		}

		// 7a. Remove generated artifact from outputDir -> forces rebuild and restores file
		if err := os.Remove(searchPath); err != nil {
			t.Fatal(err)
		}

		res2, err := Build(cfg)
		if err != nil {
			t.Fatalf("Rebuild after removing generated file failed: %v", err)
		}
		if res2.CacheHit {
			t.Errorf("Missing generated file in output directory should invalidate cache, got cache hit")
		}
		if _, err := os.Stat(searchPath); err != nil {
			t.Errorf("Removed generated artifact search.json was not restored: %v", err)
		}

		// 7b. Add untracked/orphan file to outputDir -> cache miss cleans it up
		orphanPath := filepath.Join(cfg.OutputDir, "stale_artifact.txt")
		if err := os.WriteFile(orphanPath, []byte("stale data"), 0600); err != nil {
			t.Fatal(err)
		}

		// Rebuild site (orphan file should invalidate cache due to file count/list mismatch)
		res3, err := Build(cfg)
		if err != nil {
			t.Fatalf("Rebuild with orphan file failed: %v", err)
		}
		if res3.CacheHit {
			t.Errorf("Presence of orphan file in output directory should cause cache miss, got cache hit")
		}
		if _, err := os.Stat(orphanPath); !os.IsNotExist(err) {
			t.Errorf("Orphan artifact survived build: %v", err)
		}
	})
}

// cacheFingerprint must not mutate the input Config.
func TestCacheFingerprintDoesNotMutateConfig(t *testing.T) {
	cfg, _ := setupTestSite(t)
	cfg.WatchMode = true

	_, err := cacheFingerprint(cfg)
	if err != nil {
		t.Fatalf("cacheFingerprint failed: %v", err)
	}
	if !cfg.WatchMode {
		t.Error("cacheFingerprint mutated cfg.WatchMode to false")
	}
}
