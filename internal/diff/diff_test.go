package diff

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/tbuddy/la-famille/internal/sitedata"
)

var updateLedgerGoldens = flag.Bool("update-ledger-goldens", false, "Update reviewed Change Ledger golden files")

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
	if *updateLedgerGoldens {
		if err := os.WriteFile(goldenPath, got, 0600); err != nil {
			t.Fatal(err)
		}
	}
	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s: %v", goldenPath, err)
	}
	if !bytes.Equal(got, golden) {
		t.Fatalf("diff differs from golden %s:\n%s", goldenPath, got)
	}
}

// A shared title alone does not make a rename: when both manifests carry a
// content hash and the bodies differ, the pair is a removal plus an
// unrelated addition and both pages must show up in the report (#654).
func TestCompareSameTitleDifferentContentIsNotRename(t *testing.T) {
	before := sitedata.Manifest{Version: sitedata.ManifestVersion, Pages: []sitedata.ManifestPage{
		{Identity: "x", SourcePath: "x.md", Title: "Same", ContentHash: "aaa"},
	}}
	after := sitedata.Manifest{Version: sitedata.ManifestVersion, Pages: []sitedata.ManifestPage{
		{Identity: "y", SourcePath: "y.md", Title: "Same", ContentHash: "bbb"},
	}}

	report, err := Compare(before, after)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if len(report.RemovedPages) != 1 || report.RemovedPages[0].Identity != "x" {
		t.Errorf("removed pages = %+v, want [x]", report.RemovedPages)
	}
	if len(report.AddedPages) != 1 || report.AddedPages[0].Identity != "y" {
		t.Errorf("added pages = %+v, want [y]", report.AddedPages)
	}
	for _, change := range report.ChangedPages {
		if change.Kind == "renamed" {
			t.Errorf("changed pages = %+v, want no rename pairing", report.ChangedPages)
		}
	}
}

// The title signal still pairs a page whose body is unchanged under a new
// identity — the corroborating evidence a rename needs.
func TestCompareSameTitleIdenticalContentIsRename(t *testing.T) {
	before := sitedata.Manifest{Version: sitedata.ManifestVersion, Pages: []sitedata.ManifestPage{
		{Identity: "x", SourcePath: "x.md", Title: "Same", ContentHash: "aaa"},
	}}
	after := sitedata.Manifest{Version: sitedata.ManifestVersion, Pages: []sitedata.ManifestPage{
		{Identity: "y", SourcePath: "y.md", Title: "Same", ContentHash: "aaa"},
	}}

	report, err := Compare(before, after)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if len(report.AddedPages) != 0 || len(report.RemovedPages) != 0 {
		t.Fatalf("added = %+v, removed = %+v, want a rename instead", report.AddedPages, report.RemovedPages)
	}
	if len(report.ChangedPages) != 1 || report.ChangedPages[0].Kind != "renamed" ||
		report.ChangedPages[0].Before.Identity != "x" || report.ChangedPages[0].After.Identity != "y" {
		t.Errorf("changed pages = %+v, want rename x → y", report.ChangedPages)
	}
}

// Snapshots without a content hash (v1 manifests, partial data) keep the
// legacy title-only pairing: it is the only signal they carry.
func TestCompareSameTitleWithoutHashStillPairs(t *testing.T) {
	before := sitedata.Manifest{Version: 1, Pages: []sitedata.ManifestPage{
		{Identity: "x", SourcePath: "x.md", Title: "Same"},
	}}
	after := sitedata.Manifest{Version: sitedata.ManifestVersion, Pages: []sitedata.ManifestPage{
		{Identity: "y", SourcePath: "y.md", Title: "Same"},
	}}

	report, err := Compare(before, after)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if len(report.ChangedPages) != 1 || report.ChangedPages[0].Kind != "renamed" {
		t.Errorf("changed pages = %+v, want title-only rename pairing", report.ChangedPages)
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
