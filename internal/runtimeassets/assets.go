package runtimeassets

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	siteassets "github.com/tbuddy/la-famille/assets"
	templateassets "github.com/tbuddy/la-famille/templates"
)

// DefaultTemplatePath is the path used by the default configuration.
const DefaultTemplatePath = "templates/layout.html"

// DefaultTemplate returns the release-owned default layout.
func DefaultTemplate() ([]byte, error) {
	return fs.ReadFile(templateassets.FS, "layout.html")
}

// CuratedTheme pairs a bundled layout name with the one-line description
// shown by the `themes` command and the unknown-theme error.
type CuratedTheme struct {
	Name        string
	Description string
}

// curatedThemes is the release theme packet in display order. Octoburger is
// the flagship: the octoburger TUI identity translated to a site layout.
// Terminal rounds out the packet with an existing self-contained look.
// Editorial and midnight are fully framework-free: system font stacks, no
// CDN requests at all.
var curatedThemes = []CuratedTheme{
	{Name: "layout", Description: "the default La Famille look: clean, fast, content-first"},
	{Name: "layout-octoburger", Description: "flagship soul theme; Raoul(s) the octopus holds the burger while you write"},
	{Name: "layout-terminal", Description: "retro terminal console with a synthwave glow, self-contained"},
	{Name: "layout-editorial", Description: "serif gazette with masthead and drop caps, framework-free and offline"},
	{Name: "layout-midnight", Description: "restrained dark theme for technical writing, framework-free and offline"},
}

// CuratedLayoutNames lists the layouts shipped in the release theme packet.
var CuratedLayoutNames = curatedLayoutNames()

func curatedLayoutNames() []string {
	names := make([]string, 0, len(curatedThemes))
	for _, theme := range curatedThemes {
		names = append(names, theme.Name)
	}
	return names
}

// CuratedThemes returns the bundled theme packet with one-line descriptions,
// ready for user-facing discovery output.
func CuratedThemes() []CuratedTheme {
	return append([]CuratedTheme(nil), curatedThemes...)
}

// CuratedLayouts returns the bundled theme packet keyed by layout name, ready
// for InstallMissing into a project's templates directory. Names match the
// frontmatter `layout:` values accepted by the renderer.
func CuratedLayouts() (map[string][]byte, error) {
	layouts := make(map[string][]byte, len(CuratedLayoutNames))
	for _, name := range CuratedLayoutNames {
		data, err := fs.ReadFile(templateassets.FS, name+".html")
		if err != nil {
			return nil, fmt.Errorf("read embedded layout %s: %w", name, err)
		}
		layouts[name] = data
	}
	return layouts, nil
}

// DefaultPartials returns the partials required by the default layout keyed by
// their path relative to the template directory.
func DefaultPartials() (map[string][]byte, error) {
	data, err := fs.ReadFile(templateassets.FS, "partials/search_modal.html")
	if err != nil {
		return nil, err
	}
	return map[string][]byte{"partials/search_modal.html": data}, nil
}

// DefaultAssetFiles returns only the runtime-owned files. It intentionally
// does not embed the site's image/video/testdata tree.
func DefaultAssetFiles() (map[string][]byte, error) {
	paths := []string{
		"graph/explorer.css",
		"graph/explorer.js",
		"css/theme-foundations.css",
		"css/theme.css",
		"css/layout-editorial.css",
		"css/layout-midnight.css",
		"css/search.css",
		"js/search.js",
		"img/mascot-default.jpeg",
		"img/jules-logo.png",
	}
	files := make(map[string][]byte, len(paths))
	for _, name := range paths {
		data, err := fs.ReadFile(siteassets.FS, name)
		if err != nil {
			return nil, fmt.Errorf("read embedded asset %s: %w", name, err)
		}
		files[name] = data
	}
	return files, nil
}

// InstallMissing writes release-owned defaults into a project. Existing files
// are left untouched so an operator's explicit site assets remain authoritative.
// The operation is safe to call repeatedly.
func InstallMissing(root string, files map[string][]byte, mode fs.FileMode) error {
	names := make([]string, 0, len(files))
	for rel := range files {
		names = append(names, rel)
	}
	sort.Strings(names)
	for _, rel := range names {
		data := files[rel]
		target := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("create directory for %s: %w", target, err)
		}

		file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
		if err != nil {
			if os.IsExist(err) {
				continue
			}
			return fmt.Errorf("create %s: %w", target, err)
		}
		if _, writeErr := file.Write(data); writeErr != nil {
			_ = file.Close()
			return fmt.Errorf("write %s: %w", target, writeErr)
		}
		if closeErr := file.Close(); closeErr != nil {
			return fmt.Errorf("close %s: %w", target, closeErr)
		}
	}
	return nil
}
