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

func TestCheckResolvesHtmlToIndexFallback(t *testing.T) {
	t.Run("absolute html maps to directory index", func(t *testing.T) {
		root := t.TempDir()
		writePublishFile(t, root, "index.html", `<a href="/docs/setup.html">setup</a>`)
		writePublishFile(t, root, "docs/setup/index.html", "<html>setup</html>")
		writeRequiredArtifacts(t, root)
		if _, err := Check(root); err != nil {
			t.Fatalf("Check = %v, want /docs/setup.html to resolve to docs/setup/index.html", err)
		}
	})

	t.Run("relative html maps to directory index", func(t *testing.T) {
		root := t.TempDir()
		writePublishFile(t, root, "docs/index.html", `<a href="setup.html">setup</a>`)
		writePublishFile(t, root, "docs/setup/index.html", "<html>setup</html>")
		writeRequiredArtifacts(t, root)
		writePublishFile(t, root, "index.html", "<html>home</html>")
		if _, err := Check(root); err != nil {
			t.Fatalf("Check = %v, want relative setup.html to resolve to docs/setup/index.html", err)
		}
	})

	t.Run("html with query and fragment", func(t *testing.T) {
		root := t.TempDir()
		writePublishFile(t, root, "index.html", `<a href="/docs/setup.html?x=1#anchor">setup</a>`)
		writePublishFile(t, root, "docs/setup/index.html", "<html>setup</html>")
		writeRequiredArtifacts(t, root)
		if _, err := Check(root); err != nil {
			t.Fatalf("Check = %v, want query/fragment html to resolve", err)
		}
	})

	t.Run("direct html still preferred over index", func(t *testing.T) {
		root := t.TempDir()
		writePublishFile(t, root, "index.html", `<a href="/docs/setup.html">setup</a>`)
		writePublishFile(t, root, "docs/setup.html", "<html>direct</html>")
		writePublishFile(t, root, "docs/setup/index.html", "<html>index</html>")
		writeRequiredArtifacts(t, root)
		if _, err := Check(root); err != nil {
			t.Fatalf("Check = %v, want direct html to pass", err)
		}
	})

	t.Run("missing html still fails", func(t *testing.T) {
		root := t.TempDir()
		writePublishFile(t, root, "index.html", `<a href="/missing.html">missing</a>`)
		writeRequiredArtifacts(t, root)
		if _, err := Check(root); err == nil || !strings.Contains(err.Error(), "missing.html") {
			t.Fatalf("Check error = %v, want missing html failure", err)
		}
	})
}

func TestResolveReferenceHtmlFallback(t *testing.T) {
	files := map[string]struct{}{
		"docs/setup/index.html": {},
		"index.html":            {},
	}
	if _, ok := resolveReference("index.html", "/docs/setup.html", files); !ok {
		t.Fatalf("resolveReference /docs/setup.html should resolve to docs/setup/index.html")
	}
	if _, ok := resolveReference("docs/index.html", "setup.html", files); !ok {
		t.Fatalf("resolveReference relative setup.html should resolve")
	}
	if _, ok := resolveReference("index.html", "/docs/setup.html?x=1#frag", files); !ok {
		t.Fatalf("resolveReference with query/fragment should resolve")
	}
	// Direct file should still win
	files["docs/setup.html"] = struct{}{}
	if target, ok := resolveReference("index.html", "/docs/setup.html", files); !ok || target != "docs/setup.html" {
		t.Fatalf("direct file should be preferred, got %q ok=%v", target, ok)
	}
}

// writeRequiredArtifacts writes the always-required metadata files so tests
// can exercise other validation on an otherwise complete artifact.
func writeRequiredArtifacts(t *testing.T, root string) {
	t.Helper()
	for _, required := range coreArtifacts {
		var content string
		switch required {
		case "robots.txt":
			content = "User-agent: *\nAllow: /\n"
		case "sitemap.xml":
			content = `<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"/>`
		default:
			content = "{}"
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
