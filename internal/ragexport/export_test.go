package ragexport

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
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

func TestRunExportExcludesUnpublishedNotes(t *testing.T) {
	const canary = "VAULT582_UNPUBLISHED_BODY_CANARY_4d1f9a"
	projectRoot := t.TempDir()
	contentDir := filepath.Join(projectRoot, "vault")
	writeExportTestFile(t, filepath.Join(contentDir, "public.md"), "---\ntitle: Public\n---\nPublic note.")
	writeExportTestFile(t, filepath.Join(contentDir, "private.md"), "---\ntitle: Private\npublish: false\n---\n"+canary)
	writeExportTestFile(t, filepath.Join(contentDir, "private-raw.md"), "---\ntitle: Private Raw\nrender: false\npublish: false\n---\n"+canary)
	writeExportTestFile(t, filepath.Join(contentDir, "private.md-copy.md"), "---\ntitle: Public Copy\n---\nPublic note with a similar path.")

	ragDir := filepath.Join(t.TempDir(), "rag-archive")
	cfg := config.Config{
		ProjectRoot: projectRoot,
		ContentDir:  "vault",
		RagDir:      ragDir,
	}
	if err := RunExport(cfg); err != nil {
		t.Fatalf("RunExport: %v", err)
	}

	bundle, err := os.ReadFile(filepath.Join(ragDir, "rag-content.md"))
	if err != nil {
		t.Fatalf("read content bundle: %v", err)
	}
	got := string(bundle)
	if !strings.Contains(got, `<file path="vault/public.md">`) {
		t.Errorf("content bundle is missing the published note:\n%s", got)
	}
	if !strings.Contains(got, `<file path="vault/private.md-copy.md">`) {
		t.Errorf("content bundle is missing a published note with a similar path:\n%s", got)
	}
	if strings.Contains(got, `<file path="vault/private.md">`) ||
		strings.Contains(got, `<file path="vault/private-raw.md">`) || strings.Contains(got, canary) {
		t.Errorf("content bundle includes an unpublished note:\n%s", got)
	}
}

// TestRunExport_RagDirIsProjectRoot covers #643: a RAG output directory that
// is — or contains — the project root makes the walk's own exclusion match
// every source file, so the export must refuse it instead of writing empty
// bundles and reporting success.
func TestRunExport_RagDirIsProjectRoot(t *testing.T) {
	t.Run("RagDir equals project root", func(t *testing.T) {
		projectRoot := t.TempDir()
		writeExportTestFile(t, filepath.Join(projectRoot, "main.go"), "package main")
		writeExportTestFile(t, filepath.Join(projectRoot, "content", "index.md"), "# Hello")

		err := RunExport(config.Config{ProjectRoot: projectRoot, RagDir: projectRoot})
		if err == nil {
			t.Fatal("RunExport must refuse a RagDir that is the project root")
		}
		for _, name := range []string{"rag-system.md", "rag-config.md", "rag-content.md"} {
			if info, statErr := os.Stat(filepath.Join(projectRoot, name)); statErr == nil && info.Size() == 0 {
				t.Errorf("empty bundle %s left behind and reported as created", name)
			}
		}
	})

	t.Run("RagDir contains project root", func(t *testing.T) {
		parent := t.TempDir()
		projectRoot := filepath.Join(parent, "site")
		writeExportTestFile(t, filepath.Join(projectRoot, "main.go"), "package main")

		err := RunExport(config.Config{ProjectRoot: projectRoot, RagDir: parent})
		if err == nil {
			t.Fatal("RunExport must refuse a RagDir that contains the project root")
		}
		for _, name := range []string{"rag-system.md", "rag-config.md", "rag-content.md"} {
			if info, statErr := os.Stat(filepath.Join(parent, name)); statErr == nil && info.Size() == 0 {
				t.Errorf("empty bundle %s left behind and reported as created", name)
			}
		}
	})
}

// TestRunExport_UnreadableMatchedFile covers #644: a matched file that fails
// to read must surface an error instead of being silently omitted from the
// bundle while the command reports success.
func TestRunExport_UnreadableMatchedFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root: mode 0000 is still readable")
	}
	projectRoot := t.TempDir()
	blocked := filepath.Join(projectRoot, "blocked.go")
	writeExportTestFile(t, blocked, "package main")
	if err := os.Chmod(blocked, 0000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(blocked, 0600) })

	ragDir := filepath.Join(t.TempDir(), "rag")
	err := RunExport(config.Config{ProjectRoot: projectRoot, RagDir: ragDir})
	if err == nil {
		t.Fatal("RunExport must fail when a matched file cannot be read")
	}
	if !strings.Contains(err.Error(), "blocked.go") {
		t.Errorf("error should name the unreadable file, got: %v", err)
	}
}

