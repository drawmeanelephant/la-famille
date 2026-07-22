package feed

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
)

func TestWriteDeterministicAndSorted(t *testing.T) {
	out := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.OutputDir = out
	cfg.SiteName = "Example"
	cfg.SiteURL = "https://example.com/docs/"
	err := Write(cfg, []Item{
		{Title: "Older", URL: "https://example.com/docs/older/", Date: "2024-01-01", Description: "old & useful"},
		{Title: "Newer", URL: "https://example.com/docs/newer/", Date: "2024-02-01", Description: "new"},
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(out, "feed.xml"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "<title>Example</title>") || !strings.Contains(s, "<link>https://example.com/docs/</link>") {
		t.Fatalf("missing channel metadata: %s", s)
	}
	if strings.Index(s, "Newer") > strings.Index(s, "Older") {
		t.Fatalf("items are not newest first: %s", s)
	}
	if !strings.Contains(s, "old &amp; useful") || !strings.Contains(s, "<guid>https://example.com/docs/newer/</guid>") {
		t.Fatalf("RSS escaping or GUID missing: %s", s)
	}
}

func TestWriteRemovesFeedWhenEmpty(t *testing.T) {
	out := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.OutputDir = out
	if err := os.WriteFile(filepath.Join(out, "feed.xml"), []byte("stale"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Write(cfg, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "feed.xml")); !os.IsNotExist(err) {
		t.Fatalf("feed.xml still exists, stat error=%v", err)
	}
}

func TestLocalURL(t *testing.T) {
	for _, tc := range []struct{ path, want string }{
		{"index.html", "/"},
		{"about/index.html", "/about/"},
		{"docs/install/index.html", "/docs/install/"},
	} {
		if got := LocalURL(tc.path); got != tc.want {
			t.Errorf("LocalURL(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}
