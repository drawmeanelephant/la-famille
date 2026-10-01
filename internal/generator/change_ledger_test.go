package generator

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/checker"
	"github.com/tbuddy/la-famille/internal/config"
	sitediff "github.com/tbuddy/la-famille/internal/diff"
	"github.com/tbuddy/la-famille/internal/sitedata"
)

func TestChangeLedgerRealFixtureGoldens(t *testing.T) {
	for _, fixture := range []string{"anchor-links", "artisanal-ceramics"} {
		t.Run(fixture, func(t *testing.T) {
			cfg := ledgerFixture(t, fixture)
			putLedgerFile(t, filepath.Join(cfg.ContentDir, "ledger-orphan.md"), "---\ntitle: Existing orphan\n---\nUnrelated debt.\n")
			putLedgerFile(t, filepath.Join(cfg.ContentDir, "ledger-target.md"), "---\ntitle: Ledger target\n---\nTarget.\n")
			first, err := Build(cfg)
			if err != nil {
				t.Fatal(err)
			}
			if first.Ledger == nil || !first.Ledger.Baseline || !first.Ledger.Changes.Empty() {
				t.Fatalf("first build = %+v", first.Ledger)
			}

			path, id, oldEdge := "other.md", "other", "index"
			if fixture == "artisanal-ceramics" {
				path, id, oldEdge = "care-guide.md", "care-guide", "collection/wheel-thrown-vessels"
			}
			sourcePath := filepath.Join(cfg.ContentDir, path)
			data, err := os.ReadFile(sourcePath)
			if err != nil {
				t.Fatal(err)
			}
			body := string(data)
			if fixture == "anchor-links" {
				body = "---\ntitle: Other revised\n---\n" + strings.Replace(body, "[Jump back](index.md#section)", "", 1)
			} else {
				body = strings.Replace(body, `title: "Ceramic Ware Care & Preservation Guide"`, `title: "Care guide revised"`, 1)
				body = strings.Replace(body, "[Wheel-Thrown Collection](collection/wheel-thrown-vessels.md)", "collection", 1)
			}
			putLedgerFile(t, sourcePath, body+"\n[Ledger target](ledger-target.md)\n")
			second, err := Build(cfg)
			if err != nil {
				t.Fatal(err)
			}
			r := second.Ledger.Changes
			if len(r.ChangedPages) != 1 || r.ChangedPages[0].After.Identity != id {
				t.Fatalf("source pages changed = %+v, want only %s", r.ChangedPages, id)
			}
			if len(r.AddedLinks) != 1 || r.AddedLinks[0].Page != id || r.AddedLinks[0].Destination != "ledger-target.md" {
				t.Fatalf("added links = %+v", r.AddedLinks)
			}
			if !reflect.DeepEqual(r.AddedEdges, []sitediff.GraphEdge{{From: id, To: "ledger-target"}}) ||
				!reflect.DeepEqual(r.RemovedEdges, []sitediff.GraphEdge{{From: id, To: oldEdge}}) {
				t.Fatalf("edge deltas = %+v / %+v", r.AddedEdges, r.RemovedEdges)
			}
			for _, regression := range r.Regressions {
				if regression.Page == "ledger-orphan" {
					t.Fatal("existing orphan regressed")
				}
			}
			assertLedgerGolden(t, fixture+"-diff.json", r)

			putLedgerFile(t, filepath.Join(cfg.ContentDir, "new-page.md"), "---\ntitle: New page\n---\n[Broken](nonexistent.md)\n")
			third, err := Build(cfg)
			if err != nil {
				t.Fatal(err)
			}
			r = third.Ledger.Changes
			if len(r.NewBrokenLinks) != 1 || r.NewBrokenLinks[0].Page != "new-page" ||
				len(r.OrphanChanges) != 1 || r.OrphanChanges[0] != (sitediff.OrphanChange{Page: "new-page", State: "orphaned"}) {
				t.Fatalf("new debt = %+v / %+v", r.NewBrokenLinks, r.OrphanChanges)
			}
			assertLedgerGolden(t, fixture+"-broken-diff.json", r)
			live, err := checker.Validate(cfg)
			if err != nil {
				t.Fatal(err)
			}
			snapshot, err := checker.ValidateWithManifest(cfg, filepath.Join(cfg.OutputDir, sitedata.ManifestFileName))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(linkFindingKeys(live), linkFindingKeys(snapshot)) {
				t.Fatal("manifest and checker disagree after mutations")
			}
		})
	}
}

