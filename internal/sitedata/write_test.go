package sitedata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWrite(t *testing.T) {
	tempDir := t.TempDir()

	metaData := map[string]map[string]interface{}{
		"index": {
			"title": "Home Page",
		},
		"about/me": {
			"title": "About Me",
			"slug": "jules",
		},
	}

	err := Write(tempDir, metaData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// 1. Check meta.json
	metaContent, err := os.ReadFile(filepath.Join(tempDir, "meta.json"))
	if err != nil {
		t.Fatalf("Failed to read meta.json: %v", err)
	}
	var readMeta map[string]map[string]interface{}
	if err := json.Unmarshal(metaContent, &readMeta); err != nil {
		t.Fatalf("Failed to parse meta.json: %v", err)
	}

	if readMeta["index"]["title"] != "Home Page" {
		t.Errorf("Unexpected meta content: %+v", readMeta)
	}

	// 2. Check sitemap.xml
	sitemapContentBytes, err := os.ReadFile(filepath.Join(tempDir, "sitemap.xml"))
	if err != nil {
		t.Fatalf("Failed to read sitemap.xml: %v", err)
	}
	sitemapContent := string(sitemapContentBytes)

	if !strings.Contains(sitemapContent, "<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">") {
		t.Errorf("sitemap.xml missing root urlset tag")
	}

	if !strings.Contains(sitemapContent, "<loc>/index.html</loc>") {
		t.Errorf("sitemap.xml missing loc for index.html")
	}

	if !strings.Contains(sitemapContent, "<loc>/about/jules/index.html</loc>") {
		t.Errorf("sitemap.xml missing loc for about/jules/index.html")
	}
}
