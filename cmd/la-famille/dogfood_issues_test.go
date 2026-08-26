package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/publisher"
)

func runRoot(t *testing.T, cfg config.Config, args ...string) (string, string, error) {
	t.Helper()
	rootCmd := setupRootCmd(cfg)
	var outBuf, errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return outBuf.String(), errBuf.String(), err
}

// Issue #508: slugs with spaces/uppercase must be normalized on creation, not
// written verbatim into URLs and sitemap entries.
func TestNewCommandNormalizesUnsafeSlug(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir

	out, _, err := runRoot(t, cfg, "new", "My Post With Spaces", "--title", "Space Test")
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}

	normalized := filepath.Join(contentDir, "my-post-with-spaces.md")
	if _, statErr := os.Stat(normalized); statErr != nil {
		t.Fatalf("expected normalized file %s to exist: %v", normalized, statErr)
	}
	if _, statErr := os.Stat(filepath.Join(contentDir, "My Post With Spaces.md")); statErr == nil {
		t.Fatal("verbatim unsafe filename was created; expected normalization")
	}
	if !strings.Contains(out, "my-post-with-spaces.md") {
		t.Errorf("expected a notice naming the normalized slug, got:\n%s", out)
	}
}

// Issue #511: when a project root is configured, hints use the documented
// --project-root form instead of --content <abs-path>.
func TestNewCommandProjectRootHint(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ProjectRoot = tempDir
	cfg.ContentDir = contentDir

	out, _, err := runRoot(t, cfg, "new", "hinted")
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}
	if !strings.Contains(out, "la-famille --project-root "+tempDir+" check") {
		t.Errorf("expected --project-root check hint, got:\n%s", out)
	}
	if strings.Contains(out, "--content") {
		t.Errorf("hint should prefer --project-root over --content, got:\n%s", out)
	}
}

// Issue #513: a themed init must report its own layout file, not the default.
func TestInitTemplateTargetFollowsTheme(t *testing.T) {
	def, themed := initTemplateTarget("templates/layout.html", "")
	if def != "templates/layout.html" || themed {
		t.Errorf("default theme should resolve to the plain layout, got %q themed=%v", def, themed)
	}
	got, themed := initTemplateTarget("templates/layout.html", "layout-octoburger")
	if got != filepath.Join("templates", "layout-octoburger.html") || !themed {
		t.Errorf("themed init must point at the theme layout file, got %q themed=%v", got, themed)
	}
}

// writeMinimalArtifact lays down the core generated files so publisher checks
// have something structurally complete to inspect.
func writeMinimalArtifact(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "public")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"sitemap.xml", "robots.txt", "search.json", "graph.json", "backlinks.json", "meta.json"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html><title>Home</title></html>"), 0600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// Issue #510: with --json, a failing validation emits a structured report on
// stdout instead of plain-text error lines.
func TestPublishCheckJSONEmitsStructuredErrors(t *testing.T) {
	dir := writeMinimalArtifact(t)
	broken := `<html><body><a href="/does-not-exist">x</a></body></html>`
	if err := os.WriteFile(filepath.Join(dir, "broken.html"), []byte(broken), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	out, errOut, err := runRoot(t, cfg, "publish-check", "--output", dir, "--json")
	if err == nil {
		t.Fatalf("expected non-zero exit for invalid artifact, stderr: %s", errOut)
	}

	var report publishCheckReport
	if decErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &report); decErr != nil {
		t.Fatalf("--json failure output must be a JSON object, got: %q (%v)", out, decErr)
	}
	if report.Valid {
		t.Error("expected valid=false in JSON report")
	}
	if len(report.Errors) == 0 {
		t.Errorf("expected structured errors in JSON report, got: %+v", report)
	} else if !strings.Contains(report.Errors[0], "/does-not-exist") {
		t.Errorf("expected the missing-reference problem in errors, got: %v", report.Errors)
	}
	if strings.HasPrefix(strings.TrimSpace(out), "Error:") {
		t.Errorf("plain-text error leaked into stdout alongside JSON: %q", out)
	}
}

// Issue #516: generated Missing Page stubs are reported by default and fail
// under --strict.
func TestPublishCheckReportsMissingPageStubs(t *testing.T) {
	dir := writeMinimalArtifact(t)
	stubHTML := "<html><title>Missing Page - La Famille</title><body><h3>Under Construction</h3></body></html>"
	stubRel := filepath.Join("ghost-page", "index.html")
	if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, stubRel)), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, stubRel), []byte(stubHTML), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()

	out, _, err := runRoot(t, cfg, "publish-check", "--output", dir)
	if err != nil {
		t.Fatalf("stub warnings must not fail by default: %v", err)
	}
	if !strings.Contains(out, "warning: ghost-page/index.html") {
		t.Errorf("expected a stub warning on stdout, got:\n%s", out)
	}

	_, _, strictErr := runRoot(t, cfg, "publish-check", "--output", dir, "--strict")
	if strictErr == nil {
		t.Error("expected --strict to fail on generated stubs")
	} else if !strings.Contains(strictErr.Error(), "Missing Page stub") {
		t.Errorf("unexpected strict error: %v", strictErr)
	}
}

// Publisher-level contract for #516: stub detection keys off the rendered
// markers, and ordinary pages never land in Manifest.Stubs.
func TestPublisherStubDetectionContract(t *testing.T) {
	dir := writeMinimalArtifact(t)
	stubHTML := "<html><title>Missing Page - La Famille</title><body><h3>Under Construction</h3></body></html>"
	sameTitle := "<html><title>Missing Page - La Famille</title><body>real prose</body></html>"
	stubDir := filepath.Join(dir, "stub")
	if err := os.MkdirAll(stubDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stubDir, "index.html"), []byte(stubHTML), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "not-stub.html"), []byte(sameTitle), 0600); err != nil {
		t.Fatal(err)
	}

	manifest, err := publisher.Check(dir)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if len(manifest.Stubs) != 1 || manifest.Stubs[0] != "stub/index.html" {
		t.Errorf("expected exactly stub/index.html reported, got: %v", manifest.Stubs)
	}
}
