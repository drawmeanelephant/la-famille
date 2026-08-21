package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tbuddy/la-famille/internal/config"
)

// syntheticSite creates a temporary site with n markdown pages in a
// hermetic b.TempDir(). It mirrors setupTestSite from cache_invalidation_test.go
// but is parameterized and uses testing.B where needed.
func syntheticSite(b *testing.B, n int) config.Config {
	b.Helper()
	tempDir := b.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	templateDir := filepath.Join(tempDir, "templates")
	assetDir := filepath.Join(tempDir, "assets")
	outputDir := filepath.Join(tempDir, "public")

	for _, dir := range []string{contentDir, templateDir, assetDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			b.Fatal(err)
		}
	}

	templatePath := filepath.Join(templateDir, "layout.html")
	if err := os.WriteFile(templatePath, []byte("<!DOCTYPE html><html><body>{{.Content}}</body></html>"), 0600); err != nil {
		b.Fatal(err)
	}

	// One asset to exercise the asset hash path.
	if err := os.WriteFile(filepath.Join(assetDir, "style.css"), []byte("body { color: black; }"), 0600); err != nil {
		b.Fatal(err)
	}

	for i := 0; i < n; i++ {
		body := fmt.Sprintf("---\ntitle: Page %d\ntags: [\"bench\"]\n---\n# Page %d\n\nContent for page %d. See [page %d](page%d.md).\n", i, i, i, (i+1)%n, (i+1)%n)
		if err := os.WriteFile(filepath.Join(contentDir, fmt.Sprintf("page%d.md", i)), []byte(body), 0600); err != nil {
			b.Fatal(err)
		}
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.Template = templatePath
	cfg.AssetDir = assetDir
	cfg.OutputDir = outputDir
	cfg.ProjectRoot = tempDir
	cfg.SiteURL = "https://example.com"
	cfg.SiteName = "Bench Site"
	return cfg
}

func syntheticSiteT(t *testing.T, n int) config.Config {
	t.Helper()
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	templateDir := filepath.Join(tempDir, "templates")
	assetDir := filepath.Join(tempDir, "assets")
	outputDir := filepath.Join(tempDir, "public")

	for _, dir := range []string{contentDir, templateDir, assetDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	templatePath := filepath.Join(templateDir, "layout.html")
	if err := os.WriteFile(templatePath, []byte("<!DOCTYPE html><html><body>{{.Content}}</body></html>"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "style.css"), []byte("body { color: black; }"), 0600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < n; i++ {
		body := fmt.Sprintf("---\ntitle: Page %d\n---\n# Page %d\nContent %d.\n", i, i, i)
		if err := os.WriteFile(filepath.Join(contentDir, fmt.Sprintf("page%d.md", i)), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.Template = templatePath
	cfg.AssetDir = assetDir
	cfg.OutputDir = outputDir
	cfg.ProjectRoot = tempDir
	cfg.SiteURL = "https://example.com"
	return cfg
}

// BenchmarkBuild_Cold_25 measures a build with no usable cache (cold).
func BenchmarkBuild_Cold_25(b *testing.B) {
	cfg := syntheticSite(b, 25)
	// Ensure the cache starts cold once before the timed loop.
	_ = os.Remove(cachePath(cfg))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = os.Remove(cachePath(cfg))
		// Remove output to force cold path that also checks generatedFiles.
		_ = os.RemoveAll(cfg.OutputDir)
		if _, err := Build(cfg); err != nil {
			b.Fatalf("Build failed: %v", err)
		}
	}
}

// BenchmarkBuild_WarmHit_25 measures a cache-hit rebuild (no inputs changed).
func BenchmarkBuild_WarmHit_25(b *testing.B) {
	cfg := syntheticSite(b, 25)
	if _, err := Build(cfg); err != nil {
		b.Fatalf("prime Build failed: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Build(cfg); err != nil {
			b.Fatalf("Build failed: %v", err)
		}
	}
}

// BenchmarkBuild_WarmSingleTouch_25 measures an incremental rebuild after touching one file.
func BenchmarkBuild_WarmSingleTouch_25(b *testing.B) {
	cfg := syntheticSite(b, 25)
	if _, err := Build(cfg); err != nil {
		b.Fatalf("prime Build failed: %v", err)
	}
	touchPath := filepath.Join(cfg.ContentDir, "page0.md")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Append a deterministic suffix so the content hash changes each iteration
		// but remains hermetic inside b.TempDir().
		f, err := os.OpenFile(touchPath, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			b.Fatalf("open touch file: %v", err)
		}
		_, _ = fmt.Fprintf(f, "\n<!-- touch %d -->\n", i)
		_ = f.Close()
		if _, err := Build(cfg); err != nil {
			b.Fatalf("Build failed: %v", err)
		}
	}
}

// BenchmarkBuild_Cold_300 measures a cold build on a 300-page synthetic site.
func BenchmarkBuild_Cold_300(b *testing.B) {
	cfg := syntheticSite(b, 300)
	_ = os.Remove(cachePath(cfg))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = os.Remove(cachePath(cfg))
		_ = os.RemoveAll(cfg.OutputDir)
		if _, err := Build(cfg); err != nil {
			b.Fatalf("Build failed: %v", err)
		}
	}
}

// BenchmarkBuild_WarmHit_300 measures a cache-hit rebuild on a 300-page site.
func BenchmarkBuild_WarmHit_300(b *testing.B) {
	cfg := syntheticSite(b, 300)
	if _, err := Build(cfg); err != nil {
		b.Fatalf("prime Build failed: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Build(cfg); err != nil {
			b.Fatalf("Build failed: %v", err)
		}
	}
}

// BenchmarkBuild_WarmSingleTouch_300 measures an incremental single-file touch on a 300-page site.
func BenchmarkBuild_WarmSingleTouch_300(b *testing.B) {
	cfg := syntheticSite(b, 300)
	if _, err := Build(cfg); err != nil {
		b.Fatalf("prime Build failed: %v", err)
	}
	touchPath := filepath.Join(cfg.ContentDir, "page0.md")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f, err := os.OpenFile(touchPath, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			b.Fatalf("open touch file: %v", err)
		}
		_, _ = fmt.Fprintf(f, "\n<!-- touch %d -->\n", i)
		_ = f.Close()
		if _, err := Build(cfg); err != nil {
			b.Fatalf("Build failed: %v", err)
		}
	}
}

