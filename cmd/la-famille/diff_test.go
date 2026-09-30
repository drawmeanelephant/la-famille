package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
	sitediff "github.com/tbuddy/la-famille/internal/diff"
	"github.com/tbuddy/la-famille/internal/sitedata"
)

func TestDiffCommandJSONAndSummary(t *testing.T) {
	rootDir := t.TempDir()
	beforeDir := filepath.Join(rootDir, "before")
	afterDir := filepath.Join(rootDir, "after")
	writeDiffManifest(t, beforeDir, sitedata.Manifest{
		Version: sitedata.ManifestVersion,
		Pages:   []sitedata.ManifestPage{{Identity: "index", SourcePath: "index.md", Rendered: true, URL: "/", Title: "Home"}},
	})
	writeDiffManifest(t, afterDir, sitedata.Manifest{
		Version: sitedata.ManifestVersion,
		Pages: []sitedata.ManifestPage{
			{Identity: "index", SourcePath: "index.md", Rendered: true, URL: "/", Title: "Home"},
			{Identity: "about", SourcePath: "about.md", Rendered: true, URL: "/about/", Title: "About"},
		},
	})

	var jsonOut bytes.Buffer
	cmd := setupRootCmd(config.Config{ProjectRoot: rootDir, OutputDir: "public"})
	cmd.SetOut(&jsonOut)
	cmd.SetArgs([]string{"diff", beforeDir, afterDir, "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("diff --json error = %v", err)
	}
	var output struct {
		Before  string          `json:"before"`
		After   string          `json:"after"`
		Changes sitediff.Report `json:"changes"`
	}
	if err := json.Unmarshal(jsonOut.Bytes(), &output); err != nil {
		t.Fatalf("parse JSON output: %v\n%s", err, jsonOut.String())
	}
	if output.Before != beforeDir || output.After != afterDir {
		t.Errorf("JSON inputs = %q → %q, want provided paths", output.Before, output.After)
	}
	if len(output.Changes.AddedPages) != 1 || output.Changes.AddedPages[0].Identity != "about" {
		t.Errorf("added pages = %+v, want about", output.Changes.AddedPages)
	}

	var summary bytes.Buffer
	cmd = setupRootCmd(config.Config{ProjectRoot: rootDir, OutputDir: "public"})
	cmd.SetOut(&summary)
	cmd.SetArgs([]string{"diff", beforeDir, afterDir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("diff summary error = %v", err)
	}
	for _, want := range []string{"Site diff:", "Pages: 1 added, 0 removed, 0 changed", "+ about (About)"} {
		if !strings.Contains(summary.String(), want) {
			t.Errorf("summary missing %q:\n%s", want, summary.String())
		}
	}
}

func TestDiffCommandAcceptsManifestFileInputs(t *testing.T) {
	rootDir := t.TempDir()
	beforeFile := filepath.Join(rootDir, "before.json")
	afterFile := filepath.Join(rootDir, "after.json")
	for _, path := range []string{beforeFile, afterFile} {
		if err := os.WriteFile(path, []byte(`{"version":1,"pages":[]}`), 0600); err != nil {
			t.Fatal(err)
		}
	}

	var output bytes.Buffer
	cmd := setupRootCmd(config.Config{ProjectRoot: rootDir, OutputDir: "public"})
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"diff", beforeFile, afterFile, "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("diff with manifest-file inputs: %v", err)
	}
	if !strings.Contains(output.String(), `"changes"`) {
		t.Errorf("JSON output does not contain changes object:\n%s", output.String())
	}
}

func writeDiffManifest(t *testing.T, dir string, manifest sitedata.Manifest) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := sitedata.WriteManifest(dir, manifest); err != nil {
		t.Fatal(err)
	}
}