// TestRunExport_FailureLeavesArchiveIntact covers #645: a failed export must
// not leave a mixed-generation archive. Bundles are staged beside the
// destination and swapped in only once every stage succeeded, so the
// previous archive stays byte-for-byte intact.
func TestRunExport_FailureLeavesArchiveIntact(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root: mode 0000 is still readable")
	}
	projectRoot := t.TempDir()
	writeExportTestFile(t, filepath.Join(projectRoot, "main.go"), "package main // v1")
	writeExportTestFile(t, filepath.Join(projectRoot, "content", "index.md"), "# Hello")

	ragDir := filepath.Join(t.TempDir(), "rag")
	cfg := config.Config{ProjectRoot: projectRoot, RagDir: ragDir}
	if err := RunExport(cfg); err != nil {
		t.Fatalf("first export failed: %v", err)
	}

	names := []string{"rag-system.md", "rag-config.md", "rag-content.md"}
	firstGen := map[string][]byte{}
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(ragDir, name))
		if err != nil {
			t.Fatalf("read first-generation %s: %v", name, err)
		}
		firstGen[name] = data
	}

	// The second generation would differ (v2 marker), then fail on the
	// unreadable content file — proving anything new on disk was committed
	// despite the failure.
	writeExportTestFile(t, filepath.Join(projectRoot, "main.go"), "package main // v2 marker")
	blocked := filepath.Join(projectRoot, "content", "blocked.md")
	writeExportTestFile(t, blocked, "# Blocked")
	if err := os.Chmod(blocked, 0000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(blocked, 0600) })

	if err := RunExport(cfg); err == nil {
		t.Fatal("second export should fail on the unreadable content file")
	}

	for _, name := range names {
		got, err := os.ReadFile(filepath.Join(ragDir, name))
		if err != nil {
			t.Fatalf("read post-failure %s: %v", name, err)
		}
		if !bytes.Equal(got, firstGen[name]) {
			t.Errorf("failed export left a mixed-generation archive: %s was replaced", name)
		}
	}

	leftovers, err := filepath.Glob(filepath.Join(filepath.Dir(ragDir), ".*.staging-*"))
	if err != nil {
		t.Fatalf("glob staging leftovers: %v", err)
	}
	if len(leftovers) > 0 {
		t.Errorf("staging directories left behind after failed export: %v", leftovers)
	}
}

// TestRunExport_RefusesSymlinkedDestination covers #646: a symlink planted at
// a bundle path must make the export fail with a clear error, never write
// through to the link target outside the project.
func TestRunExport_RefusesSymlinkedDestination(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on Windows")
	}
	projectRoot := t.TempDir()
	writeExportTestFile(t, filepath.Join(projectRoot, "main.go"), "package main")

	ragDir := filepath.Join(t.TempDir(), "rag")
	if err := os.MkdirAll(ragDir, 0755); err != nil {
		t.Fatal(err)
	}
	victim := filepath.Join(t.TempDir(), "victim")
	if err := os.WriteFile(victim, []byte("PRECIOUS"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, filepath.Join(ragDir, "rag-system.md")); err != nil {
		t.Fatal(err)
	}

	err := RunExport(config.Config{ProjectRoot: projectRoot, RagDir: ragDir})
	if err == nil {
		t.Fatal("RunExport must refuse a symlinked bundle destination")
	}
	data, err := os.ReadFile(victim)
	if err != nil {
		t.Fatalf("read victim: %v", err)
	}
	if string(data) != "PRECIOUS" {
		t.Error("export wrote through the symlinked destination")
	}
}

// TestRunExport_RefusesSymlinkedArchiveDir: a RagDir that is itself a symlink
// must be refused rather than replaced or written through.
func TestRunExport_RefusesSymlinkedArchiveDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on Windows")
	}
	projectRoot := t.TempDir()
	writeExportTestFile(t, filepath.Join(projectRoot, "main.go"), "package main")

	target := t.TempDir()
	link := filepath.Join(t.TempDir(), "rag")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	err := RunExport(config.Config{ProjectRoot: projectRoot, RagDir: link})
	if err == nil {
		t.Fatal("RunExport must refuse a symlinked archive directory")
	}
	if entries, readErr := os.ReadDir(target); readErr != nil || len(entries) != 0 {
		t.Errorf("export wrote into the symlink target directory (entries=%d, err=%v)", len(entries), readErr)
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
