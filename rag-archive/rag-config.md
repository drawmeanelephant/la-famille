<file path=".jules/architecture.md">
<content>
# Architecture Notes

## Refactoring Seams
* 2026-06-20: Extracted `GatherMetadata` (which walks directories and parses markdown frontmatter) out of `cmd/la-famille/main.go` into a new package `internal/content`. This improves the modularity of the codebase by moving file-system reading and parsing logic out of the CLI's main entry point, preparing it for potentially being used by other parts of the system (like the taxonomy or search features) independently of the main site generation loop.
* 2026-06-25: Extracted HTML rendering logic out of `cmd/la-famille/main.go` and `internal/generator/generator.go` into a new package `internal/render`. This isolates layout template parsing and HTML execution from the primary site generation loop, preparing the generation step for easier refactoring and sharing template responsibilities.

</content>
</file>

<file path=".jules/palette.md">
<content>
## 2025-02-23 - Skip-to-content and Semantic Navigation
**Learning:** Found that the default layout templates used a generic `div` for the navigation bar and lacked a skip-to-content link, which significantly degraded the keyboard navigation and screen reader experience.
**Action:** Added a `nav` element with `aria-label="Main Navigation"`, an `id="main-content"` on the `main` tag, and a visually hidden `Skip to content` link using Tailwind utilities (`sr-only focus:not-sr-only`) at the top of `<body>`. Will ensure future templates incorporate these semantic and accessible patterns by default.

## 2025-02-23 - Focus States and Decorative Icons in Custom Themes
**Learning:** The `cyberpunk.html` theme had custom hover states for sidebar navigation links (`hover:bg-secondary hover:text-secondary-content border-2 border-transparent hover:border-primary`) but lacked corresponding `focus-visible` utilities, making keyboard navigation hard to track. Furthermore, the inline SVGs lacked `aria-hidden="true"` and `focusable="false"`, adding noise for screen reader users.
**Action:** Always ensure that custom `hover` states have matching `focus-visible` states (e.g., `focus-visible:bg-secondary focus-visible:text-secondary-content focus-visible:border-primary focus-visible:outline-none`). Additionally, all decorative SVG icons used alongside text labels must include `aria-hidden="true"` and `focusable="false"` to optimize the screen reader experience.

## 2026-06-19 - Add missing focus-visible states to template links
**Learning:** Keyboard accessibility is compromised when custom `hover` states (like `hover:underline` or `hover:bg-primary/20`) are added without corresponding `focus-visible` states. Default browser focus rings might not provide sufficient contrast or might conflict with custom hover styling.
**Action:** Always pair interactive `hover` state utilities with matching `focus-visible` equivalents (e.g., `focus-visible:outline-none focus-visible:bg-primary/20`) to ensure custom styling is visible and functional for keyboard users.

## 2026-06-20 - Default Layout Focus Accessibility
**Learning:** The default layout in `templates/layout.html` lacked keyboard focus visibility for main navigation, "skip to content" link, and article links, making the site difficult to navigate via keyboard despite having semantic HTML in place.
**Action:** When creating or maintaining layout templates, always explicitly define `focus-visible` states using Tailwind utilities (e.g., `focus-visible:ring-2`, `focus-visible:outline`, `focus-visible:prose-a:outline`) for all interactive elements, including utility links like "Skip to content".
## 2026-06-19 - Tailwind Typography State Modifier Ordering
**Learning:** When using state modifiers (like `hover:` or `focus-visible:`) in combination with Tailwind Typography element modifiers (like `prose-a:`), the state modifier must come *after* the element modifier (e.g., `prose-a:focus-visible:outline`). If the state modifier is placed first (e.g., `focus-visible:prose-a:outline`), the state variant is applied to the parent element (`.prose`) instead of the child element (`<a>`). Additionally, DaisyUI 4 removed `-focus` color modifier classes, so base color modifiers or opacities must be used for focus states instead.
**Action:** Always verify modifier order when applying interaction styles to typography children to ensure proper a11y focus states are visually rendering on the targeted element.

## 2026-06-21 - Dashboard Action Button Context
**Learning:** Action buttons in dense, utility-focused layouts like dashboards can lack context without labels or surrounding descriptions. Additionally, relying solely on custom CSS focus rings can result in inconsistent keyboard navigation experiences if not explicitly styled.
**Action:** When adding utility or action buttons (like Export or Share) to dashboard headers, wrap them in DaisyUI tooltip components (`<div class="tooltip" data-tip="...">`) to provide clear, immediate context to users. Always ensure these buttons also explicitly define `focus-visible` states matching the design system.

