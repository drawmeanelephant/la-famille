package asset

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
)

// writeAssetFile creates a file and any parent directories it needs.
func writeAssetFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestResolveDirIsAbsoluteAndResolved pins the invariant the containment check
// depends on. filepath.EvalSymlinks keeps a relative path relative, so an
// asset directory configured as "assets" resolved to a relative path, which was
// later made absolute against the working directory without its parents being
// resolved. Comparing that against a fully resolved output path never matched,
// and the guard silently did nothing — the failure only showed up through the
// CLI, because tests using t.TempDir() pass absolute paths in.
func TestResolveDirIsAbsoluteAndResolved(t *testing.T) {
	root := t.TempDir()
	realDir := filepath.Join(root, "shared")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// A RELATIVE target, as `ln -s shared assets` produces. This matters: with
	// an absolute target EvalSymlinks happens to return an absolute path and
	// the defect hides. With a relative one it returns "shared", which is what
	// broke the containment check.
	link := filepath.Join(root, "assets")
	if err := os.Symlink("shared", link); err != nil {
		t.Skipf("symlinks not supported on this platform: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if chErr := os.Chdir(wd); chErr != nil {
			t.Fatalf("restore working directory: %v", chErr)
		}
	})
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	got := resolveDir("assets")
	if !filepath.IsAbs(got) {
		t.Errorf("resolveDir(%q) = %q, want an absolute path", "assets", got)
	}
	want, err := filepath.EvalSymlinks(realDir)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("resolveDir(%q) = %q, want the fully resolved %q", "assets", got, want)
	}
}

// TestCopyAssets_SymlinkedRootIsCopied covers an asset directory that is itself
// a symlink. filepath.WalkDir lstats its root, so such a root matched the
// inner-symlink skip on the first callback and the walk ended having copied
// nothing at all — a site published with no CSS and no warning.
func TestCopyAssets_SymlinkedRootIsCopied(t *testing.T) {
	root := t.TempDir()
	realDir := filepath.Join(root, "shared")
	writeAssetFile(t, filepath.Join(realDir, "css", "main.css"), "body{}")
	writeAssetFile(t, filepath.Join(realDir, "js", "app.js"), "console.log(1)")

	link := filepath.Join(root, "assets")
	if err := os.Symlink(realDir, link); err != nil {
		t.Skipf("symlinks not supported on this platform: %v", err)
	}

	out := filepath.Join(root, "public")
	cfg := config.Config{AssetDir: link, OutputDir: out, ProjectRoot: root}
	if err := CopyAssets(cfg, nil); err != nil {
		t.Fatalf("CopyAssets: %v", err)
	}

	for _, rel := range []string{"css/main.css", "js/app.js"} {
		if _, err := os.Stat(filepath.Join(out, "assets", filepath.FromSlash(rel))); err != nil {
			t.Errorf("asset %s missing from output through a symlinked root: %v", rel, err)
		}
	}
}

// TestCopyAssets_SymlinkedRootContainingOutputIsRefused covers the dangerous
// resolution: an asset root pointing at a directory that holds the output.
// Walking it would copy the previous build, and everything sitting beside it,
// into the new one — growing without bound and publishing files the author
// never put in assets/.
func TestCopyAssets_SymlinkedRootContainingOutputIsRefused(t *testing.T) {
	root := t.TempDir()
	site := filepath.Join(root, "site")
	writeAssetFile(t, filepath.Join(site, "content", "index.md"), "# H")
	writeAssetFile(t, filepath.Join(root, "sibling-secret.txt"), "SECRET")

	link := filepath.Join(site, "assets")
	if err := os.Symlink(root, link); err != nil {
		t.Skipf("symlinks not supported on this platform: %v", err)
	}

	out := filepath.Join(site, "public")
	cfg := config.Config{AssetDir: link, OutputDir: out, ProjectRoot: site}

	err := CopyAssets(cfg, nil)
	if err == nil {
		t.Fatal("expected an error when the asset root resolves to a directory containing the output")
	}
	if !strings.Contains(err.Error(), "output") {
		t.Errorf("error should explain the containment, got: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(out, "assets", "sibling-secret.txt")); statErr == nil {
		t.Error("a file from outside the asset tree was published")
	}
}

// TestCopyAssets_OrdinaryRootUnchanged guards against the resolution changing
// behaviour for the normal case.
func TestCopyAssets_OrdinaryRootUnchanged(t *testing.T) {
	root := t.TempDir()
	assets := filepath.Join(root, "assets")
	writeAssetFile(t, filepath.Join(assets, "css", "main.css"), "body{}")

	out := filepath.Join(root, "public")
	cfg := config.Config{AssetDir: assets, OutputDir: out, ProjectRoot: root}
	if err := CopyAssets(cfg, nil); err != nil {
		t.Fatalf("CopyAssets: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "assets", "css", "main.css")); err != nil {
		t.Errorf("ordinary asset directory regressed: %v", err)
	}
}

// TestCopyAssets_SymlinkInsideTreeStillSkipped pins the existing rule: only the
// root is resolved, links within the tree are still not followed.
func TestCopyAssets_SymlinkInsideTreeStillSkipped(t *testing.T) {
	root := t.TempDir()
	assets := filepath.Join(root, "assets")
	writeAssetFile(t, filepath.Join(assets, "css", "main.css"), "body{}")
	writeAssetFile(t, filepath.Join(root, "outside.txt"), "nope")

	if err := os.Symlink(filepath.Join(root, "outside.txt"), filepath.Join(assets, "linked.txt")); err != nil {
		t.Skipf("symlinks not supported on this platform: %v", err)
	}

	out := filepath.Join(root, "public")
	cfg := config.Config{AssetDir: assets, OutputDir: out, ProjectRoot: root}
	if err := CopyAssets(cfg, nil); err != nil {
		t.Fatalf("CopyAssets: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "assets", "linked.txt")); err == nil {
		t.Error("a symlink inside the asset tree was followed; only the root should resolve")
	}
	if _, err := os.Stat(filepath.Join(out, "assets", "css", "main.css")); err != nil {
		t.Errorf("ordinary file alongside the skipped symlink was not copied: %v", err)
	}
}
