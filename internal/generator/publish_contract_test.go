package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/publisher"
)

func TestBuildKeepsCacheOutsidePublishArtifact(t *testing.T) {
	cfg, _ := setupTestSite(t)
	if err := os.WriteFile(filepath.Join(cfg.ContentDir, "index.md"), []byte("# Home\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(cfg); err != nil {
		t.Fatalf("Build: %v", err)
	}

	if _, err := os.Stat(filepath.Join(cfg.OutputDir, publisher.CacheFileName)); !os.IsNotExist(err) {
		t.Fatalf("cache file is published at %s: stat error %v", filepath.Join(cfg.OutputDir, publisher.CacheFileName), err)
	}
	if _, err := os.Stat(filepath.Join(cfg.ProjectRoot, publisher.CacheFileName)); err != nil {
		t.Fatalf("project cache missing beside project: %v", err)
	}
	if _, err := publisher.Check(cfg.OutputDir); err != nil {
		t.Fatalf("generated publish artifact failed validation: %v", err)
	}
}

// TestPublishingDocCoversWorkflowWrittenArtifacts keeps the output contract doc
// in sync with what the deploy workflow places inside the publish artifact.
// The Pages workflow writes the RAG export into public/rag-archive after the
// build step; if the doc stops mentioning it, authors lose the contract for a
// directory that ships to production.
func TestPublishingDocCoversWorkflowWrittenArtifacts(t *testing.T) {
	docPath := filepath.Join("..", "..", "content", "docs", "publishing.md")
	data, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("failed to read publishing contract doc at %s: %v", docPath, err)
	}
	doc := string(data)

	for _, marker := range []string{"rag-archive/", "rag-system.md", "rag-config.md", "rag-content.md"} {
		if !strings.Contains(doc, marker) {
			t.Errorf("publishing.md missing RAG export contract marker %q", marker)
		}
	}

	workflowPath := filepath.Join("..", "..", ".github", "workflows", "deploy.yml")
	workflow, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("failed to read deploy workflow at %s: %v", workflowPath, err)
	}
	if !strings.Contains(string(workflow), "public/rag-archive") && strings.Contains(doc, "rag-archive/") {
		t.Errorf("deploy.yml no longer writes public/rag-archive but publishing.md still documents it; reconcile both")
	}
}
