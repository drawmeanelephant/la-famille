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
	}

	dirFields := []struct {
		name     string
		setEmpty func(*Config)
		setAbs   func(*Config)
	}{
		{name: "ContentDir", setEmpty: func(c *Config) { c.ContentDir = "" }, setAbs: func(c *Config) { c.ContentDir = "/etc/passwd" }},
		{name: "OutputDir", setEmpty: func(c *Config) { c.OutputDir = "" }, setAbs: func(c *Config) { c.OutputDir = "/etc/passwd" }},
		{name: "Template", setEmpty: func(c *Config) { c.Template = "" }, setAbs: func(c *Config) { c.Template = "/etc/passwd" }},
		{name: "AssetDir", setEmpty: func(c *Config) { c.AssetDir = "" }, setAbs: func(c *Config) { c.AssetDir = "/etc/passwd" }},
		{name: "RagDir", setEmpty: func(c *Config) { c.RagDir = "" }, setAbs: func(c *Config) { c.RagDir = "/etc/passwd" }},
		{name: "ProjectRoot", setEmpty: func(c *Config) { c.ProjectRoot = "" }, setAbs: func(c *Config) { c.ProjectRoot = "/etc/passwd" }},
	}

	for _, field := range dirFields {
		tests = append(tests, struct {
			name    string
			cfg     Config
			wantErr bool
		}{
			name: "empty " + field.name,
			cfg: func() Config {
				c := DefaultConfig()
				field.setEmpty(&c)
				return c
			}(),
			wantErr: true,
		})
		tests = append(tests, struct {
			name    string
			cfg     Config
			wantErr bool
		}{
			name: "absolute path for " + field.name,
			cfg: func() Config {
				c := DefaultConfig()
				field.setAbs(&c)
				return c
			}(),
			wantErr: true,
		})
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

func TestSiteURLValidation(t *testing.T) {
	valid := []string{"https://example.com", "http://localhost:8080/site///"}
	for _, value := range valid {
		c := DefaultConfig()
		c.SiteURL = value
		if err := c.Validate(); err != nil {
			t.Errorf("Validate(%q) = %v", value, err)
		}
	}
	invalid := []string{"example.com", "ftp://example.com", "https:///missing-host", "https://user@example.com", "https://example.com/?q=1", "https://example.com/#frag", "https://example.com/a/../b", "https://example.com/a/%2e%2e/b"}
	for _, value := range invalid {
		c := DefaultConfig()
		c.SiteURL = value
		if err := c.Validate(); err == nil {
			t.Errorf("Validate(%q) unexpectedly succeeded", value)
		}
	}
}

func TestLegacySiteURLAlias(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(p, []byte("site_url: https://example.com/\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.SiteURL != "https://example.com/" {
		t.Fatalf("legacy site_url not accepted: %q", c.SiteURL)
	}
}

func TestLegacySiteURLValidation(t *testing.T) {
	c := DefaultConfig()
	c.LegacySiteURL = "https://example.com/../private"
	if err := c.Validate(); err == nil {
		t.Fatal("Validate unexpectedly accepted an invalid legacy site_url")
	}
}

func TestURLForOutputPath(t *testing.T) {
	tests := []struct{ name, site, output, want string }{
		{"root", "https://example.com", "index.html", "https://example.com/"},
		{"root page", "https://example.com", "about/index.html", "https://example.com/about/"},
		{"nested index", "https://example.com", "docs/index.html", "https://example.com/docs/"},
		{"nested page", "https://example.com/", "docs/install/index.html", "https://example.com/docs/install/"},
		{"slug", "https://example.com///", "guides/quick-start/index.html", "https://example.com/guides/quick-start/"},
		{"slug override output", "https://example.com", "posts/custom/index.html", "https://example.com/posts/custom/"},
		{"empty", "", "about/index.html", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := (Config{SiteURL: tt.site}).URLForOutputPath(tt.output); got != tt.want {
				t.Errorf("URLForOutputPath = %q, want %q", got, tt.want)
			}
		})
	}
}
