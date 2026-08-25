package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadProjectConfigResolvesPathsFromExplicitProjectRoot(t *testing.T) {
	project := t.TempDir()
	invoker := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "config.yaml"), []byte("project_root: ignored-by-flag\ncontent_dir: docs\noutput_dir: site\nasset_dir: static\ntemplate: layouts/base.html\nrag_dir: build/rag\n"), 0600); err != nil {
		t.Fatal(err)
	}

	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(invoker); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })

	cfg, err := loadProjectConfig([]string{"--project-root", project})
	if err != nil {
		t.Fatalf("loadProjectConfig: %v", err)
	}
	if cfg.ProjectRoot != project {
		t.Errorf("ProjectRoot = %q, want %q", cfg.ProjectRoot, project)
	}
	for name, got := range map[string]string{
		"ContentDir": cfg.ContentDir,
		"OutputDir":  cfg.OutputDir,
		"AssetDir":   cfg.AssetDir,
		"Template":   cfg.Template,
		"RagDir":     cfg.RagDir,
	} {
		if !strings.HasPrefix(got, project+string(filepath.Separator)) {
			t.Errorf("%s = %q, want a path below explicit project root %q", name, got, project)
		}
	}
}

func TestLoadProjectConfigUsesConfigDirectoryWhenConfigIsExplicit(t *testing.T) {
	project := t.TempDir()
	configFile := filepath.Join(project, "nested", "site.yaml")
	if err := os.MkdirAll(filepath.Dir(configFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configFile, []byte("content_dir: docs\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadProjectConfig([]string{"--config", configFile})
	if err != nil {
		t.Fatalf("loadProjectConfig: %v", err)
	}
	wantRoot := filepath.Dir(configFile)
	if cfg.ProjectRoot != wantRoot {
		t.Errorf("ProjectRoot = %q, want config directory %q", cfg.ProjectRoot, wantRoot)
	}
	if cfg.ContentDir != filepath.Join(wantRoot, "docs") {
		t.Errorf("ContentDir = %q, want %q", cfg.ContentDir, filepath.Join(wantRoot, "docs"))
	}
}

func TestInitAndBuildFromExplicitProjectRootOutsideProject(t *testing.T) {
	project := t.TempDir()
	invoker := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(invoker); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })

	cfg, err := loadProjectConfig([]string{"--project-root", project})
	if err != nil {
		t.Fatalf("loadProjectConfig: %v", err)
	}

	root := setupRootCmd(cfg)
	root.SetArgs([]string{"init"})
	if err := root.Execute(); err != nil {
		t.Fatalf("init from outside project: %v", err)
	}

	root = setupRootCmd(cfg)
	// init scaffolds content/index.md, so the first authored post gets its
	// own slug rather than colliding with the scaffolded homepage.
	root.SetArgs([]string{"new", "hello", "--title", "Hello", "--date", "2026-08-01"})
	if err := root.Execute(); err != nil {
		t.Fatalf("new from outside project: %v", err)
	}

	root = setupRootCmd(cfg)
	root.SetArgs([]string{"build"})
	if err := root.Execute(); err != nil {
		t.Fatalf("build from outside project: %v", err)
	}

	for _, rel := range []string{
		"public/index.html",
		"public/graph/index.html",
		"public/assets/graph/explorer.js",
		".la-famille-cache.json",
	} {
		if _, err := os.Stat(filepath.Join(project, filepath.FromSlash(rel))); err != nil {
			t.Errorf("expected %s in explicit project root: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(invoker, "public")); !os.IsNotExist(err) {
		t.Errorf("build from outside project wrote output in invoker directory: %v", err)
	}
}

func TestRagOutputCanBeStagedInsidePublicFromOutsideProject(t *testing.T) {
	project := t.TempDir()
	invoker := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(invoker); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })

	cfg, err := loadProjectConfig([]string{"--project-root", project})
	if err != nil {
		t.Fatalf("loadProjectConfig: %v", err)
	}
	if err := os.MkdirAll(cfg.ContentDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.Template), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.ContentDir, "index.md"), []byte("# Home\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg.Template, []byte("<html><body>{{.Content}}</body></html>"), 0600); err != nil {
		t.Fatal(err)
	}

	root := setupRootCmd(cfg)
	root.SetArgs([]string{"rag", "--output", "public/rag-archive"})
	if err := root.Execute(); err != nil {
		t.Fatalf("rag output staging: %v", err)
	}
	if _, err := os.Stat(filepath.Join(project, "public", "rag-archive", "rag-content.md")); err != nil {
		t.Fatalf("RAG archive was not written below public: %v", err)
	}
	for _, path := range []string{
		filepath.Join(project, "rag-archive"),
		filepath.Join(invoker, "rag-archive"),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("RAG export unexpectedly wrote checkout/CWD path %s: %v", path, err)
		}
	}
}

func TestLoadProjectConfigRejectsOutputOverlapAfterResolution(t *testing.T) {
	project := t.TempDir()
	configFile := filepath.Join(project, "config.yaml")
	if err := os.WriteFile(configFile, []byte("content_dir: docs\noutput_dir: docs\n"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := loadProjectConfig([]string{"--config", configFile})
	if err == nil || !strings.Contains(err.Error(), "same directory as ContentDir") {
		t.Fatalf("loadProjectConfig error = %v, want resolved output-overlap error", err)
	}
}

func TestBootstrapCLIArgsAcceptsEqualsForms(t *testing.T) {
	root := t.TempDir()
	boot, err := bootstrapCLIArgs([]string{"build", "--project-root=" + root, "--config", "custom.yaml"})
	if err != nil {
		t.Fatalf("bootstrapCLIArgs: %v", err)
	}
	if boot.ProjectRoot != root || boot.ConfigPath != filepath.Join(mustWorkingDir(t), "custom.yaml") {
		t.Errorf("bootstrap result = %+v", boot)
	}
}

func mustWorkingDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return dir
}
