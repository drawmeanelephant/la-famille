package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
)

func TestBuildBodyTagsShareFrontmatterTaxonomyArchive(t *testing.T) {
	repoRoot := repositoryRoot(t)
	fixtureRoot := filepath.Join(repoRoot, "assets", "testdata", "sites", "body-tags")
	projectRoot := t.TempDir()
	templatePath := filepath.Join(projectRoot, "layout.html")
	if err := os.WriteFile(templatePath, []byte("<html><body>{{.Content}}</body></html>"), 0600); err != nil {
		t.Fatal(err)
	}
	assetDir := filepath.Join(projectRoot, "assets")
	if err := os.MkdirAll(assetDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ProjectRoot = projectRoot
	cfg.ContentDir = filepath.Join(fixtureRoot, "content")
	cfg.OutputDir = filepath.Join(projectRoot, "public")
	cfg.AssetDir = assetDir
	cfg.Template = templatePath
	cfg.GraphExplorer = false

	if _, err := Build(cfg); err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	tagIndex := readOutput(t, cfg, "tags/index.html")
	if !strings.Contains(tagIndex, `href="ceramics/"`) {
		t.Errorf("tags/index.html does not list body tag ceramics:\n%s", tagIndex)
	}

	archive := readOutput(t, cfg, "tags/ceramics/index.html")
	for _, want := range []string{
		"Body-tagged note",
		"Frontmatter-tagged note",
		`href="../../body-note/"`,
		`href="../../frontmatter-note/"`,
	} {
		if !strings.Contains(archive, want) {
			t.Errorf("tags/ceramics/index.html missing %q:\n%s", want, archive)
		}
	}

	bodyPage := readOutput(t, cfg, "body-note/index.html")
	if !strings.Contains(bodyPage, `class="tag-link" href="../tags/ceramics/"`) {
		t.Errorf("body-tagged page has no tag archive link:\n%s", bodyPage)
	}
	if _, err := os.Stat(filepath.Join(cfg.OutputDir, "tags", "heading-only", "index.html")); !os.IsNotExist(err) {
		t.Errorf("heading hashtag unexpectedly generated an archive: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cfg.OutputDir, "tags", "code-only", "index.html")); !os.IsNotExist(err) {
		t.Errorf("code hashtag unexpectedly generated an archive: %v", err)
	}
}