// TestBuild_BenchColdVsWarm is a deterministic functional check that the cache
// hit path is faster than a cold build. It uses time.Since rather than
// testing.B so it runs in `go test ./...` and is stable in CI.
func TestBuild_BenchColdVsWarm(t *testing.T) {
	for _, n := range []int{25, 100} {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			cfg := syntheticSiteT(t, n)

			// Cold: remove cache + output, time the build.
			_ = os.Remove(cachePath(cfg))
			_ = os.RemoveAll(cfg.OutputDir)
			startCold := time.Now()
			resCold, err := Build(cfg)
			coldDur := time.Since(startCold)
			if err != nil {
				t.Fatalf("cold Build failed: %v", err)
			}
			if resCold.CacheHit {
				t.Fatalf("cold build should be a miss, got hit")
			}

			// Warm hit: no touch, should be a hit and faster.
			startWarm := time.Now()
			resWarm, err := Build(cfg)
			warmDur := time.Since(startWarm)
			if err != nil {
				t.Fatalf("warm Build failed: %v", err)
			}
			if !resWarm.CacheHit {
				t.Fatalf("warm build should be a hit, got miss")
			}
			t.Logf("n=%d cold=%s warmHit=%s ratio=%.2fx", n, coldDur, warmDur, float64(coldDur)/float64(warmDur+time.Nanosecond))

			// Warm hit must be faster than cold — order-of-magnitude, not strict.
			// On very fast 25-page sites the ratio may be small in CI, so only
			// assert that warm is not slower than cold.
			if warmDur > coldDur {
				t.Errorf("warm hit (%s) slower than cold (%s) for n=%d — cache not effective", warmDur, coldDur, n)
			}

			// Single-touch: modify one file, should be a miss.
			touchPath := filepath.Join(cfg.ContentDir, "page0.md")
			f, err := os.OpenFile(touchPath, os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				t.Fatalf("open touch file: %v", err)
			}
			_, _ = fmt.Fprint(f, "\n<!-- bench touch -->\n")
			_ = f.Close()
			resTouch, err := Build(cfg)
			if err != nil {
				t.Fatalf("touch Build failed: %v", err)
			}
			if resTouch.CacheHit {
				t.Fatalf("single-touch build should be a miss, got hit")
			}
		})
	}
}
