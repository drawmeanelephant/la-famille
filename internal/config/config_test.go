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
	if cfg.AssetDir != "assets" {
		t.Errorf("Expected DefaultConfig AssetDir to be 'assets', got %s", cfg.AssetDir)
	}
	if cfg.RagDir != "rag-archive" {
		t.Errorf("Expected DefaultConfig RagDir to be 'rag-archive', got %s", cfg.RagDir)
	}
	if !cfg.CookieNotice {
		t.Errorf("Expected DefaultConfig CookieNotice to be true, got %v", cfg.CookieNotice)
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
asset_dir: "my_assets"
rag_dir: "my_rag"
port: 8081
cookienotice: false
`)
	testConfigFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(testConfigFile, yamlContent, 0600); err != nil {
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
	if loadedCfg.AssetDir != "my_assets" {
		t.Errorf("Expected AssetDir to be 'my_assets', got %s", loadedCfg.AssetDir)
	}
	if loadedCfg.RagDir != "my_rag" {
		t.Errorf("Expected RagDir to be 'my_rag', got %s", loadedCfg.RagDir)
	}
	if loadedCfg.Port != 8081 {
		t.Errorf("Expected Port to be 8081, got %d", loadedCfg.Port)
	}
	if loadedCfg.CookieNotice != false {
		t.Errorf("Expected CookieNotice to be false, got %v", loadedCfg.CookieNotice)
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
	if !cfg.CookieNotice {
		t.Errorf("Expected generated config to have CookieNotice true, got %v", cfg.CookieNotice)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "valid default config",
			cfg:     DefaultConfig(),
			wantErr: false,
		},
		{
			name: "empty content_dir",
			cfg: func() Config {
				c := DefaultConfig()
				c.ContentDir = ""
				return c
			}(),
			wantErr: true,
		},
		{
			name: "invalid port (too low)",
			cfg: func() Config {
				c := DefaultConfig()
				c.Port = 0
				return c
			}(),
			wantErr: true,
		},
		{
			name: "invalid port (too high)",
			cfg: func() Config {
				c := DefaultConfig()
				c.Port = 70000
				return c
			}(),
			wantErr: true,
		},
		{
			name: "absolute path for output_dir",
			cfg: func() Config {
				c := DefaultConfig()
				c.OutputDir = "/etc/passwd"
				return c
			}(),
			wantErr: true,
		},
		{
			name: "empty output_dir",
			cfg: func() Config {
				c := DefaultConfig()
				c.OutputDir = ""
				return c
			}(),
			wantErr: true,
		},
		{
			name: "empty template",
			cfg: func() Config {
				c := DefaultConfig()
				c.Template = ""
				return c
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
