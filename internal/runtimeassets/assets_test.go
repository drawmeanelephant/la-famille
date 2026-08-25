package runtimeassets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCuratedLayoutsPacket(t *testing.T) {
	layouts, err := CuratedLayouts()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"layout", "layout-octoburger", "layout-terminal"} {
		data, ok := layouts[name]
		if !ok {
			t.Errorf("curated packet missing %q", name)
			continue
		}
		if len(data) == 0 {
			t.Errorf("curated layout %q is empty", name)
			continue
		}
		if !strings.Contains(string(data), "<!DOCTYPE html>") {
			t.Errorf("curated layout %q does not look like an HTML document", name)
		}
		if strings.Contains(string(data), "{{.") && strings.Count(string(data), "{{") > 0 {
			// Template actions are expected; this guard only fails on obviously
			// truncated files that end mid-action.
			if strings.HasSuffix(strings.TrimSpace(string(data)), "{{") {
				t.Errorf("curated layout %q appears truncated mid-action", name)
			}
		}
	}
}

func TestEmbeddedDefaultsAreAvailable(t *testing.T) {
	template, err := DefaultTemplate()
	if err != nil {
		t.Fatal(err)
	}
	if len(template) == 0 {
		t.Fatal("embedded default template is empty")
	}
	partials, err := DefaultPartials()
	if err != nil {
		t.Fatal(err)
	}
	if len(partials["partials/search_modal.html"]) == 0 {
		t.Fatal("embedded search partial is empty")
	}
	assets, err := DefaultAssetFiles()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"graph/explorer.css", "graph/explorer.js", "css/theme-foundations.css", "css/search.css", "js/search.js", "img/mascot-default.jpeg"} {
		if len(assets[name]) == 0 {
			t.Errorf("embedded asset %q is empty", name)
		}
	}
}

func TestInstallMissingPreservesSiteOverride(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "assets", "graph", "explorer.js")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("site override"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := InstallMissing(root, map[string][]byte{
		"assets/graph/explorer.js": []byte("embedded"),
		"assets/css/search.css":    []byte("fallback"),
	}, 0644); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "site override" {
		t.Errorf("site override changed to %q", got)
	}
	if _, err := os.Stat(filepath.Join(root, "assets", "css", "search.css")); err != nil {
		t.Fatalf("missing fallback asset: %v", err)
	}
}
