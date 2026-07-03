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
	ProjectRoot        string     `yaml:"project_root"`
	DefaultDescription string     `yaml:"default_description"`
	DefaultOGImage     string     `yaml:"default_og_image"`
	SiteLinks          []SiteLink `yaml:"site_links"`
	Port               int        `yaml:"port"`
	WatchMode          bool       `yaml:"-"`
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
		CookieNotice: true,
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

# cookienotice: Whether to show the cookie consent toast banner.
cookienotice: true

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
