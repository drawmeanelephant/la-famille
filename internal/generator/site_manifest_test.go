package generator

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/tbuddy/la-famille/internal/checker"
	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/sitedata"
)

var updateLedgerGoldens = flag.Bool("update-ledger-goldens", false, "Update reviewed Change Ledger golden files")

func TestBuildWritesDeterministicSiteManifestAndGolden(t *testing.T) {
	repoRoot := repositoryRoot(t)
	fixtureRoot := filepath.Join(repoRoot, "assets", "testdata", "sites", "artisanal-ceramics")
	goldenPath := filepath.Join(repoRoot, "internal", "generator", "testdata", "artisanal-ceramics-site-manifest.json")

	buildFixture := func() config.Config {
		t.Helper()
		projectRoot := t.TempDir()
		templateDir := filepath.Join(projectRoot, "templates")
		if err := os.MkdirAll(templateDir, 0755); err != nil {
			t.Fatal(err)
		}
		templatePath := filepath.Join(templateDir, "layout.html")
		if err := os.WriteFile(templatePath, []byte("<html><body>{{.Content}}</body></html>"), 0600); err != nil {
			t.Fatal(err)
		}
		cfg := config.DefaultConfig()
		cfg.ContentDir = filepath.Join(fixtureRoot, "content")
		cfg.AssetDir = filepath.Join(fixtureRoot, "assets")
		cfg.OutputDir = filepath.Join(projectRoot, "public")
		cfg.Template = templatePath
		cfg.ProjectRoot = projectRoot
		cfg.SiteURL = "https://example.com/ceramics"
		cfg.GraphExplorer = false
		return cfg
	}

	firstCfg := buildFixture()
	if _, err := Build(firstCfg); err != nil {
		t.Fatalf("first Build() error = %v", err)
	}
	firstManifest, err := os.ReadFile(filepath.Join(firstCfg.OutputDir, sitedata.ManifestFileName))
	if err != nil {
		t.Fatal(err)
	}

	cachedResult, err := Build(firstCfg)
	if err != nil {
		t.Fatalf("cached Build() error = %v", err)
	}
	if !cachedResult.CacheHit {
		cache, cacheErr := loadBuildCache(cachePath(firstCfg))
		if cacheErr != nil {
			t.Fatalf("second Build() missed cache and cache load failed: %v", cacheErr)
		}
		fingerprint, fingerprintErr := cacheFingerprint(firstCfg, firstCfg.ContentDir, filepath.Dir(firstCfg.Template), firstCfg.AssetDir, filepath.Join(firstCfg.ProjectRoot, ".gitignore"))
		if fingerprintErr != nil {
			t.Fatal(fingerprintErr)
		}
		t.Fatalf("second Build() missed cache: usable=%t fingerprintMatch=%t generatedFiles=%d", cacheUsable(cache, firstCfg.OutputDir, fingerprint), cache.Fingerprint == fingerprint, len(cache.GeneratedFiles))
	}
	cachedManifest, err := os.ReadFile(filepath.Join(firstCfg.OutputDir, sitedata.ManifestFileName))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstManifest, cachedManifest) {
		t.Fatal("manifest changed across an unchanged cached build")
	}

	secondCfg := buildFixture()
	if _, err := Build(secondCfg); err != nil {
		t.Fatalf("second clean Build() error = %v", err)
	}
	secondManifest, err := os.ReadFile(filepath.Join(secondCfg.OutputDir, sitedata.ManifestFileName))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstManifest, secondManifest) {
		t.Fatal("manifest bytes differ across clean builds of the same source tree")
	}

	if *updateLedgerGoldens {
		if err := os.WriteFile(goldenPath, firstManifest, 0600); err != nil {
			t.Fatal(err)
		}
	}
	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read manifest golden %s: %v\nGenerated manifest:\n%s", goldenPath, err, firstManifest)
	}
	if !bytes.Equal(firstManifest, golden) {
		t.Fatalf("manifest differs from golden %s:\n%s", goldenPath, firstManifest)
	}

	cache, err := loadBuildCache(cachePath(firstCfg))
	if err != nil {
		t.Fatalf("load build cache: %v", err)
	}
	manifest, err := sitedata.ReadManifest(filepath.Join(firstCfg.OutputDir, sitedata.ManifestFileName))
	if err != nil {
		t.Fatalf("read output manifest: %v", err)
	}
	if !reflect.DeepEqual(cache.Manifest, manifest) {
		t.Fatalf("cached manifest = %#v, output manifest = %#v", cache.Manifest, manifest)
	}
}

func TestManifestBackedCheckMatchesSourceLinkFindings(t *testing.T) {
	projectRoot := t.TempDir()
	contentDir := filepath.Join(projectRoot, "content")
	templateDir := filepath.Join(projectRoot, "templates")
	for _, dir := range []string{contentDir, templateDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	templatePath := filepath.Join(templateDir, "layout.html")
	if err := os.WriteFile(templatePath, []byte("{{.Content}}"), 0600); err != nil {
		t.Fatal(err)
	}

	pages := map[string]string{
		"index.md": `---
title: Home
description: Home page
---
[About](about.md)
[Broken](missing.md)
`,
		"about.md": `---
title: About
description: About page
---
[Home](index.md)
`,
		"orphan.md": `---
title: Orphan
description: Orphan page
---
No inbound links.
`,
	}
	for name, body := range pages {
		if err := os.WriteFile(filepath.Join(contentDir, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}

	cfg := config.DefaultConfig()
	cfg.ProjectRoot = projectRoot
	cfg.ContentDir = contentDir
	cfg.AssetDir = filepath.Join(projectRoot, "assets")
	cfg.OutputDir = filepath.Join(projectRoot, "public")
	cfg.Template = templatePath
	cfg.SiteURL = "https://example.com"
	cfg.GraphExplorer = false
	if _, err := Build(cfg); err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	sourceResult, err := checker.Validate(cfg)
	if err != nil {
		t.Fatalf("source-backed Validate() error = %v", err)
	}
	manifestResult, err := checker.ValidateWithManifest(cfg, filepath.Join(cfg.OutputDir, sitedata.ManifestFileName))
	if err != nil {
		t.Fatalf("manifest-backed Validate() error = %v", err)
	}

	if got, want := linkFindingKeys(manifestResult), linkFindingKeys(sourceResult); !reflect.DeepEqual(got, want) {
		t.Fatalf("manifest-backed link findings = %v, source-backed findings = %v", got, want)
	}
}

func linkFindingKeys(result *checker.Result) []string {
	var findings []string
	for _, finding := range result.Findings {
		if finding.Category == checker.CategoryBrokenLink || finding.Category == checker.CategoryOrphan {
			findings = append(findings, finding.String()+"|"+finding.Category)
		}
	}
	return findings
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