## 2026-06-22 - Dropdown Link Accessibility
**Learning:** Anchor tags (`<a>`) placed inside interactive components like DaisyUI dropdown menus are not focusable or selectable via keyboard navigation if they lack an `href` attribute.
**Action:** Always include an `href` attribute (even if it's just `href="#"` or a placeholder path) on any interactive anchor tag within navigation and dropdown components to ensure basic keyboard focusability.

</content>
</file>

<file path=".jules/sentinel.md">
<content>
## 2023-10-27 - [Prevent Path Traversal in Link Transformation]
**Vulnerability:** Arbitrary file write due to path traversal when generating missing file stubs.
**Learning:** The application parsed Markdown links and resolved relative paths to generate HTML stubs for missing files. However, it did not restrict paths to the output directory, allowing paths like `../../../tmp/hack.md` to break out and write `.html` files elsewhere on the system.
**Prevention:** Use `filepath.IsLocal` to validate all resolved relative paths before writing files or treating them as missing files, ensuring they do not escape the intended directory boundaries.

## 2023-10-28 - [Prevent XSS in Missing File Stubs]
**Vulnerability:** Cross-Site Scripting (XSS) in dynamically generated HTML for missing pages.
**Learning:** When generating HTML stubs for missing Markdown files, the filenames of the "parent" pages that linked to the missing page were injected directly into the HTML without sanitization. A maliciously crafted parent filename (e.g., `<script>alert(1)</script>.md`) could execute arbitrary JavaScript. Even seemingly benign data like filenames can act as XSS vectors if they are reflected into HTML.
**Prevention:** Always sanitize unsanitized or user-influenced strings, including filenames, using `html.EscapeString` before injecting them into HTML templates or string builders.

## 2026-06-21 - [Prevent XSS in Dynamically Generated HTML containing URLs]
**Vulnerability:** Potential Cross-Site Scripting (XSS) when generating HTML involving URLs dynamically.
**Learning:** `html.EscapeString` alone is insufficient when injecting user-influenced strings into HTML attributes like `href`, as it does not prevent schemes like `javascript:`.
**Prevention:** Always use a robust HTML sanitization library (like `bluemonday`'s `UGCPolicy()`) and apply `SanitizeBytes` to the entire generated HTML string before casting it to `template.HTML`, ensuring that both scripts and dangerous URL schemes are mitigated.
* **Layout Allowlist**: Added layout allowlist in render.go to validate frontmatter layout against templates/ directory.

</content>
</file>

<file path="internal/config/config.go">
<content>
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2" // Using v2 to match the indirect dependency from frontmatter
)

// Config represents the site configuration.
type SiteLink struct {
	Label string `yaml:"label"`
	URL   string `yaml:"url"`
}

// Config represents the site configuration.
type Config struct {
	SiteName           string     `yaml:"site_name"`
	Template           string     `yaml:"template"`
	ContentDir         string     `yaml:"content_dir"`
	OutputDir          string     `yaml:"output_dir"`
	AssetDir           string     `yaml:"asset_dir"`
	RagDir             string     `yaml:"rag_dir"`
	Theme              string     `yaml:"theme"`
	Port               int        `yaml:"port"`
	ProjectRoot        string     `yaml:"project_root"`
	DefaultDescription string     `yaml:"default_description"`
	DefaultOGImage     string     `yaml:"default_og_image"`
	WatchMode          bool       `yaml:"-"`
	SiteLinks          []SiteLink `yaml:"site_links"`
}

// DefaultConfig returns a Config with sensible default values.
func DefaultConfig() Config {
	return Config{
		SiteName:   "La Famille",
		Template:   "templates/layout.html",
		ContentDir: "content",
		OutputDir:  "public",
		AssetDir:   "assets",
		RagDir:     "rag-archive",
		Theme:      "retro",
		Port:       8080,
		ProjectRoot: ".",
	}
}

// Load reads a configuration file and parses it into a Config struct.
// If the file does not exist, it returns the DefaultConfig and no error.
func Load(filepath string) (Config, error) {
	config := DefaultConfig()

	data, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			// It's perfectly fine if the config file doesn't exist
			return config, nil
		}
		return config, err
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}

