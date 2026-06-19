package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.SiteName != "La Famille" {
		t.Errorf("Expected DefaultConfig SiteName to be 'La Famille', got %s", cfg.SiteName)
	}
	if cfg.Theme != "retro" {
		t.Errorf("Expected DefaultConfig Theme to be 'retro', got %s", cfg.Theme)
	}
}

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Test file not exists -> returns default
	cfg, err := Load(filepath.Join(tmpDir, "nonexistent.yaml"))
	if err != nil {
		t.Fatalf("Expected no error when config file does not exist, got %v", err)
	}
	if cfg.SiteName != "La Famille" {
		t.Errorf("Expected Load to return DefaultConfig SiteName when missing, got %s", cfg.SiteName)
	}

	// Test valid yaml loading
	yamlContent := []byte(`
site_name: "Test Site"
theme: "dark"
content_dir: "my_content"
output_dir: "my_public"
template: "my_templates/layout.html"
`)
	testConfigFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(testConfigFile, yamlContent, 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	loadedCfg, err := Load(testConfigFile)
	if err != nil {
		t.Fatalf("Failed to load valid config file: %v", err)
	}

	if loadedCfg.SiteName != "Test Site" {
		t.Errorf("Expected SiteName to be 'Test Site', got %s", loadedCfg.SiteName)
	}
	if loadedCfg.Theme != "dark" {
		t.Errorf("Expected Theme to be 'dark', got %s", loadedCfg.Theme)
	}
	if loadedCfg.ContentDir != "my_content" {
		t.Errorf("Expected ContentDir to be 'my_content', got %s", loadedCfg.ContentDir)
	}
	if loadedCfg.OutputDir != "my_public" {
		t.Errorf("Expected OutputDir to be 'my_public', got %s", loadedCfg.OutputDir)
	}
	if loadedCfg.Template != "my_templates/layout.html" {
		t.Errorf("Expected Template to be 'my_templates/layout.html', got %s", loadedCfg.Template)
	}
}

func TestWriteDefault(t *testing.T) {
	tmpDir := t.TempDir()
	testConfigFile := filepath.Join(tmpDir, "config.yaml")

	err := WriteDefault(testConfigFile)
	if err != nil {
		t.Fatalf("Failed to write default config: %v", err)
	}

	cfg, err := Load(testConfigFile)
	if err != nil {
		t.Fatalf("Failed to load the generated default config: %v", err)
	}

	if cfg.SiteName != "La Famille" {
		t.Errorf("Expected generated config to have SiteName 'La Famille', got %s", cfg.SiteName)
	}
}
