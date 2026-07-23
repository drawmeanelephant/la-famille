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
	})

	t.Run("6_ChangedConfiguration_TriggersRebuild", func(t *testing.T) {
		cfg, _ := setupTestSite(t)

		res1, err := Build(cfg)
		if err != nil || res1.CacheHit {
			t.Fatalf("Initial build failed: err=%v, cacheHit=%v", err, res1.CacheHit)
		}

		cfg.SiteName = "Renamed Site Title"
		res2, err := Build(cfg)
		if err != nil {
			t.Fatalf("Rebuild after config edit failed: %v", err)
		}
		if res2.CacheHit {
			t.Errorf("Changed configuration (SiteName) should trigger rebuild, got cache hit")
		}

		cfg.Theme = "dark"
		res3, err := Build(cfg)
		if err != nil {
			t.Fatalf("Rebuild after theme config edit failed: %v", err)
		}
		if res3.CacheHit {
			t.Errorf("Changed configuration (Theme) should trigger rebuild, got cache hit")
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
