package config

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

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
	SiteURL            string     `yaml:"siteurl"`
	LegacySiteURL      string     `yaml:"site_url"`
	SiteLinks          []SiteLink `yaml:"site_links"`
	Port               int        `yaml:"port"`
	WatchMode          bool       `yaml:"-"`
	CheckAssetHealth   bool       `yaml:"check_asset_health"`
	MaxAssetSizeBytes  int64      `yaml:"max_asset_size_bytes"`
	GraphExplorer      bool       `yaml:"graph_explorer"`
}

// DefaultConfig returns a Config with sensible default values.
func DefaultConfig() Config {
	return Config{
		SiteName:          "La Famille",
		Template:          "templates/layout.html",
		ContentDir:        "content",
		OutputDir:         "public",
		AssetDir:          "assets",
		RagDir:            "rag-archive",
		Theme:             "retro",
		Port:              8080,
		ProjectRoot:       ".",
		CheckAssetHealth:  false,
		MaxAssetSizeBytes: 5 * 1024 * 1024,
		GraphExplorer:     true,
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

	if config.SiteURL == "" {
		config.SiteURL = config.LegacySiteURL
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

# siteurl: The public base URL used for canonical links, og:url, and discovery files.
# siteurl: "https://example.github.io/my-site"

# site_links: Optional links for headers/footers
# site_links:
#   - label: "GitHub"
#     url: "https://github.com"
#   - label: "Twitter"
#     url: "https://twitter.com"

# port: The port on which the local development server will run.
port: 8080

# graph_explorer: Controls generation of the interactive Knowledge Graph page
# at /graph/index.html. Defaults to true; set to false to skip emission (no
# /graph/ output, no nav link).
# graph_explorer: true
`
	return os.WriteFile(filepath, []byte(defaultYaml), 0600)
}

// URLForOutputPath returns the canonical public URL for an output path. An
// unavailable or invalid SiteURL intentionally produces an empty result so
// local builds do not emit malformed absolute URLs.
func (c Config) URLForOutputPath(outputPath string) string {
	base, ok := c.publicURL()
	if !ok {
		return ""
	}
	publicPath := publicPathForOutput(outputPath)
	base.Path = strings.TrimRight(base.Path, "/") + publicPath
	return base.String()
}

// PublicPathForOutput returns the site-root-relative URL path for a generated
// output file, including the base path when siteurl points at a subdirectory
// (for example a GitHub Pages project site at https://user.github.io/project).
//
// Unlike URLForOutputPath this never returns an absolute URL and does not
// require siteurl to be set, so it is safe for links that must work in both
// local and published builds.
func (c Config) PublicPathForOutput(outputPath string) string {
	base := ""
	if u, ok := c.publicURL(); ok {
		base = u.Path
	}
	return base + publicPathForOutput(outputPath)
}

// BasePath returns the URL path prefix the site is served under, derived from
// siteurl. It is empty when siteurl is unset or the site is served from the
// root of its host.
func (c Config) BasePath() string {
	u, ok := c.publicURL()
	if !ok {
		return ""
	}
	return u.Path
}

func (c Config) publicURL() (*url.URL, bool) {
	siteURL := c.SiteURL
	if strings.TrimSpace(siteURL) == "" {
		siteURL = c.LegacySiteURL
	}
	if strings.TrimSpace(siteURL) == "" {
		return nil, false
	}
	u, err := url.Parse(strings.TrimSpace(siteURL))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, false
	}
	for _, segment := range strings.Split(u.EscapedPath(), "/") {
		if segment == ".." || segment == "." || strings.Contains(strings.ToLower(segment), "%2e") {
			return nil, false
		}
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawPath = ""
	return u, true
}

func publicPathForOutput(outputPath string) string {
	outputPath = strings.TrimPrefix(filepath.ToSlash(outputPath), "/")
	if outputPath == "index.html" {
		return "/"
	}
	if strings.HasSuffix(outputPath, "/index.html") {
		return "/" + strings.TrimSuffix(outputPath, "index.html")
	}
	return "/" + path.Clean(outputPath)
}

// ValidateSiteURL checks that SiteURL (or LegacySiteURL), if set, is a valid absolute HTTP or HTTPS URL.
func (c Config) ValidateSiteURL() error {
	if strings.TrimSpace(c.SiteURL) != "" || strings.TrimSpace(c.LegacySiteURL) != "" {
		if _, ok := c.publicURL(); !ok {
			return fmt.Errorf("SiteURL must be an absolute HTTP or HTTPS URL without query, fragment, userinfo, or traversal")
		}
	}
	return nil
}

// Validate checks that the configuration values are safe and correct.
func (c Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("Port must be between 1 and 65535, got %d", c.Port)
	}
	if err := c.ValidateSiteURL(); err != nil {
		return err
	}

	dirs := []struct{ name, path string }{
		{"ContentDir", c.ContentDir},
		{"OutputDir", c.OutputDir},
		{"Template", c.Template},
		{"AssetDir", c.AssetDir},
		{"RagDir", c.RagDir},
		{"ProjectRoot", c.ProjectRoot},
	}

	for _, d := range dirs {
		name, path := d.name, d.path
		if path == "" {
			return fmt.Errorf("%s cannot be empty", name)
		}
		if !filepath.IsLocal(path) {
			return fmt.Errorf("%s must be a local path, got %s", name, path)
		}
	}

	return nil
}
