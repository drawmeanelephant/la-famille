package generator

import (
	"os"
	"path/filepath"
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
