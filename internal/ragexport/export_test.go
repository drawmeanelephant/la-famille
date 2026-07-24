package ragexport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
)

// TestBundleDir covers how a configured directory is normalised before it is
// used in bundle patterns and walks, which are all resolved against
// ProjectRoot. An absolute directory is the interesting case: passing one
// straight to filepath.Join(root, dir) yields root+dir and silently exports
// nothing.
func TestBundleDir(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "srv", "site")

	cases := []struct {
		name       string
		configured string
		fallback   string
		want       string
	}{
		{"empty falls back", "", "content", "content"},
		{"whitespace falls back", "   ", "content", "content"},
		{"relative is kept", "docs", "content", "docs"},
		{"nested relative is kept", "src/pages", "content", "src/pages"},
		{"trailing slash trimmed", "docs/", "content", "docs"},
		{"dot falls back", ".", "content", "content"},
		{"absolute inside root becomes relative", filepath.Join(root, "docs"), "content", "docs"},
		{"absolute nested inside root becomes relative", filepath.Join(root, "src", "pages"), "content", "src/pages"},
		{"absolute equal to root falls back", root, "content", "content"},
		{"absolute outside root falls back", filepath.Join(string(filepath.Separator), "elsewhere", "docs"), "content", "content"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := bundleDir(c.configured, c.fallback, root); got != c.want {
				t.Errorf("bundleDir(%q, %q, %q) = %q, want %q", c.configured, c.fallback, root, got, c.want)
			}
		})
	}
}

// TestTemplateBundleTarget covers a template configured at the project root.
// filepath.Dir of "layout.html" is ".", and walking "." would list the whole
// project, so that case must resolve to the single file instead of falling
// back to an unrelated templates/ directory that the site does not use.
func TestTemplateBundleTarget(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "srv", "site")

	cases := []struct {
		name         string
		template     string
		wantDir      string
		wantFileOnly string
	}{
		{"default nested template", "templates/layout.html", "templates", ""},
		{"custom nested template", "layouts/base.html", "layouts", ""},
		{"root-level template", "layout.html", "", "layout.html"},
		{"absolute root-level template", filepath.Join(root, "layout.html"), "", "layout.html"},
		{"absolute nested template", filepath.Join(root, "layouts", "base.html"), "layouts", ""},
		{"unset falls back", "", "templates", ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := config.Config{Template: c.template, ProjectRoot: root}
			dir, file := templateBundleTarget(cfg)
			if dir != c.wantDir || file != c.wantFileOnly {
				t.Errorf("templateBundleTarget(%q) = (%q, %q), want (%q, %q)",
					c.template, dir, file, c.wantDir, c.wantFileOnly)
			}
		})
	}
}

