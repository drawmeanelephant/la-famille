package diff

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/tbuddy/la-famille/internal/sitedata"
)

func TestCompareGoldenManifestDiff(t *testing.T) {
	before := readFixtureManifest(t, "rename-before.json")
	after := readFixtureManifest(t, "rename-after.json")
	report, err := Compare(before, after)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if len(report.ChangedPages) != 3 {
		t.Fatalf("changed pages = %+v, want renamed page and two linked pages", report.ChangedPages)
	}
	var renamed bool
	for _, change := range report.ChangedPages {
		if change.Kind == "renamed" && change.Before.Identity == "old" && change.After.Identity == "new" {
			renamed = true
		}
	}
	if !renamed {
		t.Errorf("changed pages do not identify old → new rename: %+v", report.ChangedPages)
	}
	if len(report.AddedLinks) != 2 ||
		report.AddedLinks[0] != (LinkChange{Page: "index", Destination: "new.md", Target: "new.md"}) ||
		report.AddedLinks[1] != (LinkChange{Page: "target", Destination: "missing.md", Target: "missing.md"}) {
		t.Errorf("added links = %+v, want index → new.md and target → missing.md", report.AddedLinks)
	}
	wantAddedEdges := []GraphEdge{
		{From: "index", To: "new"},
		{From: "new", To: "target"},
		{From: "target", To: "missing"},
	}
	if len(report.AddedEdges) != len(wantAddedEdges) {
		t.Fatalf("added edges = %+v, want %+v", report.AddedEdges, wantAddedEdges)
	}
	for i, want := range wantAddedEdges {
		if report.AddedEdges[i] != want {
			t.Errorf("added edge[%d] = %+v, want %+v", i, report.AddedEdges[i], want)
		}
	}
	wantRemovedEdges := []GraphEdge{
		{From: "index", To: "old"},
		{From: "old", To: "target"},
	}
	if len(report.RemovedEdges) != len(wantRemovedEdges) {
		t.Fatalf("removed edges = %+v, want %+v", report.RemovedEdges, wantRemovedEdges)
	}
	for i, want := range wantRemovedEdges {
		if report.RemovedEdges[i] != want {
			t.Errorf("removed edge[%d] = %+v, want %+v", i, report.RemovedEdges[i], want)
		}
	}
	if len(report.NewBrokenLinks) != 1 ||
		report.NewBrokenLinks[0].Page != "target" ||
		report.NewBrokenLinks[0].Destination != "missing.md" {
		t.Errorf("new broken links = %+v, want target → missing.md", report.NewBrokenLinks)
	}
	if len(report.OrphanChanges) != 0 {
		t.Errorf("orphan changes = %+v, want unchanged pre-existing orphan omitted", report.OrphanChanges)
	}

	got, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal diff: %v", err)
	}
	got = append(got, '\n')
	goldenPath := filepath.Join("testdata", "rename-golden.json")
	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s: %v", goldenPath, err)
	}
	if !bytes.Equal(got, golden) {
		t.Fatalf("diff differs from golden %s:\n%s", goldenPath, got)
	}
}

func TestCompareRejectsDuplicatePageIdentity(t *testing.T) {
	manifest := sitedata.Manifest{
		Version: sitedata.ManifestVersion,
		Pages: []sitedata.ManifestPage{
			{Identity: "same"},
			{Identity: "same"},
		},
	}
	if _, err := Compare(manifest, sitedata.Manifest{Version: sitedata.ManifestVersion}); err == nil {
		t.Fatal("Compare() accepted duplicate page identities")
	}
}

func TestReportSummaryIncludesChangedCategories(t *testing.T) {
	report := Report{
		AddedPages:   []PageRef{{Identity: "added", Title: "Added"}},
		RemovedPages: []PageRef{},
		ChangedPages: []PageChange{},
		MetadataChanges: []MetadataChange{
			{Page: "page", Field: "title", Before: "Old", After: "New"},
		},
		TaxonomyChanges: []TaxonomyChange{
			{Page: "page", Kind: "tag", Term: "notes", Action: "added"},
		},
		AddedLinks:          []LinkChange{{Page: "page", Destination: "added.md", Target: "added.md"}},
		RemovedLinks:        []LinkChange{},
		AddedEdges:          []GraphEdge{{From: "page", To: "added"}},
		RemovedEdges:        []GraphEdge{},
		OrphanChanges:       []OrphanChange{{Page: "added", State: "orphaned"}},
		NewBrokenLinks:      []BrokenLink{{Page: "page", Destination: "missing.md", Target: "missing.md", Line: 3}},
		ResolvedBrokenLinks: []BrokenLink{},
	}
	summary := report.Summary("before", "after")
	for _, want := range []string{
		"Pages: 1 added, 0 removed, 0 changed",
		"Metadata: 1 field changes",
		"Taxonomy: 1 membership changes",
		"Links: 1 added, 0 removed",
		"Graph edges: 1 added, 0 removed",
		"Orphans: 1 newly orphaned, 0 no longer orphaned",
		"Broken links: 1 newly broken, 0 resolved",
	} {
		if !bytes.Contains([]byte(summary), []byte(want)) {
			t.Errorf("summary missing %q:\n%s", want, summary)
		}
	}
}

func readFixtureManifest(t *testing.T, name string) sitedata.Manifest {
	t.Helper()
	manifest, err := sitedata.ReadManifest(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture manifest %s: %v", name, err)
	}
	return manifest
}
