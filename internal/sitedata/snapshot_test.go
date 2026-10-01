package sitedata

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSnapshotUsesPublishedBytesAndActualSitemap(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"index.html": "rendered", "raw.md": "raw", "asset.png": "image",
		"sitemap.xml": `<urlset><url><loc>/actual/</loc></url><url><loc>/</loc></url><url><loc>/actual/</loc></url></urlset>`,
		"diff.json":   "must not recurse", "diff.txt": "must not recurse", ManifestFileName: "old",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	m := Manifest{Version: 2, Pages: []ManifestPage{
		{Identity: "index", Rendered: true}, {Identity: "raw.md", SourcePath: "raw.md"},
	}}
	if err := SnapshotOutput(&m, dir, map[string]string{"index": "index.html"}); err != nil {
		t.Fatal(err)
	}
	if !m.OutputCaptured || m.Pages[0].OutputHash == "" || m.Pages[1].OutputHash == "" ||
		!reflect.DeepEqual(m.Sitemap, []string{"/", "/actual/"}) {
		t.Fatalf("snapshot = %+v", m)
	}
	if len(m.Files) != 2 || m.Files[0].Path != "asset.png" || m.Files[1].Path != "sitemap.xml" {
		t.Fatalf("files = %+v", m.Files)
	}
	first := m.Pages[0].OutputHash
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := SnapshotOutput(&m, dir, map[string]string{"index": "index.html"}); err != nil {
		t.Fatal(err)
	}
	if m.Pages[0].OutputHash == first {
		t.Fatal("published-byte change missed")
	}
}

func TestSnapshotMissingPageAndMalformedSitemapFail(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "sitemap.xml"), []byte("<urlset/>"), 0600); err != nil {
		t.Fatal(err)
	}
	m := Manifest{Version: 2, Pages: []ManifestPage{{Identity: "index", Rendered: true}}}
	if SnapshotOutput(&m, dir, map[string]string{"index": "index.html"}) == nil {
		t.Fatal("missing published page accepted")
	}
	if err := os.WriteFile(filepath.Join(dir, "sitemap.xml"), []byte("<bad"), 0600); err != nil {
		t.Fatal(err)
	}
	if SnapshotOutput(&Manifest{}, dir, nil) == nil {
		t.Fatal("malformed sitemap accepted")
	}
}