// WriteDefault writes the default configuration to the specified filepath.
func WriteDefault(filepath string) error {
	// We use text templates to preserve comments, rather than yaml.Marshal
	// which strips comments and ordering.

	defaultYaml := `# La Famille Site Configuration
#
# site_name: The name of your site, used in the navbar and footer.
site_name: "La Famille"

# template: The path to the HTML layout file used to render pages.
template: "templates/layout.html"

# content_dir: The directory containing your markdown source files.
content_dir: "content"

# output_dir: The directory where the generated HTML site will be placed.
output_dir: "public"

# asset_dir: The directory containing static assets.
asset_dir: "assets"

# rag_dir: The directory where RAG markdown bundles will be exported.
rag_dir: "rag-archive"

# theme: The DaisyUI theme applied to the site (e.g., retro, dark, cupcake, corporate).
theme: "retro"

# default_description: A default description for SEO meta tags.
# default_description: "A wonderful site built with La Famille"

# default_og_image: A default OpenGraph image URL.
# default_og_image: "/assets/default-og.png"

# site_links: Optional links for headers/footers
# site_links:
#   - label: "GitHub"
#     url: "https://github.com"
#   - label: "Twitter"
#     url: "https://twitter.com"

# port: The port on which the local development server will run.
port: 8080
`
	return os.WriteFile(filepath, []byte(defaultYaml), 0600)
}

// Validate checks that the configuration values are safe and correct.
func (c Config) Validate() error {
	if c.ContentDir == "" {
		return errors.New("ContentDir cannot be empty")
	}
	if c.OutputDir == "" {
		return errors.New("OutputDir cannot be empty")
	}
	if c.Template == "" {
		return errors.New("Template cannot be empty")
	}
	if c.AssetDir == "" {
		return errors.New("AssetDir cannot be empty")
	}
	if c.RagDir == "" {
		return errors.New("RagDir cannot be empty")
	}

	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("Port must be between 1 and 65535, got %d", c.Port)
	}

	// Validate path locality (prevent directory traversal)
	if !filepath.IsLocal(c.ContentDir) {
		return fmt.Errorf("ContentDir must be a local path, got %s", c.ContentDir)
	}
	if !filepath.IsLocal(c.OutputDir) {
		return fmt.Errorf("OutputDir must be a local path, got %s", c.OutputDir)
	}
	if !filepath.IsLocal(c.Template) {
		return fmt.Errorf("Template must be a local path, got %s", c.Template)
	}
	if !filepath.IsLocal(c.AssetDir) {
		return fmt.Errorf("AssetDir must be a local path, got %s", c.AssetDir)
	}
	if !filepath.IsLocal(c.RagDir) {
		return fmt.Errorf("RagDir must be a local path, got %s", c.RagDir)
	}

	return nil
}

</content>
</file>

<file path="internal/config/config_test.go">
<content>
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

</content>
</file>