func TestRunExport_ProjectRoot(t *testing.T) {
	// Create a temp directory to represent our project
	tempDir := t.TempDir()

	// Create some files inside the project
	err := os.MkdirAll(filepath.Join(tempDir, "internal", "foo"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tempDir, "internal", "foo", "foo.go"), []byte("package foo"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	err = os.MkdirAll(filepath.Join(tempDir, "assets"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tempDir, "assets", "logo.png"), []byte("PNG"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	// We'll run the export from a DIFFERENT working directory
	invokeDir := t.TempDir()

	cfg := config.Config{
		ProjectRoot: tempDir,
		RagDir:      filepath.Join(invokeDir, "my-rag"),
	}

	err = RunExport(cfg)
	if err != nil {
		t.Fatalf("RunExport failed: %v", err)
	}

	// Verify the output exists in my-rag
	systemBundlePath := filepath.Join(invokeDir, "my-rag", "rag-system.md")
	content, err := os.ReadFile(systemBundlePath)
	if err != nil {
		t.Fatalf("Failed to read system bundle: %v", err)
	}

	// The path in the bundle should be relative to ProjectRoot
	expectedPath := "<file path=\"internal/foo/foo.go\">"
	if !strings.Contains(string(content), expectedPath) {
		t.Errorf("Expected system bundle to contain %q, but it didn't.\nContent:\n%s", expectedPath, content)
	}

	configBundlePath := filepath.Join(invokeDir, "my-rag", "rag-config.md")
	cfgContent, err := os.ReadFile(configBundlePath)
	if err != nil {
		t.Fatalf("Failed to read config bundle: %v", err)
	}

	expectedAssetPath := "assets/logo.png"
	if !strings.Contains(string(cfgContent), expectedAssetPath) {
		t.Errorf("Expected config bundle to contain %q, but it didn't.\nContent:\n%s", expectedAssetPath, cfgContent)
	}
}

func TestRunExport_RootLevelMatch(t *testing.T) {
	tempDir := t.TempDir()

	// Should be included (root)
	err := os.WriteFile(filepath.Join(tempDir, "README.md"), []byte("Root README"), 0600)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tempDir, "root.go"), []byte("package main"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	// Should be excluded (nested)
	err = os.MkdirAll(filepath.Join(tempDir, "nested"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tempDir, "nested", "README.md"), []byte("Nested README"), 0600)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tempDir, "nested", "nested.go"), []byte("package nested"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	invokeDir := t.TempDir()
	cfg := config.Config{
		ProjectRoot: tempDir,
		RagDir:      filepath.Join(invokeDir, "my-rag"),
	}

	err = RunExport(cfg)
	if err != nil {
		t.Fatalf("RunExport failed: %v", err)
	}

	systemBundlePath := filepath.Join(invokeDir, "my-rag", "rag-system.md")
	content, err := os.ReadFile(systemBundlePath)
	if err != nil {
		t.Fatalf("Failed to read system bundle: %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "<file path=\"README.md\">") {
		t.Errorf("Expected system bundle to contain root README.md")
	}
	if strings.Contains(contentStr, "<file path=\"nested/README.md\">") {
		t.Errorf("Expected system bundle NOT to contain nested/README.md")
	}

	if !strings.Contains(contentStr, "<file path=\"root.go\">") {
		t.Errorf("Expected system bundle to contain root.go")
	}
	if strings.Contains(contentStr, "<file path=\"nested/nested.go\">") {
		t.Errorf("Expected system bundle NOT to contain nested/nested.go")
	}
}

func TestRunExport_ExcludesConfiguredOutputDirectory(t *testing.T) {
	projectRoot := t.TempDir()
	ragDir := filepath.Join(projectRoot, "internal", "rag-archive")

	writeExportTestFile(t, filepath.Join(projectRoot, "internal", "included", "included.go"), "package included")
	writeExportTestFile(t, filepath.Join(ragDir, "stale.go"), "package stale")
	writeExportTestFile(t, filepath.Join(projectRoot, "internal", "rag-archive-backup", "included.go"), "package backup")

	if err := RunExport(config.Config{ProjectRoot: projectRoot, RagDir: ragDir}); err != nil {
		t.Fatalf("RunExport failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(ragDir, "rag-system.md"))
	if err != nil {
		t.Fatalf("read system bundle: %v", err)
	}

	bundle := string(content)
	if strings.Contains(bundle, "internal/rag-archive/stale.go") {
		t.Error("system bundle included a file from the configured RAG output directory")
	}
	if !strings.Contains(bundle, "internal/included/included.go") {
		t.Error("system bundle did not include a project source file")
	}
	if !strings.Contains(bundle, "internal/rag-archive-backup/included.go") {
		t.Error("system bundle excluded a sibling directory with a similar name")
	}
}

func TestRunExport_ExcludesConfiguredOutputDirectory_AssetsAndTemplates(t *testing.T) {
	t.Run("Under Assets", func(t *testing.T) {
		projectRoot := t.TempDir()
		ragDir := filepath.Join(projectRoot, "assets", "rag-archive")

		writeExportTestFile(t, filepath.Join(projectRoot, "assets", "valid.png"), "PNG")
		writeExportTestFile(t, filepath.Join(ragDir, "stale_asset.png"), "STALE")

		if err := RunExport(config.Config{ProjectRoot: projectRoot, RagDir: ragDir}); err != nil {
			t.Fatalf("RunExport failed: %v", err)
		}

		cfgContent, err := os.ReadFile(filepath.Join(ragDir, "rag-config.md"))
		if err != nil {
			t.Fatalf("read config bundle: %v", err)
		}
		cfgStr := string(cfgContent)

		if strings.Contains(cfgStr, "assets/rag-archive") || strings.Contains(cfgStr, "stale_asset.png") {
			t.Errorf("config bundle included files/dirs from configured RAG directory under assets:\n%s", cfgStr)
		}
		if !strings.Contains(cfgStr, "assets/valid.png") {
			t.Errorf("config bundle missing valid asset:\n%s", cfgStr)
		}
	})

	t.Run("Under Templates", func(t *testing.T) {
		projectRoot := t.TempDir()
		ragDir := filepath.Join(projectRoot, "templates", "rag-archive")

		writeExportTestFile(t, filepath.Join(projectRoot, "templates", "base.html"), "<html></html>")
		writeExportTestFile(t, filepath.Join(ragDir, "stale_template.html"), "<stale></stale>")

		if err := RunExport(config.Config{ProjectRoot: projectRoot, RagDir: ragDir}); err != nil {
			t.Fatalf("RunExport failed: %v", err)
		}

		cfgContent, err := os.ReadFile(filepath.Join(ragDir, "rag-config.md"))
		if err != nil {
			t.Fatalf("read config bundle: %v", err)
		}
		cfgStr := string(cfgContent)

		if strings.Contains(cfgStr, "templates/rag-archive") || strings.Contains(cfgStr, "stale_template.html") {
			t.Errorf("config bundle included files/dirs from configured RAG directory under templates:\n%s", cfgStr)
		}
		if !strings.Contains(cfgStr, "templates/base.html") {
			t.Errorf("config bundle missing valid template:\n%s", cfgStr)
		}
	})

	t.Run("Directly Under Root", func(t *testing.T) {
		projectRoot := t.TempDir()
		ragDir := filepath.Join(projectRoot, "rag-archive")

		writeExportTestFile(t, filepath.Join(projectRoot, "content", "index.md"), "# Hello")
		writeExportTestFile(t, filepath.Join(projectRoot, "main.go"), "package main")
		writeExportTestFile(t, filepath.Join(ragDir, "stale_content.md"), "# Stale Content")
		writeExportTestFile(t, filepath.Join(ragDir, "stale.go"), "package stale")

		if err := RunExport(config.Config{ProjectRoot: projectRoot, RagDir: ragDir}); err != nil {
			t.Fatalf("RunExport failed: %v", err)
		}

		sysContent, err := os.ReadFile(filepath.Join(ragDir, "rag-system.md"))
		if err != nil {
			t.Fatalf("read system bundle: %v", err)
		}
		sysStr := string(sysContent)
		if strings.Contains(sysStr, "rag-archive/stale.go") {
			t.Errorf("system bundle included stale file from root RAG directory:\n%s", sysStr)
		}

		cntContent, err := os.ReadFile(filepath.Join(ragDir, "rag-content.md"))
		if err != nil {
			t.Fatalf("read content bundle: %v", err)
		}
		cntStr := string(cntContent)
		if strings.Contains(cntStr, "stale_content.md") {
			t.Errorf("content bundle included stale file from root RAG directory:\n%s", cntStr)
		}
	})
}

func TestRunExport_HonoursConfiguredDirectories(t *testing.T) {
	projectRoot := t.TempDir()

	writeExportTestFile(t, filepath.Join(projectRoot, "posts", "hello.md"), "# Hello")
	writeExportTestFile(t, filepath.Join(projectRoot, "posts", "jules", "internal.md"), "# Internal")
	writeExportTestFile(t, filepath.Join(projectRoot, "static", "logo.png"), "PNG")
	writeExportTestFile(t, filepath.Join(projectRoot, "layouts", "base.html"), "<html></html>")

	ragDir := filepath.Join(t.TempDir(), "my-rag")
	cfg := config.Config{
		ProjectRoot: projectRoot,
		RagDir:      ragDir,
		ContentDir:  "posts",
		AssetDir:    "static",
		Template:    "layouts/base.html",
	}

	if err := RunExport(cfg); err != nil {
		t.Fatalf("RunExport failed: %v", err)
	}

	contentBundle, err := os.ReadFile(filepath.Join(ragDir, "rag-content.md"))
	if err != nil {
		t.Fatalf("read content bundle: %v", err)
	}
	contentStr := string(contentBundle)
	if !strings.Contains(contentStr, "<file path=\"posts/hello.md\">") {
		t.Errorf("content bundle missing the configured content directory:\n%s", contentStr)
	}
	if strings.Contains(contentStr, "posts/jules/internal.md") {
		t.Errorf("content bundle included the jules directory of the configured content dir:\n%s", contentStr)
	}

	cfgBundle, err := os.ReadFile(filepath.Join(ragDir, "rag-config.md"))
	if err != nil {
		t.Fatalf("read config bundle: %v", err)
	}
	cfgStr := string(cfgBundle)
	if !strings.Contains(cfgStr, "static/logo.png") {
		t.Errorf("config bundle missing the configured asset directory:\n%s", cfgStr)
	}
	if !strings.Contains(cfgStr, "layouts/base.html") {
		t.Errorf("config bundle missing the configured template directory:\n%s", cfgStr)
	}
}

func writeExportTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}