func TestLedgerProseAssetsFrontmatterAndFailedBuild(t *testing.T) {
	cfg := ledgerFixture(t, "anchor-links")
	first, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	putLedgerFile(t, filepath.Join(cfg.ContentDir, "index.md"), "# Index\n[Other](other.md)\nReworded prose.\n")
	prose, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if prose.Ledger.Baseline || len(prose.Ledger.Changes.ChangedPages) != 1 || prose.Ledger.Changes.GateError() != nil {
		t.Fatalf("prose comparison = %+v", prose.Ledger)
	}
	cached, err := Build(cfg)
	if err != nil || !cached.CacheHit || !reflect.DeepEqual(cached.Ledger, prose.Ledger) {
		t.Fatalf("cached ledger = %+v, %v", cached.Ledger, err)
	}
	if first.Ledger == nil {
		t.Fatal("missing baseline")
	}
	putLedgerFile(t, filepath.Join(cfg.AssetDir, "sample.txt"), "asset one")
	if _, err := Build(cfg); err != nil {
		t.Fatal(err)
	}
	putLedgerFile(t, filepath.Join(cfg.AssetDir, "sample.txt"), "asset two")
	asset, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, file := range asset.Ledger.Changes.FileChanges {
		found = found || file.Path == "assets/sample.txt" && file.Action == "modified"
	}
	if !found || asset.Ledger.Changes.GateError() != nil {
		t.Fatalf("asset drift = %+v", asset.Ledger.Changes)
	}
	putLedgerFile(t, filepath.Join(cfg.ContentDir, "other.md"), "---\nauthor: Revised author\ndescription: Revised description\n---\n# Other\n")
	meta, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	found = false
	for _, change := range meta.Ledger.Changes.MetadataChanges {
		found = found || change.Field == "author" && change.After == "Revised author"
	}
	if !found {
		t.Fatalf("author change missing: %+v", meta.Ledger.Changes)
	}
	before, err := os.ReadFile(filepath.Join(cfg.OutputDir, sitediff.JSONFileName))
	if err != nil {
		t.Fatal(err)
	}
	putLedgerFile(t, cfg.Template, "{{broken")
	if _, err := Build(cfg); err == nil {
		t.Fatal("invalid template unexpectedly built")
	}
	after, err := os.ReadFile(filepath.Join(cfg.OutputDir, sitediff.JSONFileName))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("failed build replaced successful ledger")
	}
	putLedgerFile(t, cfg.Template, "<html><body>{{.Content}}</body></html>")
	repaired, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !repaired.CacheHit && !repaired.Ledger.Changes.Empty() {
		t.Fatal("failed build advanced the baseline")
	}
}

func ledgerFixture(t *testing.T, fixture string) config.Config {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(repositoryRoot(t), "assets", "testdata", "sites", fixture)
	for _, tree := range []string{"content", "assets"} {
		from := filepath.Join(source, tree)
		if _, err := os.Stat(from); os.IsNotExist(err) {
			continue
		}
		err := filepath.WalkDir(from, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(source, path)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			putLedgerFile(t, filepath.Join(root, rel), string(data))
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := config.DefaultConfig().ResolvePaths(root)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Template = filepath.Join(root, "templates", "layout.html")
	cfg.SiteURL = "https://example.com/ledger"
	cfg.GraphExplorer = false
	putLedgerFile(t, cfg.Template, "<html><body>{{.Content}}</body></html>")
	return cfg
}

func putLedgerFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
}

func assertLedgerGolden(t *testing.T, name string, report sitediff.Report) {
	t.Helper()
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	path := filepath.Join("testdata", name)
	if *updateLedgerGoldens {
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	golden, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, golden) {
		t.Fatalf("ledger differs from %s:\n%s", path, data)
	}
}