<file path="assets/">
<content>
assets/
assets/img/
assets/img/1782325842-catfact.txt (size: 445 bytes)
assets/img/Octopus_mascot_Electric_Blue_body_202606200817.jpeg (size: 423240 bytes)
assets/img/Octopus_mascot_cleaning_litterbox_202606200817.jpeg (size: 351068 bytes)
assets/img/Octopus_mascot_riding_skateboard…_202606200817.jpeg (size: 316393 bytes)
assets/img/Octopus_mascot_writing_music_dia…_202606200817.jpeg (size: 355829 bytes)
assets/img/Octopus_riding_skateboard_holdin…_202606200817_2.jpeg (size: 331259 bytes)
assets/img/Octopus_riding_skateboard_musica…_202606200817_2.jpeg (size: 418984 bytes)
assets/img/jules-logo.png (size: 11886 bytes)
assets/img/mascot-40.jpeg (size: 1133524 bytes)
assets/img/mascot-brown-bag.jpeg (size: 1613481 bytes)
assets/img/mascot-default.jpeg (size: 1403430 bytes)
assets/img/mascot-electric-blue.jpeg (size: 423240 bytes)
assets/js/
assets/js/search.js (size: 5045 bytes)
assets/testdata/
assets/testdata/sites/
assets/testdata/sites/anchor-links/
assets/testdata/sites/anchor-links/content/
assets/testdata/sites/anchor-links/content/index.md (size: 137 bytes)
assets/testdata/sites/anchor-links/content/other.md (size: 97 bytes)
assets/testdata/sites/anchor-links/expected/
assets/testdata/sites/anchor-links/expected/pages/
assets/testdata/sites/anchor-links/expected/pages/backlinks.json (size: 76 bytes)
assets/testdata/sites/anchor-links/expected/pages/graph.json (size: 293 bytes)
assets/testdata/sites/anchor-links/expected/pages/meta.json (size: 131 bytes)
assets/testdata/sites/anchor-links/expected/pages/search.json (size: 238 bytes)
assets/testdata/sites/clean-urls/
assets/testdata/sites/clean-urls/content/
assets/testdata/sites/clean-urls/content/bio.md (size: 79 bytes)
assets/testdata/sites/clean-urls/content/blog/
assets/testdata/sites/clean-urls/content/blog/post.md (size: 62 bytes)
assets/testdata/sites/clean-urls/content/index.md (size: 102 bytes)
assets/testdata/sites/clean-urls/expected/
assets/testdata/sites/clean-urls/expected/pages/
assets/testdata/sites/clean-urls/expected/pages/backlinks.json (size: 110 bytes)
assets/testdata/sites/clean-urls/expected/pages/graph.json (size: 404 bytes)
assets/testdata/sites/clean-urls/expected/pages/meta.json (size: 195 bytes)
assets/testdata/sites/clean-urls/expected/pages/search.json (size: 268 bytes)
assets/testdata/sites/devlog/
assets/testdata/sites/devlog/config.yaml (size: 116 bytes)
assets/testdata/sites/devlog/content/
assets/testdata/sites/devlog/content/devlog/
assets/testdata/sites/devlog/content/devlog/added-search-feature.md (size: 633 bytes)
assets/testdata/sites/devlog/expected/
assets/testdata/sites/devlog/expected/pages/
assets/testdata/sites/devlog/expected/pages/backlinks.json (size: 3 bytes)
assets/testdata/sites/devlog/expected/pages/graph.json (size: 122 bytes)
assets/testdata/sites/devlog/expected/pages/meta.json (size: 153 bytes)
assets/testdata/sites/devlog/expected/pages/search.json (size: 227 bytes)
assets/testdata/sites/edge-cases/
assets/testdata/sites/edge-cases/content/
assets/testdata/sites/edge-cases/content/index.md (size: 223 bytes)
assets/testdata/sites/edge-cases/expected/
assets/testdata/sites/edge-cases/expected/pages/
assets/testdata/sites/edge-cases/expected/pages/backlinks.json (size: 3 bytes)
assets/testdata/sites/edge-cases/expected/pages/graph.json (size: 100 bytes)
assets/testdata/sites/edge-cases/expected/pages/meta.json (size: 67 bytes)
assets/testdata/sites/edge-cases/expected/pages/search.json (size: 216 bytes)
assets/testdata/sites/frontmatter/
assets/testdata/sites/frontmatter/content/
assets/testdata/sites/frontmatter/content/fallback-title.md (size: 37 bytes)
assets/testdata/sites/frontmatter/content/post.md (size: 84 bytes)
assets/testdata/sites/frontmatter/expected/
assets/testdata/sites/frontmatter/expected/pages/
assets/testdata/sites/frontmatter/expected/pages/backlinks.json (size: 3 bytes)
assets/testdata/sites/frontmatter/expected/pages/graph.json (size: 173 bytes)
assets/testdata/sites/frontmatter/expected/pages/meta.json (size: 261 bytes)
assets/testdata/sites/frontmatter/expected/pages/search.json (size: 169 bytes)
assets/testdata/sites/link-graph/
assets/testdata/sites/link-graph/content/
assets/testdata/sites/link-graph/content/a.md (size: 18 bytes)
assets/testdata/sites/link-graph/content/b.md (size: 10 bytes)
assets/testdata/sites/link-graph/content/c.md (size: 10 bytes)
assets/testdata/sites/link-graph/content/index.md (size: 36 bytes)
assets/testdata/sites/link-graph/expected/
assets/testdata/sites/link-graph/expected/pages/
assets/testdata/sites/link-graph/expected/pages/backlinks.json (size: 77 bytes)
assets/testdata/sites/link-graph/expected/pages/graph.json (size: 395 bytes)
assets/testdata/sites/link-graph/expected/pages/meta.json (size: 231 bytes)
assets/testdata/sites/link-graph/expected/pages/search.json (size: 246 bytes)
assets/testdata/sites/nested-dirs/
assets/testdata/sites/nested-dirs/content/
assets/testdata/sites/nested-dirs/content/about.md (size: 51 bytes)
assets/testdata/sites/nested-dirs/content/blog/
assets/testdata/sites/nested-dirs/content/blog/post.md (size: 63 bytes)
assets/testdata/sites/nested-dirs/content/index.md (size: 47 bytes)
assets/testdata/sites/nested-dirs/expected/
assets/testdata/sites/nested-dirs/expected/pages/
assets/testdata/sites/nested-dirs/expected/pages/backlinks.json (size: 125 bytes)
assets/testdata/sites/nested-dirs/expected/pages/graph.json (size: 575 bytes)
assets/testdata/sites/nested-dirs/expected/pages/meta.json (size: 196 bytes)
assets/testdata/sites/nested-dirs/expected/pages/search.json (size: 265 bytes)
assets/testdata/sites/query-fragments/
assets/testdata/sites/query-fragments/content/
assets/testdata/sites/query-fragments/content/index.md (size: 448 bytes)
assets/testdata/sites/query-fragments/content/page.md (size: 23 bytes)
assets/testdata/sites/query-fragments/expected/
assets/testdata/sites/query-fragments/expected/pages/
assets/testdata/sites/query-fragments/expected/pages/backlinks.json (size: 103 bytes)
assets/testdata/sites/query-fragments/expected/pages/graph.json (size: 520 bytes)
assets/testdata/sites/query-fragments/expected/pages/meta.json (size: 128 bytes)
assets/testdata/sites/query-fragments/expected/pages/search.json (size: 291 bytes)
assets/testdata/sites/render-modes/
assets/testdata/sites/render-modes/content/
assets/testdata/sites/render-modes/content/hidden.md (size: 45 bytes)
assets/testdata/sites/render-modes/content/index.md (size: 26 bytes)
assets/testdata/sites/render-modes/content/raw.md (size: 40 bytes)
assets/testdata/sites/render-modes/expected/
assets/testdata/sites/render-modes/expected/hidden.md (size: 45 bytes)
assets/testdata/sites/render-modes/expected/pages/
assets/testdata/sites/render-modes/expected/pages/backlinks.json (size: 3 bytes)
assets/testdata/sites/render-modes/expected/pages/graph.json (size: 231 bytes)
assets/testdata/sites/render-modes/expected/pages/hidden.md (size: 45 bytes)
assets/testdata/sites/render-modes/expected/pages/meta.json (size: 186 bytes)
assets/testdata/sites/render-modes/expected/pages/raw.md (size: 40 bytes)
assets/testdata/sites/render-modes/expected/pages/search.json (size: 54 bytes)
assets/testdata/sites/render-modes/expected/raw.md (size: 40 bytes)
assets/testdata/sites/simple-site/
assets/testdata/sites/simple-site/content/
assets/testdata/sites/simple-site/content/about.md (size: 36 bytes)
assets/testdata/sites/simple-site/content/index.md (size: 34 bytes)
assets/testdata/sites/simple-site/expected/
assets/testdata/sites/simple-site/expected/pages/
assets/testdata/sites/simple-site/expected/pages/backlinks.json (size: 3 bytes)
assets/testdata/sites/simple-site/expected/pages/graph.json (size: 165 bytes)
assets/testdata/sites/simple-site/expected/pages/meta.json (size: 122 bytes)
assets/testdata/sites/simple-site/expected/pages/search.json (size: 128 bytes)
assets/testdata/sites/simple-site/public/
assets/testdata/sites/simple-site/public/about/
assets/testdata/sites/simple-site/public/about/index.html (size: 0 bytes)
assets/vid/
assets/vid/video_202606200852.mp4 (size: 1113941 bytes)
</content>
</file>

