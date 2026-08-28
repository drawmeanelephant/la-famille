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
	Template           string `yaml:"template"`
	DefaultOGImage     string `yaml:"default_og_image"`
	ContentDir         string `yaml:"content_dir"`
	SiteName           string `yaml:"site_name"`
	AssetDir           string `yaml:"asset_dir"`
	RagDir             string `yaml:"rag_dir"`
	Theme              string `yaml:"theme"`
	ProjectRoot        string `yaml:"project_root"`
	SiteURL            string `yaml:"siteurl"`
	DefaultDescription string `yaml:"default_description"`
	OutputDir          string `yaml:"output_dir"`
	LegacySiteURL      string `yaml:"site_url"`
	// ConfigPath is populated by the CLI bootstrapper and is not part of the
	// site configuration or build fingerprint. It lets commands such as init
	// write to an explicitly selected configuration file while keeping the
	// public Config type useful to library callers.
	ConfigPath        string     `yaml:"-" json:"-"`
	SiteLinks         []SiteLink `yaml:"site_links"`
	Port              int        `yaml:"port"`
	MaxAssetSizeBytes int64      `yaml:"max_asset_size_bytes"`
	WatchMode         bool       `yaml:"-"`
	CheckAssetHealth  bool       `yaml:"check_asset_health"`
	GraphExplorer     bool       `yaml:"graph_explorer"`
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
	return WriteDefaultWithLayout(filepath, "templates/layout.html")
}

// WriteDefaultWithLayout writes the default configuration using layoutPath as
// the configured template path. It exists so `init --theme` can select a
// bundled layout as the site default while preserving the commented config.
func WriteDefaultWithLayout(filepath, layoutPath string) error {
	// We use text templates to preserve comments, rather than yaml.Marshal
	// which strips comments and ordering.

	defaultYaml := strings.Replace(defaultConfigYaml,
		`template: "templates/layout.html"`,
		`template: "`+layoutPath+`"`, 1)
	return os.WriteFile(filepath, []byte(defaultYaml), 0600)
}

const defaultConfigYaml = `# La Famille Site Configuration
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

# project_root: Optional root for all relative paths. The CLI --project-root
# flag takes precedence over this value.
# project_root: "."

# theme: The built-in palette applied by the default layout
# (retro, ink, sepia, slate, moss).
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
	// Both halves are decoded, so the combined result is escaped once at the
	// end: a raw space in a filename-derived slug must not reach search.json or
	// a canonical href unencoded (#531).
	base := ""
	if u, ok := c.publicURL(); ok {
		base = u.Path
	}
	return escapeURLPathSegments(base + publicPathForOutput(outputPath))
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

// publicPathForOutput returns the site-root-relative URL path for an output
// file, in decoded form. URLForOutputPath feeds this into a url.URL whose
// String method re-escapes it correctly; callers that emit a bare string (such
// as PublicPathForOutput or the discovery fallbacks) must escape separately so
// a decoded space or non-ASCII rune never reaches a machine-readable consumer
// raw (#531).
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

// escapeURLPathSegments percent-escapes every segment of a URL path without
// touching its / separators. A page whose filename contains a space or non-ASCII
// character (a hand-written `my post.md`, which low-level authors can still
// create directly) used to ship a <loc> in sitemap.xml with the raw space and an
// unencoded feed <link>: the sitemaps.org protocol requires URL-encoding, so
// search engines rejected the artifact (#531). Ordinary slug paths pass through
// byte for byte.
func escapeURLPathSegments(p string) string {
	segments := strings.Split(p, "/")
	for i, seg := range segments {
		if seg == "" {
			continue
		}
		segments[i] = url.PathEscape(seg)
	}
	return strings.Join(segments, "/")
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

// ResolvePaths makes all project-relative paths explicit. The CLI calls this
// after loading config.yaml so a binary launched from outside a project does
// not accidentally interpret content/, templates/, or public/ relative to
// the process CWD.
//
// Absolute paths are preserved. This is intentional for explicit overrides
// such as --asset-dir /tmp/site-assets; the source configuration file remains
// restricted to local relative paths by Validate.
func (c Config) ResolvePaths(projectRoot string) (Config, error) {
	root, err := filepath.Abs(filepath.Clean(projectRoot))
	if err != nil {
		return Config{}, fmt.Errorf("resolve project root %q: %w", projectRoot, err)
	}

	c.ProjectRoot = root
	c.ContentDir = resolvePath(root, c.ContentDir)
	c.OutputDir = resolvePath(root, c.OutputDir)
	c.AssetDir = resolvePath(root, c.AssetDir)
	c.RagDir = resolvePath(root, c.RagDir)
	c.Template = resolvePath(root, c.Template)
	return c, nil
}

func resolvePath(root, configured string) string {
	if filepath.IsAbs(configured) {
		return filepath.Clean(configured)
	}
	return filepath.Clean(filepath.Join(root, configured))
}

// Validate checks that a configuration file contains safe, local paths and
// valid values. Config files are deliberately not allowed to smuggle absolute
// paths into a build; callers that have resolved an explicit project-root may
// use ValidateResolved instead.
func (c Config) Validate() error {
	return c.validate(false)
}

// ValidateResolved validates a runtime configuration after ResolvePaths has
// made its paths absolute. It keeps the same value and output-isolation checks
// as Validate while allowing explicit paths selected by the operator.
func (c Config) ValidateResolved() error {
	return c.validate(true)
}

func (c Config) validate(allowAbsolutePaths bool) error {
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
		if !allowAbsolutePaths && !filepath.IsLocal(path) {
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
			if in.name == "RagDir" {
				// RAG archives may intentionally be staged below public for a
				// Pages artifact. The build does not read RagDir; a later `rag`
				// command repopulates that child after the static build swaps the
				// output directory into place.
				continue
			}
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
