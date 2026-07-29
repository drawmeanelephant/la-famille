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
	DefaultDescription string     `yaml:"default_description"`
	SiteURL            string     `yaml:"siteurl"`
	ContentDir         string     `yaml:"content_dir"`
	OutputDir          string     `yaml:"output_dir"`
	AssetDir           string     `yaml:"asset_dir"`
	RagDir             string     `yaml:"rag_dir"`
	Theme              string     `yaml:"theme"`
	ProjectRoot        string     `yaml:"project_root"`
	Template           string     `yaml:"template"`
	SiteName           string     `yaml:"site_name"`
	DefaultOGImage     string     `yaml:"default_og_image"`
	LegacySiteURL      string     `yaml:"site_url"`
	SiteLinks          []SiteLink `yaml:"site_links"`
	Port               int        `yaml:"port"`
	MaxAssetSizeBytes  int64      `yaml:"max_asset_size_bytes"`
	WatchMode          bool       `yaml:"-"`
	CheckAssetHealth   bool       `yaml:"check_asset_health"`
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
//
// Contract: whenever Load returns a non-nil error it returns the zero Config,
// never a usable one. A config file that exists but cannot be read or parsed
// is not the same as no config file at all: falling back to defaults there
// silently drops settings the operator did supply (siteurl above all), and
// returning the struct yaml.Unmarshal half-filled in mixes file values with
// defaults. Callers must treat (Config{}, err) as "there is no configuration"
// and refuse to do configured work, rather than warning and carrying on.
func Load(filepath string) (Config, error) {
	config := DefaultConfig()

	data, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			// It's perfectly fine if the config file doesn't exist
			return config, nil
		}
		return Config{}, err
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		// yaml.v2 applies every key it read before the one that failed, so
		// config is now a mix of file values and defaults. Discard it.
		return Config{}, err
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
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		// ForceQuery is set for a URL ending in a bare "?"; RawQuery stays
		// empty for it, but (*url.URL).String re-emits the "?" into every
		// canonical URL we build from it.
		return nil, false
	}
	// Both spellings of the path have to be checked. Encoded dots (%2e%2e)
	// only survive in the escaped form, while an encoded separator (..%2F)
	// leaves the escaped form one opaque segment and only splits apart in the
	// decoded form, which is the one URLForOutputPath actually emits.
	for _, candidate := range []string{u.EscapedPath(), u.Path} {
		for _, segment := range strings.Split(candidate, "/") {
			if segment == ".." || segment == "." || strings.Contains(strings.ToLower(segment), "%2e") {
				return nil, false
			}
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

	return c.validateOutputIsolation()
}

// validateOutputIsolation rejects an output directory that overlaps an input.
//
// A successful build renames the output directory aside, installs the staged
// tree in its place and deletes the backup. That is safe only while the output
// directory holds nothing but generated files: point it at the content
// directory and a single build deletes every source file, reporting success.
// The overlap is the thing to catch, because by the time the swap runs the
// source is already gone.
func (c Config) validateOutputIsolation() error {
	output := canonicalDir(c.OutputDir)
	if output == "" {
		return nil
	}

	// Checked before the individual inputs: building over the project root
	// would replace everything, and saying so plainly beats reporting whichever
	// input happened to be compared first.
	if root := canonicalDir(c.ProjectRoot); root != "" {
		if root == output {
			return fmt.Errorf("OutputDir (%s) is the project root; a build would replace the entire project directory", c.OutputDir)
		}
		if isWithin(output, root) {
			return fmt.Errorf("ProjectRoot (%s) is inside OutputDir (%s); a build would delete it when it replaces the output directory", c.ProjectRoot, c.OutputDir)
		}
	}

	inputs := []struct{ name, path string }{
		{"ContentDir", c.ContentDir},
		{"AssetDir", c.AssetDir},
		{"the template directory", filepath.Dir(c.Template)},
		{"RagDir", c.RagDir},
	}

	root := canonicalDir(c.ProjectRoot)

	for _, in := range inputs {
		other := canonicalDir(in.path)
		if other == "" {
			continue
		}
		// An input that IS the project root contains the output directory in
		// every ordinary layout, exactly as ProjectRoot itself does: a flat
		// site with content_dir "." and its markdown at the repo root, or a
		// single layout.html beside config.yaml, whose template directory is
		// therefore ".". Applying the "output inside an input" rule to those
		// rejected working sites outright.
		if other == root {
			if other == output {
				return fmt.Errorf("OutputDir (%s) is the same directory as %s (%s); a build would replace it and delete its contents", c.OutputDir, in.name, in.path)
			}
			continue
		}
		switch {
		case other == output:
			return fmt.Errorf("OutputDir (%s) is the same directory as %s (%s); a build would replace it and delete its contents", c.OutputDir, in.name, in.path)
		case isWithin(output, other):
			// The input lives inside the output directory, so the swap takes
			// it with the rest of the replaced tree.
			return fmt.Errorf("%s (%s) is inside OutputDir (%s); a build would delete it when it replaces the output directory", in.name, in.path, c.OutputDir)
		case isWithin(other, output):
			// The output lives inside an input. The swap only replaces the
			// output subtree, so nothing is deleted, but generated files land
			// among the sources and the next build reads them back as input.
			return fmt.Errorf("OutputDir (%s) is inside %s (%s); generated files would be written among your sources", c.OutputDir, in.name, in.path)
		}
	}

	// Note ProjectRoot deliberately gets no "output inside input" rule: it
	// contains the output directory in every ordinary layout (public/ inside
	// .), so that check would reject the default configuration.
	return nil
}

// canonicalDir resolves a configured directory for comparison. Symlinks are
// resolved where the path exists, so two names for one directory are not
// mistaken for two directories; a path that does not exist yet is still
// compared lexically rather than skipped.
func canonicalDir(dir string) string {
	if strings.TrimSpace(dir) == "" {
		return ""
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return filepath.Clean(dir)
	}
	if resolved, resolveErr := filepath.EvalSymlinks(abs); resolveErr == nil {
		return resolved
	}
	return abs
}

// isWithin reports whether target sits inside base. Equal paths are not
// "within" — the caller reports that case separately.
func isWithin(base, target string) bool {
	if base == "" || target == "" || base == target {
		return false
	}
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}
