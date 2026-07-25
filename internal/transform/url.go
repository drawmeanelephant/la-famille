package transform

import (
	"path"
	"path/filepath"
	"strings"
)

// IsUsableSlug reports whether a frontmatter slug can name an output
// directory. A slug that fails this is ignored and the page keeps its
// filename-derived path, so every consumer predicting a URL has to apply the
// same rule — otherwise it links to a directory the renderer never wrote.
//
// This is the single definition of that rule. GetOutputURL applies it, so
// callers may pass a raw frontmatter slug without checking it first.
func IsUsableSlug(slug string) bool {
	return filepath.IsLocal(slug) &&
		!strings.Contains(slug, ".") &&
		!strings.Contains(slug, string(filepath.Separator)) &&
		!strings.Contains(slug, "/")
}

// GetOutputURL calculates the output URL (with index.html) for a given .md relative path and optional slug override.
func GetOutputURL(relPath string, slug string, render bool) string {
	if !render {
		return relPath
	}
	dir := path.Dir(relPath)
	if dir == "." {
		dir = ""
	}

	if relPath == "index.md" || path.Base(relPath) == "index.md" {
		if dir == "" {
			return "index.html"
		}
		return path.Join(dir, "index.html")
	}

	base := path.Base(relPath)
	name := strings.TrimSuffix(base, ".md")

	// An unusable slug is ignored here rather than trusted, so a caller that
	// passes frontmatter through verbatim still predicts the path the renderer
	// actually writes.
	if slug != "" && IsUsableSlug(slug) {
		name = slug
	}

	return path.Join(dir, name, "index.html")
}
