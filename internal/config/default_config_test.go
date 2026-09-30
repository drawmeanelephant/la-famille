package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDefaultConfigYamlMatchesCanonical pins the documented default (the
// repository's config.yaml) to the config that `la-famille init` actually
// writes (#548). They share one source of truth via the gendefault generator;
// if this test fails, run `go generate ./internal/config` and commit the
// regenerated default_config_gen.go.
func TestDefaultConfigYamlMatchesCanonical(t *testing.T) {
	canonical, err := os.ReadFile(filepath.Join("..", "..", "config.yaml"))
	if err != nil {
		t.Fatalf("read canonical config.yaml: %v", err)
	}
	if defaultConfigYaml != string(canonical) {
		t.Errorf("defaultConfigYaml drifted from config.yaml; run `go generate ./internal/config` and commit default_config_gen.go")
	}
}

func TestIncludeUnusedThemeAssetsDefaultAndOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := WriteDefault(path); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.IncludeUnusedThemeAssets {
		t.Fatal("bundled theme assets should be selective by default")
	}
	if err := os.WriteFile(path, []byte("include_unused_theme_assets: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err = Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.IncludeUnusedThemeAssets {
		t.Fatal("explicitly requested unused theme assets were not enabled")
	}
}
