package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/search"
	"github.com/tbuddy/la-famille/internal/sitedata"
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

	for _, reference := range []string{"599", "617"} {
		if _, err := os.Stat(filepath.Join(cfg.OutputDir, "tags", reference, "index.html")); !os.IsNotExist(err) {
			t.Errorf("issue reference #%s unexpectedly generated an archive: %v", reference, err)
		}
		if strings.Contains(tagIndex, `href="`+reference+`/"`) ||
			strings.Contains(bodyPage, `href="../tags/`+reference+`/"`) {
			t.Errorf("issue reference #%s appeared in taxonomy navigation", reference)
		}
		if !strings.Contains(bodyPage, "#"+reference) {
			t.Errorf("issue reference #%s was removed from page prose", reference)
		}
	}
	numericArchive := readOutput(t, cfg, "tags/2026/index.html")
	if !strings.Contains(numericArchive, "Frontmatter-tagged note") {
		t.Error("explicit numeric frontmatter tag is missing its page")
	}

	manifest, err := sitedata.ReadManifest(filepath.Join(cfg.OutputDir, sitedata.ManifestFileName))
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string][]string{
		"body-note":        {"ceramics"},
		"frontmatter-note": {"2026", "ceramics"},
	}
	for _, page := range manifest.Pages {
		if want, ok := expected[page.Identity]; ok && !reflect.DeepEqual(page.Tags, want) {
			t.Errorf("manifest tags for %s = %v, want %v", page.Identity, page.Tags, want)
		}
	}
	var searchIndex []search.Item
	if err := json.Unmarshal([]byte(readOutput(t, cfg, "search.json")), &searchIndex); err != nil {
		t.Fatal(err)
	}
	for _, item := range searchIndex {
		for _, tag := range item.Tags {
			if tag == "599" || tag == "617" {
				t.Errorf("issue reference tag %q appeared in search metadata", tag)
			}
		}
	}
}
