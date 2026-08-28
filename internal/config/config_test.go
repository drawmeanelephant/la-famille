package config

import (
	"os"
	"path/filepath"
	"reflect"
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

// A failed Load must not hand back anything a caller could build with. These
// two tests pin the contract for both failure paths: a file that parses
// partially, and a file that cannot be read at all.

func TestLoadUnparsableConfigIsNotUsable(t *testing.T) {
	tmpDir := t.TempDir()
	testConfigFile := filepath.Join(tmpDir, "config.yaml")
	// A quoted int is a routine YAML mistake: yaml.v2 applies site_name and
	// output_dir before it fails on port, so the struct it filled in is half
	// applied -- file values for some fields, defaults for the rest.
	yamlContent := []byte("site_name: \"X\"\noutput_dir: content\nport: \"not-a-number\"\n")
	if err := os.WriteFile(testConfigFile, yamlContent, 0600); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	cfg, err := Load(testConfigFile)
	if err == nil {
		t.Fatal("Expected Load to report an error for an unparsable config file")
	}
	if !reflect.DeepEqual(cfg, Config{}) {
		t.Errorf("Load returned a non-zero Config alongside a parse error: %+v", cfg)
	}
}

func TestLoadUnreadableConfigIsNotUsable(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root: mode 0000 is still readable")
	}
	tmpDir := t.TempDir()
	testConfigFile := filepath.Join(tmpDir, "config.yaml")
	// A real config with a siteurl the operator cares about. Falling back to
	// defaults here would silently drop it and build every canonical URL,
	// og:url and sitemap entry empty.
	yamlContent := []byte("siteurl: \"https://example.com/my-site\"\n")
	if err := os.WriteFile(testConfigFile, yamlContent, 0600); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}
	if err := os.Chmod(testConfigFile, 0000); err != nil {
		t.Fatalf("Failed to chmod test config file: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(testConfigFile, 0600) })

	cfg, err := Load(testConfigFile)
	if err == nil {
		t.Fatal("Expected Load to report an error for an unreadable config file")
	}
	if !reflect.DeepEqual(cfg, Config{}) {
		t.Errorf("Load returned a non-zero Config alongside a read error: %+v", cfg)
	}
}

func TestLoadMissingConfigStillReturnsDefaults(t *testing.T) {
	// The zero-Config contract applies to files that exist but cannot be used.
	// A missing config.yaml remains a supported, error-free way to run on
	// defaults, and must keep working.
	cfg, err := Load(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	if err != nil {
		t.Fatalf("Load(missing) returned an error: %v", err)
	}
	if !reflect.DeepEqual(cfg, DefaultConfig()) {
		t.Errorf("Load(missing) = %+v, want DefaultConfig()", cfg)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Load(missing) returned a config that does not validate: %v", err)
	}
}

func TestResolvePathsMakesProjectRootExplicit(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "tmp", "la-famille-site")
	cfg := DefaultConfig()
	cfg.ContentDir = "docs"
	cfg.OutputDir = "dist"
	cfg.AssetDir = "static"
	cfg.RagDir = "build/rag"
	cfg.Template = "layouts/base.html"

	resolved, err := cfg.ResolvePaths(root)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.ProjectRoot != root {
		t.Errorf("ProjectRoot = %q, want %q", resolved.ProjectRoot, root)
	}
	for name, got := range map[string]string{
		"ContentDir": resolved.ContentDir,
		"OutputDir":  resolved.OutputDir,
		"AssetDir":   resolved.AssetDir,
		"RagDir":     resolved.RagDir,
		"Template":   resolved.Template,
	} {
		if !filepath.IsAbs(got) {
			t.Errorf("%s = %q, want absolute path", name, got)
		}
	}
	if err := resolved.ValidateResolved(); err != nil {
		t.Fatalf("ValidateResolved: %v", err)
	}
}

func TestResolvePathsPreservesExplicitAbsoluteOverrides(t *testing.T) {
	root := t.TempDir()
	externalAssets := filepath.Join(t.TempDir(), "assets")
	cfg := DefaultConfig()
	cfg.AssetDir = externalAssets
	resolved, err := cfg.ResolvePaths(root)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.AssetDir != externalAssets {
		t.Errorf("AssetDir = %q, want explicit absolute override %q", resolved.AssetDir, externalAssets)
	}
	if err := resolved.ValidateResolved(); err != nil {
		t.Fatalf("ValidateResolved: %v", err)
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
		setEmpty func(*Config)
		setAbs   func(*Config)
		name     string
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
	invalid := []string{
		"example.com", "ftp://example.com", "https:///missing-host", "https://user@example.com",
		"https://example.com/?q=1", "https://example.com/#frag",
		"https://example.com/a/../b", "https://example.com/a/%2e%2e/b",
		// Bare "?" sets url.URL.ForceQuery, not RawQuery, but String() still
		// re-emits it into every canonical URL.
		"https://example.com/?",
		// An encoded separator keeps the escaped path one opaque segment; the
		// traversal only appears once the path is decoded.
		"https://example.com/a/..%2Fb", "https://example.com/a/..%2fb", "https://example.com/..%2F..%2Fetc",
	}
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

// Issue #531: the bare-string emit path (search.json, discovery fallbacks,
// graph) must escape the same way URLForOutputPath's url.URL does, or a raw
// space reaches JSON consumers unencoded.
func TestPublicPathForOutputEscapesSegments(t *testing.T) {
	tests := []struct{ name, site, output, want string }{
		{"no siteurl", "", "about/index.html", "/about/"},
		{"no siteurl spaced", "", "my post/index.html", "/my%20post/"},
		{"no siteurl unicode", "", "über café/index.html", "/%C3%BCber%20caf%C3%A9/"},
		{"base path", "https://example.com/my-site", "about/index.html", "/my-site/about/"},
		{"base path spaced", "https://example.com/my-site", "my post/index.html", "/my-site/my%20post/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := (Config{SiteURL: tt.site}).PublicPathForOutput(tt.output); got != tt.want {
				t.Errorf("PublicPathForOutput = %q, want %q", got, tt.want)
			}
		})
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
		// A filename-derived slug with a space or non-ASCII character must be
		// percent-escaped once so the <loc> stays sitemaps-valid (#531).
		{"spaced slug", "https://example.com", "my post/index.html", "https://example.com/my%20post/"},
		{"unicode slug", "https://example.com", "über café/index.html", "https://example.com/%C3%BCber%20caf%C3%A9/"},
		{"spaced slug under base", "https://example.com/my-site", "my post/index.html", "https://example.com/my-site/my%20post/"},
		// Rejected siteurls must produce no URL at all, not a traversal- or
		// query-bearing canonical that crawlers resolve elsewhere.
		{"encoded traversal separator", "https://example.com/..%2F..%2Fetc", "about/index.html", ""},
		{"forced empty query", "https://example.com/?", "about/index.html", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := (Config{SiteURL: tt.site}).URLForOutputPath(tt.output); got != tt.want {
				t.Errorf("URLForOutputPath = %q, want %q", got, tt.want)
			}
		})
	}
}
