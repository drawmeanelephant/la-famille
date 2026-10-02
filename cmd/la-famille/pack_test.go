package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/pack"
)

func TestPackCommands(t *testing.T) {
	root := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.ProjectRoot = root
	cfg.SiteName = "CLI site"
	for _, name := range []string{"backlinks.json", "graph.json", "meta.json", "search.json", "site-manifest.json"} {
		path := filepath.Join(root, cfg.OutputDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, cfg.RagDir), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, cfg.RagDir, "rag-content.md"), []byte("content"), 0600); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) (string, error) {
		t.Helper()
		cmd := setupRootCmd(cfg)
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetArgs(args)
		err := cmd.Execute()
		return out.String(), err
	}
	out, err := run("pack", "build", "--site-output", cfg.OutputDir, "--rag-dir", cfg.RagDir, "--output", "site.tar")
	if err != nil || !strings.Contains(out, "Built pack site.tar: 6 members, content root") {
		t.Fatalf("build = %q, %v", out, err)
	}
	m, err := pack.VerifyFile(filepath.Join(root, "site.tar"))
	if err != nil {
		t.Fatal(err)
	}
	if m.Site.Name != cfg.SiteName || m.Provenance.Version != buildVersion || m.Provenance.Generator != "la-famille" {
		t.Fatalf("manifest = %+v", m)
	}
	if buildCommit == "unknown" && m.Provenance.Commit != "" || buildDate == "unknown" && m.Provenance.BuildDate != "" {
		t.Fatalf("unknown build provenance not omitted: %+v", m.Provenance)
	}
	out, err = run("pack", "verify", "site.tar")
	if err != nil || !strings.Contains(out, "Verified pack site.tar: 6 members, content root "+m.ContentRoot) {
		t.Fatalf("verify = %q, %v", out, err)
	}
	for _, args := range [][]string{
		{"pack", "build"}, {"pack", "build", "--output", "site.tar"},
		{"pack", "build", "--output", "other.tar", "extra"},
		{"pack", "verify"}, {"pack", "verify", "missing.tar"},
		{"pack", "verify", "site.tar", "extra"},
	} {
		if _, err := run(args...); err == nil {
			t.Fatalf("args %v succeeded unexpectedly", args)
		}
	}
}

func TestPackVerifyIndependentOfBrokenConfig(t *testing.T) {
	for _, args := range [][]string{{"pack", "verify", "missing.tar"}, {"pack", "build", "--output", "site.tar"}} {
		cmd, st := setupRootCmdState(config.DefaultConfig())
		guardUnusableConfig(cmd, st, errors.New("broken site config"))
		cmd.SetArgs(args)
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected failure")
		}
		gotConfigError := strings.Contains(err.Error(), "broken site config")
		if gotConfigError != (args[1] == "build") {
			t.Fatalf("args %v error = %v", args, err)
		}
	}
}
