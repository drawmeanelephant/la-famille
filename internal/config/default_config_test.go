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