<file path="templates/">
<content>
templates/
templates/brutalist.html (size: 10715 bytes)
templates/cyberpunk.html (size: 6511 bytes)
templates/devlog.html (size: 6706 bytes)
templates/layout-asymmetric.html (size: 7216 bytes)
templates/layout-bento.html (size: 4529 bytes)
templates/layout-centered-minimalist.html (size: 4448 bytes)
templates/layout-dashboard.html (size: 13398 bytes)
templates/layout-documentation.html (size: 4118 bytes)
templates/layout-drawer.html (size: 3957 bytes)
templates/layout-floating-cards.html (size: 7301 bytes)
templates/layout-glassmorphism.html (size: 3422 bytes)
templates/layout-hero.html (size: 4287 bytes)
templates/layout-magazine-grid.html (size: 5959 bytes)
templates/layout-neon.html (size: 6079 bytes)
templates/layout-sidebar.html (size: 5310 bytes)
templates/layout-split-screen.html (size: 4341 bytes)
templates/layout-terminal.html (size: 7050 bytes)
templates/layout-the-hacker.html (size: 14437 bytes)
templates/layout.html (size: 4690 bytes)
templates/luxury_magazine.html (size: 10287 bytes)
templates/partials/
templates/partials/footer-hacker.html (size: 1015 bytes)
</content>
</file>
