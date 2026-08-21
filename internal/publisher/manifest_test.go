package publisher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckValidatesCleanURLAndRelativeReferences(t *testing.T) {
	root := t.TempDir()
	writePublishFile(t, root, "index.html", `<a href="/about/">About</a><script src="/assets/site.js"></script><a href="https://example.com">external</a>`)
	writePublishFile(t, root, "about/index.html", `<a href="../">Home</a>`)
	writePublishFile(t, root, "assets/site.js", "console.log('ok')")
	writeRequiredArtifacts(t, root)

	manifest, err := Check(root)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(manifest.Files) != 3+len(coreArtifacts) {
		t.Errorf("manifest files = %v", manifest.Files)
	}
}

func TestCheckRequiresGraphCompanionArtifacts(t *testing.T) {
	root := t.TempDir()
	writePublishFile(t, root, "index.html", "<html></html>")
	writePublishFile(t, root, "graph/index.html", `<link rel="stylesheet" href="../assets/graph/explorer.css"><script src="../assets/graph/explorer.js"></script>`)
	if _, err := Check(root); err == nil || !strings.Contains(err.Error(), "graph/data.json") {
		t.Fatalf("Check error = %v, want missing graph/data.json", err)
	}
}

func TestCheckRejectsCacheInPublic(t *testing.T) {
	root := t.TempDir()
	writePublishFile(t, root, "index.html", "<html></html>")
	writePublishFile(t, root, CacheFileName, "internal")
	if _, err := Check(root); err == nil || !strings.Contains(err.Error(), CacheFileName) {
		t.Fatalf("Check error = %v, want cache policy error", err)
	}
}

func TestCheckReportsMissingLocalReference(t *testing.T) {
	root := t.TempDir()
	writePublishFile(t, root, "index.html", `<a href="missing/">Missing</a>`)
	if _, err := Check(root); err == nil || !strings.Contains(err.Error(), "missing/") {
		t.Fatalf("Check error = %v, want missing-reference error", err)
	}
}

func TestCheckRequiresCoreArtifacts(t *testing.T) {
	root := t.TempDir()
	writePublishFile(t, root, "index.html", "<html></html>")
	_, err := Check(root)
	if err == nil {
		t.Fatal("Check succeeded on a tree without sitemap/robots/search/graph/backlinks/meta")
	}
	for _, required := range coreArtifacts {
		if !strings.Contains(err.Error(), required) {
			t.Errorf("Check error does not mention missing required file %q:\n%v", required, err)
		}
	}
}

func TestCheckFeedRequirementFollowsMetaData(t *testing.T) {
	t.Run("dated rendered page requires feed.xml", func(t *testing.T) {
		root := t.TempDir()
		writePublishFile(t, root, "index.html", "<html></html>")
		writeRequiredArtifacts(t, root)
		writePublishFile(t, root, "meta.json", `{"posts/hello":{"title":"Hello","render":true,"date":"2026-07-15"}}`)

		if _, err := Check(root); err == nil || !strings.Contains(err.Error(), "feed.xml") {
			t.Fatalf("Check error = %v, want missing feed.xml", err)
		}

		writePublishFile(t, root, "feed.xml", `<?xml version="1.0"?><rss/>`)
		if _, err := Check(root); err != nil {
			t.Fatalf("Check with feed.xml present: %v", err)
		}
	})

	t.Run("undated pages do not require feed.xml", func(t *testing.T) {
		root := t.TempDir()
		writePublishFile(t, root, "index.html", "<html></html>")
		writeRequiredArtifacts(t, root)
		writePublishFile(t, root, "meta.json", `{"index":{"title":"Home","render":true,"date":""}}`)

		if _, err := Check(root); err != nil {
			t.Fatalf("Check error = %v, want undated artifact to pass without feed.xml", err)
		}
	})

	t.Run("unrendered dated pages do not require feed.xml", func(t *testing.T) {
		root := t.TempDir()
		writePublishFile(t, root, "index.html", "<html></html>")
		writeRequiredArtifacts(t, root)
		writePublishFile(t, root, "meta.json", `{"notes/raw.md":{"title":"Raw","render":false,"date":"2026-07-15"}}`)

		if _, err := Check(root); err != nil {
			t.Fatalf("Check error = %v, want render:false dated page to skip feed.xml", err)
		}
	})
}

func TestCheckRejectsStagingDirectories(t *testing.T) {
	root := t.TempDir()
	writePublishFile(t, root, "index.html", "<html></html>")
	writeRequiredArtifacts(t, root)
	writePublishFile(t, root, ".staging-build-123/partial/index.html", "<html></html>")

	_, err := Check(root)
	if err == nil || !strings.Contains(err.Error(), stagingDirPrefix) {
		t.Fatalf("Check error = %v, want staging directory rejection", err)
	}
}

// writeRequiredArtifacts writes the always-required metadata files so tests
// can exercise other validation on an otherwise complete artifact.
func writeRequiredArtifacts(t *testing.T, root string) {
	t.Helper()
	for _, required := range coreArtifacts {
		content := "{}"
		if required == "robots.txt" {
			content = "User-agent: *\nAllow: /\n"
		} else if required == "sitemap.xml" {
			content = `<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"/>`
		}
		writePublishFile(t, root, required, content)
	}
}

func writePublishFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}
