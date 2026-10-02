package pack

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/generator"
	"github.com/tbuddy/la-famille/internal/ragexport"
)

func TestCanonicalBuilderDeltaFixtureGolden(t *testing.T) {
	fixture, err := filepath.Abs("../../assets/testdata/sites/artisanal-ceramics")
	if err != nil {
		t.Fatal(err)
	}
	template, err := filepath.Abs("../../templates/layout.html")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	for _, dir := range []string{"content", "assets"} {
		if err := os.CopyFS(filepath.Join(root, dir), os.DirFS(filepath.Join(fixture, dir))); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.DefaultConfig()
	cfg.ProjectRoot = root
	cfg.ContentDir = filepath.Join(root, "content")
	cfg.AssetDir = filepath.Join(root, "assets")
	cfg.Template = template
	cfg.OutputDir = filepath.Join(root, "public")
	cfg.RagDir = filepath.Join(root, "rag")
	cfg.SiteName = "Kintsugi & Co. Studio"
	cfg.SiteURL = "https://kintsugi.example.com"
	options := BuildOptions{
		OutputDir: cfg.OutputDir, RagDir: cfg.RagDir,
		Site:       Site{Name: cfg.SiteName, URL: cfg.SiteURL},
		Provenance: Provenance{Generator: "la-famille", Version: "dev"},
	}
	build := func(name string) {
		t.Helper()
		if _, err := generator.Build(cfg); err != nil {
			t.Fatal(err)
		}
		if err := ragexport.RunExport(cfg); err != nil {
			t.Fatal(err)
		}
		if _, err := Build(options, filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
	build("before.tar")
	page := filepath.Join(cfg.ContentDir, "care-guide.md")
	edited := bytes.Replace(readTestFile(t, page), []byte("  - care\n"), []byte("  - preservation\n"), 1)
	edited = append(edited, []byte("\n## Local pack update\n\nDry each vessel fully before storage. See the [studio overview](index.md).\n")...)
	writeInput(t, page, edited)
	build("after.tar")
	before, after, delta := filepath.Join(root, "before.tar"), filepath.Join(root, "after.tar"), filepath.Join(root, "delta.tar")
	baseBytes := readTestFile(t, before)
	report, err := Diff(before, after, delta)
	if err != nil {
		t.Fatal(err)
	}
	if report.Ledger == nil || len(report.Ledger.AddedEdges) != 1 || len(report.Ledger.TaxonomyChanges) != 2 {
		t.Fatalf("fixture did not reuse Ledger graph/taxonomy comparisons: %+v", report)
	}
	result := filepath.Join(root, "result.tar")
	if _, err := Apply(before, delta, result); err != nil {
		t.Fatal(err)
	}
	targetBytes := readTestFile(t, after)
	if !bytes.Equal(readTestFile(t, result), targetBytes) {
		t.Fatal("canonical v1 builder apply result is not byte-identical to the target full pack")
	}
	if !bytes.Equal(readTestFile(t, before), baseBytes) {
		t.Fatal("fixture base modified")
	}
	if _, err := VerifyFile(result); err != nil {
		t.Fatal(err)
	}
	repeated := filepath.Join(root, "repeated.tar")
	if _, err := Diff(before, after, repeated); err != nil {
		t.Fatal(err)
	}
	deltaBytes := readTestFile(t, delta)
	if !bytes.Equal(deltaBytes, readTestFile(t, repeated)) {
		t.Fatal("fixture deltas not deterministic")
	}
	// The golden captures the member boundary, not toolchain-specific hashes.
	golden := struct {
		Added           []string `json:"added"`
		Removed         []string `json:"removed"`
		Changed         []string `json:"changed"`
		DeltaPayloads   int      `json:"delta_payloads"`
		AddedGraphEdges int      `json:"added_graph_edges"`
		TaxonomyChanges int      `json:"taxonomy_changes"`
		TargetIdentical bool     `json:"target_byte_identical"`
		DeltaIdentical  bool     `json:"repeated_delta_byte_identical"`
	}{
		report.Added, report.Removed, report.Changed,
		len(archiveEntries(t, deltaBytes)) - 2,
		len(report.Ledger.AddedEdges), len(report.Ledger.TaxonomyChanges), true, true,
	}
	actual, err := json.MarshalIndent(golden, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	actual = append(actual, '\n')
	want := readTestFile(t, "testdata/artisanal-ceramics-delta.json")
	if !bytes.Equal(actual, want) {
		t.Fatalf("fixture golden differs:\n%s", actual)
	}
}
