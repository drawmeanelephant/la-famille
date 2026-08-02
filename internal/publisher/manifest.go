// Package publisher validates the directory that will be uploaded to a static
// host. It deliberately operates only on the generated tree, so CI can check
// an artifact without a source checkout or generator internals.
package publisher

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// CacheFileName is internal generator state and is never part of the public
// artifact. Keeping the name in one package-level contract avoids naming drift
// between the generator and publisher checks.
const CacheFileName = ".la-famille-cache.json"

// Manifest is a deterministic list of files in a publish artifact.
type Manifest struct {
	Files []string `json:"files"`
}

var htmlReference = regexp.MustCompile(`(?is)(?:href|src)\s*=\s*["']([^"']+)["']`)

// Check returns the complete output manifest and verifies that local HTML
// references resolve within it. The cache policy is strict: a cache accidentally
// copied into public is an error rather than a silently exposed implementation
// detail.
func Check(outputDir string) (Manifest, error) {
	root, err := filepath.Abs(outputDir)
	if err != nil {
		return Manifest{}, fmt.Errorf("resolve publish directory: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return Manifest{}, fmt.Errorf("inspect publish directory: %w", err)
	}
	if !info.IsDir() {
		return Manifest{}, fmt.Errorf("publish path is not a directory: %s", outputDir)
	}

	var manifest Manifest
	err = filepath.WalkDir(root, func(filePath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("publish artifact contains a symlink: %s", filePath)
		}
		rel, err := filepath.Rel(root, filePath)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == CacheFileName {
			return fmt.Errorf("publish artifact contains internal build cache %q; cache state belongs beside the project, not in public", CacheFileName)
		}
		manifest.Files = append(manifest.Files, rel)
		return nil
	})
	if err != nil {
		return Manifest{}, err
	}
	sort.Strings(manifest.Files)

	files := make(map[string]struct{}, len(manifest.Files))
	for _, file := range manifest.Files {
		files[file] = struct{}{}
	}

	var problems []string
	for _, rel := range manifest.Files {
		if strings.EqualFold(filepath.Ext(rel), ".html") {
			data, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
			if readErr != nil {
				problems = append(problems, fmt.Sprintf("read %s: %v", rel, readErr))
				continue
			}
			for _, match := range htmlReference.FindAllStringSubmatch(string(data), -1) {
				reference := strings.TrimSpace(match[1])
				if reference == "" || isExternalReference(reference) {
					continue
				}
				if target, ok := resolveReference(rel, reference, files); !ok {
					problems = append(problems, fmt.Sprintf("%s references missing local file %q", rel, reference))
				} else if target == "" {
					problems = append(problems, fmt.Sprintf("%s references invalid local file %q", rel, reference))
				}
			}
		}
	}

	if _, ok := files["graph/index.html"]; ok {
		for _, required := range []string{"graph/data.json", "assets/graph/explorer.css", "assets/graph/explorer.js"} {
			if _, exists := files[required]; !exists {
				problems = append(problems, fmt.Sprintf("graph explorer is missing required artifact %q", required))
			}
		}
	}

	if len(problems) > 0 {
		return manifest, fmt.Errorf("publish artifact validation failed:\n- %s", strings.Join(problems, "\n- "))
	}
	return manifest, nil
}

func isExternalReference(reference string) bool {
	if strings.HasPrefix(reference, "#") || strings.HasPrefix(reference, "//") {
		return true
	}
	u, err := url.Parse(reference)
	if err != nil {
		return false
	}
	return u.IsAbs() || u.Host != ""
}

func resolveReference(from, reference string, files map[string]struct{}) (string, bool) {
	u, err := url.Parse(reference)
	if err != nil {
		return "", false
	}
	name, err := url.PathUnescape(u.Path)
	if err != nil {
		return "", false
	}
	if name == "" {
		name = "."
	}

	var candidate string
	if strings.HasPrefix(name, "/") {
		candidate = path.Clean(strings.TrimPrefix(name, "/"))
	} else {
		candidate = path.Clean(path.Join(path.Dir(from), name))
	}
	if candidate == "." {
		candidate = "index.html"
	}
	if candidate == ".." || strings.HasPrefix(candidate, "../") {
		return "", false
	}

	if _, ok := files[candidate]; ok {
		return candidate, true
	}
	if strings.HasSuffix(candidate, "/") {
		candidate = path.Join(candidate, "index.html")
	} else if path.Ext(candidate) == "" {
		candidate = path.Join(candidate, "index.html")
	}
	_, ok := files[candidate]
	return candidate, ok
}
