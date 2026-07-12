package transform

import (
	"path"
	"strings"
)

// GetOutputURL calculates the output URL (with index.html) for a given .md relative path and optional slug override.
func GetOutputURL(relPath string, slug string) string {
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

	if slug != "" {
		name = slug
	}


	return path.Join(dir, name, "index.html")
}
