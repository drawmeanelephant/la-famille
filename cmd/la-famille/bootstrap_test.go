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

func TestRagFromParentWithRelativeProjectRoot(t *testing.T) {
	for _, tc := range []struct {
		name           string
		relativeOutput bool
	}{
		{name: "absolute output"},
		{name: "project-relative output", relativeOutput: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parent, project, relativeRoot := nestedProjectFixture(t)
			t.Chdir(parent)

			output := filepath.Join(project, "public", "rag-archive")
			if tc.relativeOutput {
				output = filepath.Join("public", "rag-archive")
			}
			args := []string{"--project-root", relativeRoot, "rag", "--output", output}
			cfg, err := loadProjectConfig(args)
			if err != nil {
				t.Fatalf("loadProjectConfig: %v", err)
			}
			root := setupRootCmd(cfg)
			root.SetArgs(args)
			if err := root.Execute(); err != nil {
				t.Fatalf("rag from parent: %v", err)
			}

			archive := filepath.Join(project, "public", "rag-archive")
			for name, want := range map[string]string{
				"rag-system.md":  `<file path="README.md">`,
				"rag-content.md": `<file path="docs/index.md">`,
				"rag-config.md":  "static/logo.png",
			} {
				data, err := os.ReadFile(filepath.Join(archive, name))
				if err != nil {
					t.Fatalf("read %s: %v", name, err)
				}
				if !strings.Contains(string(data), want) {
					t.Errorf("%s missing %q:\n%s", name, want, data)
				}
			}
			if _, err := os.Stat(filepath.Join(project, relativeRoot)); !os.IsNotExist(err) {
				t.Errorf("project root was nested twice: %v", err)
			}
			if _, err := os.Stat(filepath.Join(parent, "public")); !os.IsNotExist(err) {
				t.Errorf("RAG output was written relative to the invocation directory: %v", err)
			}
		})
	}
}

func TestBuildFromParentWithRelativeProjectRoot(t *testing.T) {
	parent, project, relativeRoot := nestedProjectFixture(t)
	t.Chdir(parent)

	args := []string{"--project-root=" + relativeRoot, "build"}
	cfg, err := loadProjectConfig(args)
	if err != nil {
		t.Fatalf("loadProjectConfig: %v", err)
	}
	root := setupRootCmd(cfg)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		t.Fatalf("build from parent: %v", err)
	}
	for _, name := range []string{"public/index.html", ".la-famille-cache.json"} {
		if _, err := os.Stat(filepath.Join(project, name)); err != nil {
			t.Errorf("expected %s in selected project: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(project, relativeRoot)); !os.IsNotExist(err) {
		t.Errorf("project root was nested twice: %v", err)
	}
}

func nestedProjectFixture(t *testing.T) (parent, project, relativeRoot string) {
	t.Helper()
	parent = t.TempDir()
	relativeRoot = filepath.Join("sites", "zai")
	project = filepath.Join(parent, relativeRoot)
	for name, body := range map[string]string{
		"config.yaml":         "content_dir: docs\nasset_dir: static\ntemplate: layouts/base.html\n",
		"README.md":           "# Site\n",
		"docs/index.md":       "# Home\n",
		"static/logo.png":     "PNG",
		"layouts/base.html":   "<html><body>{{.Content}}</body></html>",
		"internal/app/app.go": "package app\n",
	} {
		path := filepath.Join(project, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return parent, project, relativeRoot
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
