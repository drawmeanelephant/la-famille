package sitedata

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/graph"
)

func TestNewManifestSortsPagesAndReferences(t *testing.T) {
	raw := false
	fileMap := map[string]*content.FileMeta{
		"index.md": {
			Title:      "Home",
			Date:       "2026-01-02",
			Tags:       []string{"zeta", "alpha"},
			Categories: []string{"news"},
		},
		"about.md": {
			Title: "About",
			Tags:  []string{"reference"},
		},
		"private.md": {
			Title:  "Private",
			Render: &raw,
		},
	}
	g := graph.Graph{
		Nodes: map[string]graph.Node{
			"index":      {Type: "page", Render: true},
			"about":      {Type: "page", Render: true},
			"private.md": {Type: "page", Render: false},
		},
		Edges: [][2]string{
			{"index", "about"},
			{"index", "about"},
			{"about", "index"},
		},
	}
	links := map[string][]ManifestLink{
		"index.md": {
			{Destination: "about.md", Target: "about.md", GraphTarget: "about", Line: 3, Resolved: true},
			{Destination: "missing.md", Target: "missing.md", GraphTarget: "missing", Line: 4},
		},
	}
	assetReferences := map[string][]string{
		"index.md": {"images/z.svg", "images/a.png", "images/a.png"},
	}
	manifest := NewManifest(
		config.Config{SiteURL: "https://example.com/docs"},
		fileMap,
		g,
		map[string][]string{
			"index": {"about"},
			"about": {"index", "index"},
		},
		map[string]string{
			"index": "index.html",
			"about": "guides/about/index.html",
		},
		links,
		assetReferences,
	)

	if len(manifest.Pages) != 3 {
		t.Fatalf("len(Pages) = %d, want 3", len(manifest.Pages))
	}
	if got, want := []string{manifest.Pages[0].Identity, manifest.Pages[1].Identity, manifest.Pages[2].Identity}, []string{"about", "index", "private.md"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("page identities = %v, want %v", got, want)
	}
	home := manifest.Pages[1]
	if home.URL != "/docs/" || home.Title != "Home" || home.Date != "2026-01-02" {
		t.Fatalf("home metadata = %+v", home)
	}
	if !reflect.DeepEqual(home.Tags, []string{"alpha", "zeta"}) {
		t.Errorf("home tags = %v, want sorted tags", home.Tags)
	}
	if !reflect.DeepEqual(home.OutboundLinks, []string{"about"}) {
		t.Errorf("home outbound links = %v, want deduplicated graph links", home.OutboundLinks)
	}
	if home.InboundLinkCount != 1 {
		t.Errorf("home inbound link count = %d, want 1", home.InboundLinkCount)
	}
	if !reflect.DeepEqual(home.AssetReferences, []string{"images/a.png", "images/z.svg"}) {
		t.Errorf("home asset references = %v, want sorted unique references", home.AssetReferences)
	}
	if len(home.Links) != 2 || home.Links[0].Destination != "about.md" {
		t.Errorf("home links = %+v, want source-order links", home.Links)
	}
	rawPage := manifest.Pages[2]
	if rawPage.Rendered || rawPage.URL != "/docs/private.md" {
		t.Errorf("raw page = %+v, want non-rendered page with raw URL", rawPage)
	}

	dir1, dir2 := t.TempDir(), t.TempDir()
	if err := WriteManifest(dir1, manifest); err != nil {
		t.Fatal(err)
	}
	if err := WriteManifest(dir2, manifest); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(filepath.Join(dir1, ManifestFileName))
	if err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(filepath.Join(dir2, ManifestFileName))
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("identical manifest values did not serialize to identical bytes")
	}
	parsed, err := ReadManifest(filepath.Join(dir1, ManifestFileName))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(parsed, manifest) {
		t.Fatalf("ReadManifest() = %#v, want %#v", parsed, manifest)
	}
}

func TestParseManifestValidatesAndNormalizesJSON(t *testing.T) {
	manifest, err := ParseManifest([]byte(`{"version":1,"pages":[]}`))
	if err != nil {
		t.Fatalf("ParseManifest(valid) error = %v", err)
	}
	if manifest.Pages == nil {
		t.Fatal("ParseManifest(valid) left Pages nil")
	}

	if _, err := ParseManifest([]byte(`{"version":`)); err == nil {
		t.Fatal("ParseManifest(invalid JSON) succeeded")
	}
	if _, err := ParseManifest([]byte(`{"version":99,"pages":[]}`)); err == nil {
		t.Fatal("ParseManifest(unsupported version) succeeded")
	}
}
