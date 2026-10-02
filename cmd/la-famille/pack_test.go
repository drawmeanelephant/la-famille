package main

import (
	"bytes"
	"encoding/json"
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
	baseBytes, err := os.ReadFile(filepath.Join(root, "site.tar"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, cfg.RagDir, "rag-content.md"), []byte("updated content"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := run("pack", "build", "--output", "target.tar"); err != nil {
		t.Fatal(err)
	}
	out, err = run("pack", "diff", "site.tar", "target.tar", "--output", "delta.tar")
	if err != nil || !strings.Contains(out, "0 added, 0 removed, 1 changed members") ||
		!strings.Contains(out, "~ rag-content.md") {
		t.Fatalf("diff = %q, %v", out, err)
	}
	out, err = run("pack", "diff", "site.tar", "target.tar", "--output", "delta-json.tar", "--json")
	var report pack.Comparison
	if err != nil || json.Unmarshal([]byte(out), &report) != nil ||
		len(report.Changed) != 1 || report.Changed[0] != "rag-content.md" {
		t.Fatalf("JSON diff = %q, %v", out, err)
	}
	out, err = run("pack", "apply", "site.tar", "delta.tar", "--output", "result.tar")
	if err != nil || !strings.Contains(out, "Applied pack result.tar: 6 members, content root") {
		t.Fatalf("apply = %q, %v", out, err)
	}
	result, err := os.ReadFile(filepath.Join(root, "result.tar"))
	if err != nil {
		t.Fatal(err)
	}
	target, err := os.ReadFile(filepath.Join(root, "target.tar"))
	if err != nil || !bytes.Equal(result, target) {
		t.Fatal("CLI apply is not byte-identical to target")
	}
	afterBase, err := os.ReadFile(filepath.Join(root, "site.tar"))
	if err != nil || !bytes.Equal(baseBytes, afterBase) {
		t.Fatal("CLI apply modified base")
	}
	if _, err := run("pack", "verify", "result.tar"); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"pack", "build"}, {"pack", "build", "--output", "site.tar"},
		{"pack", "build", "--output", "other.tar", "extra"},
		{"pack", "verify"}, {"pack", "verify", "missing.tar"},
		{"pack", "verify", "site.tar", "extra"},
		{"pack", "diff"}, {"pack", "diff", "site.tar", "target.tar"},
		{"pack", "diff", "missing.tar", "target.tar", "--output", "other.tar"},
		{"pack", "diff", "site.tar", "target.tar", "--output", "delta.tar"},
		{"pack", "apply"}, {"pack", "apply", "site.tar", "delta.tar"},
		{"pack", "apply", "target.tar", "delta.tar", "--output", "wrong.tar"},
		{"pack", "apply", "site.tar", "delta.tar", "--output", "result.tar"},
		{"pack", "apply", "site.tar", "missing.tar", "--output", "missing-result.tar"},
	} {
		if _, err := run(args...); err == nil {
			t.Fatalf("args %v succeeded unexpectedly", args)
		}
	}
	for _, name := range []string{"wrong.tar", "missing-result.tar", "other.tar"} {
		if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Fatalf("failed command left %s: %v", name, err)
		}
	}
}

func TestPackVerifyIndependentOfBrokenConfig(t *testing.T) {
	for _, args := range [][]string{
		{"pack", "verify", "missing.tar"}, {"pack", "build", "--output", "site.tar"},
		{"pack", "diff", "missing.tar", "target.tar", "--output", "delta.tar"},
		{"pack", "apply", "missing.tar", "delta.tar", "--output", "result.tar"},
	} {
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
