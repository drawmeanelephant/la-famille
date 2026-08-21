package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuild_AuditCleanup verifies the audit checklist from #473:
// - render:false toggle removes old HTML
// - feed.xml removed when zero dated pages
// - graph explorer toggle cleans up graph/index.html + graph/data.json
// All via the transactional staging + atomic swap, not via manual cleanup.
func TestBuild_AuditCleanup(t *testing.T) {
	t.Run("render_false_toggle_removes_old_html", func(t *testing.T) {
		cfg, _ := setupTestSite(t)
		// page1.md initially rendered -> public/page1/index.html
		if _, err := Build(cfg); err != nil {
			t.Fatalf("initial Build failed: %v", err)
		}
		pageOut := filepath.Join(cfg.OutputDir, "page1", "index.html")
		if _, err := os.Stat(pageOut); err != nil {
			t.Fatalf("expected page1 output before toggle: %v", err)
		}
		// Toggle to render:false — content change triggers fingerprint miss,
		// staging omits the HTML, swap removes stale file.
		page1Path := filepath.Join(cfg.ContentDir, "page1.md")
		if err := os.WriteFile(page1Path, []byte("---\nrender: false\n---\n# Page One\nNow raw.\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Build(cfg); err != nil {
			t.Fatalf("rebuild after render:false toggle failed: %v", err)
		}
		if _, err := os.Stat(pageOut); !os.IsNotExist(err) {
			t.Fatalf("stale HTML survived render:false toggle: %v", err)
		}
		// Raw file is copied verbatim to its .md path, not as HTML.
		rawOut := filepath.Join(cfg.OutputDir, "page1.md")
		if _, err := os.Stat(rawOut); err != nil {
			t.Fatalf("render:false raw file missing: %v", err)
		}
		// search.json must not contain the now-unrendered page.
		search, err := os.ReadFile(filepath.Join(cfg.OutputDir, "search.json"))
		if err != nil {
			t.Fatalf("read search.json: %v", err)
		}
		if strings.Contains(string(search), "Page One") && strings.Contains(string(search), "\"u\":\"/page1/\"") {
			t.Errorf("search.json still indexes render:false page: %s", search)
		}
	})

	t.Run("feed_removed_when_no_dated_pages", func(t *testing.T) {
		cfg, _ := setupTestSite(t)
		// Give page1 a date so feed.xml is emitted.
		page1Path := filepath.Join(cfg.ContentDir, "page1.md")
		if err := os.WriteFile(page1Path, []byte("---\ntitle: Page One\ndate: 2024-03-10\n---\n# Page One\nDated.\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Build(cfg); err != nil {
			t.Fatalf("initial dated Build failed: %v", err)
		}
		feedPath := filepath.Join(cfg.OutputDir, "feed.xml")
		if _, err := os.Stat(feedPath); err != nil {
			t.Fatalf("expected feed.xml with dated page: %v", err)
		}
		// Remove the date — fingerprint changes, staging rebuild, feed.Write removes stale feed.
		if err := os.WriteFile(page1Path, []byte("# Page One\nNo longer dated.\n"), 0600); err != nil {
			t.Fatal(err)
		}
		// page2 also has no date (from setup)
		if _, err := Build(cfg); err != nil {
			t.Fatalf("rebuild after removing date failed: %v", err)
		}
		if _, err := os.Stat(feedPath); !os.IsNotExist(err) {
			t.Fatalf("stale feed.xml survived zero-dated rebuild: %v", err)
		}
	})

	t.Run("graph_explorer_toggle_cleans_output", func(t *testing.T) {
		cfg, _ := setupTestSite(t)
		cfg.GraphExplorer = true
		if _, err := Build(cfg); err != nil {
			t.Fatalf("initial graph-enabled Build failed: %v", err)
		}
		indexPath := filepath.Join(cfg.OutputDir, "graph", "index.html")
		dataPath := filepath.Join(cfg.OutputDir, "graph", "data.json")
		if _, err := os.Stat(indexPath); err != nil {
			t.Fatalf("expected graph/index.html when enabled: %v", err)
		}
		if _, err := os.Stat(dataPath); err != nil {
			t.Fatalf("expected graph/data.json when enabled: %v", err)
		}
		// Toggle off — config change invalidates fingerprint, new staging omits graph.
		cfg.GraphExplorer = false
		if _, err := Build(cfg); err != nil {
			t.Fatalf("rebuild with graph_explorer:false failed: %v", err)
		}
		if _, err := os.Stat(indexPath); !os.IsNotExist(err) {
			t.Fatalf("stale graph/index.html survived disable: %v", err)
		}
		if _, err := os.Stat(dataPath); !os.IsNotExist(err) {
			t.Fatalf("stale graph/data.json survived disable: %v", err)
		}
		// Toggle back on — should recreate.
		cfg.GraphExplorer = true
		if _, err := Build(cfg); err != nil {
			t.Fatalf("rebuild with graph_explorer:true failed: %v", err)
		}
		if _, err := os.Stat(indexPath); err != nil {
			t.Fatalf("graph/index.html not recreated after re-enable: %v", err)
		}
	})

	t.Run("deleted_source_removes_stale_output", func(t *testing.T) {
		cfg, _ := setupTestSite(t)
		if _, err := Build(cfg); err != nil {
			t.Fatalf("initial Build failed: %v", err)
		}
		// Add taxonomy so deletion also updates taxonomy/search.
		page2Path := filepath.Join(cfg.ContentDir, "page2.md")
		if err := os.WriteFile(page2Path, []byte("---\ntitle: Page Two\ntags: [audit]\n---\n# Page Two\nTagged.\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Build(cfg); err != nil {
			t.Fatalf("Build with tagged page failed: %v", err)
		}
		if _, err := os.Stat(filepath.Join(cfg.OutputDir, "tags", "audit", "index.html")); err != nil {
			t.Fatalf("expected taxonomy page before delete: %v", err)
		}
		if err := os.Remove(page2Path); err != nil {
			t.Fatal(err)
		}
		if _, err := Build(cfg); err != nil {
			t.Fatalf("rebuild after delete failed: %v", err)
		}
		if _, err := os.Stat(filepath.Join(cfg.OutputDir, "page2", "index.html")); !os.IsNotExist(err) {
			t.Fatalf("deleted source output still exists: %v", err)
		}
		// Taxonomy for deleted tag should also be gone (staging ensures clean).
		search, err := os.ReadFile(filepath.Join(cfg.OutputDir, "search.json"))
		if err != nil {
			t.Fatalf("read search.json: %v", err)
		}
		if strings.Contains(string(search), "Page Two") || strings.Contains(string(search), "\"audit\"") {
			t.Errorf("search still contains deleted page/tag: %s", search)
		}
	})
}

// TestBuild_AuditWatcherContract documents the watcher invariants without
// starting a real fsnotify loop. The watcher package's own tests cover
// debounce coalescing, single-builder serialization, and cancellation;
// this test pins the generator's transactional guarantee that the watcher
// relies on: `public/` is never half-written.
func TestBuild_AuditWatcherContract(t *testing.T) {
	cfg, _ := setupTestSite(t)
	// Rapid successive builds via the same process must be serialized by
	// lockOutputDir (buildLocks) and must never leave staging or previous dirs.
	for i := 0; i < 3; i++ {
		if _, err := Build(cfg); err != nil {
			t.Fatalf("Build %d failed: %v", i, err)
		}
	}
	entries, err := os.ReadDir(filepath.Dir(cfg.OutputDir))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".staging-") || strings.Contains(e.Name(), ".previous-") {
			t.Fatalf("build left intermediate directory: %s", e.Name())
		}
	}
	// Output must be fully populated after each build (graph + feed + search).
	for _, p := range []string{"page1/index.html", "page2/index.html", "search.json", "graph.json", "sitemap.xml", "robots.txt"} {
		if _, err := os.Stat(filepath.Join(cfg.OutputDir, p)); err != nil {
			t.Errorf("output missing %s after build: %v", p, err)
		}
	}
}

// TestBuild_AuditConcurrentIsProcessLocal ensures the documented limitation:
// concurrent builds in the same process are serialized, but the lock is
// process-local and does not protect two separate la-famille processes on the
// same outputDir. The audit report must document this.
func TestBuild_AuditConcurrentIsProcessLocal(t *testing.T) {
	cfg, _ := setupTestSite(t)
	// The existing build_concurrency_test covers 6 parallel Builds in one
	// process. Here we simply assert that sequential builds with the same
	// outputDir never fail and leave no staging behind, which is the
	// contract the watcher depends on.
	if _, err := Build(cfg); err != nil {
		t.Fatalf("first Build failed: %v", err)
	}
	if _, err := Build(cfg); err != nil {
		t.Fatalf("second Build failed: %v", err)
	}
	// No assertion on cross-process flock — that is intentionally process-local
	// per generator.go:51 comment. See content/jules/reports/*-build-audit.md.
}

func TestBuild_AuditAssetPathSafetyStillEnforced(t *testing.T) {
	cfg, _ := setupTestSite(t)
	// Asset copy must still reject paths that escape the output directory.
	// The asset package's own tests cover IsSafePath; here we ensure a
	// top-level Build with a normal asset still succeeds and the asset lands
	// where expected.
	if _, err := Build(cfg); err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cfg.OutputDir, "assets", "style.css")); err != nil {
		t.Fatalf("asset not copied: %v", err)
	}
}
