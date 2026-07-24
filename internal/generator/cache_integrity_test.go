package generator

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A cache hit republishes whatever is already in the output directory, so the
// cache has to notice when that content no longer matches what was generated.
func TestBuild_ModifiedOutputInvalidatesCache(t *testing.T) {
	cfg, _ := setupTestSite(t)

	if res, err := Build(cfg); err != nil || res.CacheHit {
		t.Fatalf("initial build: err=%v, cacheHit=%v", err, res.CacheHit)
	}
	res2, err := Build(cfg)
	if err != nil {
		t.Fatalf("second build failed: %v", err)
	}
	if !res2.CacheHit {
		t.Fatalf("unchanged site should be a cache hit, got a miss")
	}

	pagePath := filepath.Join(cfg.OutputDir, "page1", "index.html")
	original, err := os.ReadFile(pagePath)
	if err != nil {
		t.Fatalf("read generated page: %v", err)
	}
	// Same path set, different bytes: only a content check can see this.
	if err := os.WriteFile(pagePath, []byte("<html>TAMPERED CONTENT</html>"), 0600); err != nil {
		t.Fatal(err)
	}

	res3, err := Build(cfg)
	if err != nil {
		t.Fatalf("rebuild after output tampering failed: %v", err)
	}
	if res3.CacheHit {
		t.Errorf("modified generated file should invalidate the cache, got a hit")
	}

	restored, err := os.ReadFile(pagePath)
	if err != nil {
		t.Fatalf("read regenerated page: %v", err)
	}
	if string(restored) != string(original) {
		t.Errorf("tampered page was not regenerated: got %q, want %q", restored, original)
	}
}

// Output produced by a different build of the generator must never be reused:
// the fingerprint otherwise covers only the site's inputs, which are identical.
func TestBuild_DifferentGeneratorInvalidatesCache(t *testing.T) {
	cfg, _ := setupTestSite(t)

	original := generatorIdentity
	t.Cleanup(func() { generatorIdentity = original })
	generatorIdentity = func() string { return "generator-v1" }

	if res, err := Build(cfg); err != nil || res.CacheHit {
		t.Fatalf("initial build: err=%v, cacheHit=%v", err, res.CacheHit)
	}
	res2, err := Build(cfg)
	if err != nil {
		t.Fatalf("second build failed: %v", err)
	}
	if !res2.CacheHit {
		t.Fatalf("unchanged site should be a cache hit, got a miss")
	}

	generatorIdentity = func() string { return "generator-v2" }
	res3, err := Build(cfg)
	if err != nil {
		t.Fatalf("rebuild with a different generator failed: %v", err)
	}
	if res3.CacheHit {
		t.Errorf("a different generator build should invalidate the cache, got a hit")
	}
}

// What distinguishes one build of the generator from another is the binary, so
// the identity has to be derived from the running executable.
func TestExecutableIdentityIsStableAndSpecific(t *testing.T) {
	first := executableIdentity()
	if first == "" {
		t.Fatal("executableIdentity() = \"\", want a non-empty value")
	}
	if second := executableIdentity(); second != first {
		t.Errorf("executableIdentity() is not stable: %q then %q", first, second)
	}

	exe, err := os.Executable()
	if err != nil {
		t.Skipf("os.Executable() unavailable: %v", err)
	}
	data, err := os.ReadFile(exe) // #nosec G304 - path comes from os.Executable
	if err != nil {
		t.Skipf("test binary unreadable: %v", err)
	}
	sum := sha256.Sum256(data)
	if want := hex.EncodeToString(sum[:]); first != want {
		t.Errorf("executableIdentity() = %q, want the hash of the running binary %q", first, want)
	}
}

// filepath.WalkDir lstats its root, so a root that is itself a symlink is
// skipped and hashes to nothing - while the template loader follows it and uses
// the files. Every later edit then reports cache=hit and republishes stale HTML.
func TestBuild_SymlinkedTemplateDirIsFingerprinted(t *testing.T) {
	cfg, tempDir := setupTestSite(t)

	realTemplates := filepath.Join(tempDir, "real-templates")
	if err := os.MkdirAll(realTemplates, 0755); err != nil {
		t.Fatal(err)
	}
	layout := filepath.Join(realTemplates, "layout.html")
	if err := os.WriteFile(layout, []byte("<!DOCTYPE html><html><body>V1{{.Content}}</body></html>"), 0600); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(tempDir, "linked-templates")
	if err := os.Symlink(realTemplates, linked); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	cfg.Template = filepath.Join(linked, "layout.html")

	if _, err := Build(cfg); err != nil {
		t.Fatalf("first Build() error = %v", err)
	}
	if got := readOutput(t, cfg, filepath.Join("page1", "index.html")); !strings.Contains(got, "V1") {
		t.Fatalf("page1 = %q, want it rendered through the symlinked template", got)
	}

	if err := os.WriteFile(layout, []byte("<!DOCTYPE html><html><body>V2{{.Content}}</body></html>"), 0600); err != nil {
		t.Fatal(err)
	}
	res, err := Build(cfg)
	if err != nil {
		t.Fatalf("second Build() error = %v", err)
	}
	if res.CacheHit {
		t.Errorf("editing a template behind a symlinked directory should invalidate the cache, got a hit")
	}
	if got := readOutput(t, cfg, filepath.Join("page1", "index.html")); !strings.Contains(got, "V2") {
		t.Errorf("page1 = %q, want the edited template republished", got)
	}
}

// The same defect on the asset root: its contents must reach the fingerprint.
func TestBuild_SymlinkedAssetDirIsFingerprinted(t *testing.T) {
	cfg, tempDir := setupTestSite(t)

	realAssets := filepath.Join(tempDir, "real-assets")
	if err := os.MkdirAll(realAssets, 0755); err != nil {
		t.Fatal(err)
	}
	style := filepath.Join(realAssets, "style.css")
	if err := os.WriteFile(style, []byte("body { color: black; }"), 0600); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(tempDir, "linked-assets")
	if err := os.Symlink(realAssets, linked); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	cfg.AssetDir = linked

	if _, err := Build(cfg); err != nil {
		t.Fatalf("first Build() error = %v", err)
	}
	if res, err := Build(cfg); err != nil || !res.CacheHit {
		t.Fatalf("unchanged rebuild: err=%v, cacheHit=%v, want a hit", err, res.CacheHit)
	}

	if err := os.WriteFile(style, []byte("body { color: rebecca; }"), 0600); err != nil {
		t.Fatal(err)
	}
	res, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build() after asset edit error = %v", err)
	}
	if res.CacheHit {
		t.Errorf("editing an asset behind a symlinked directory should invalidate the cache, got a hit")
	}
}
