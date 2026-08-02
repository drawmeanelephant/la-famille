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

	manifest, err := Check(root)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(manifest.Files) != 3 {
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
