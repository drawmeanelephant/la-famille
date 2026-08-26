// Package publisher validates the directory that will be uploaded to a static
// host. It deliberately operates only on the generated tree, so CI can check
// an artifact without a source checkout or generator internals.
package publisher

import (
	"bytes"
	"encoding/json"
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

// stagingDirPrefix marks temporary atomic-build directories. A correct build
// cleans them up beside the project root; one inside the publish artifact
// means a partial or interrupted build was captured.
const stagingDirPrefix = ".staging-"

// coreArtifacts are written on every build regardless of configuration, so a
// generated publish artifact without them is incomplete.
var coreArtifacts = []string{
	"sitemap.xml",
	"robots.txt",
	"search.json",
	"graph.json",
	"backlinks.json",
	"meta.json",
}

// Manifest is a deterministic list of files in a publish artifact.
type Manifest struct {
	Files []string `json:"files"`
	// Stubs lists generated "Missing Page" placeholder files (#516). They are
	// reported separately from validation failures: shipping one means an
	// author typo'd an internal link, which deserves a signal before deploy
	// but is not a malformed artifact.
	Stubs []string `json:"stubs,omitempty"`
}

// ValidationError reports every concrete problem found in an artifact so
// callers can render structured output instead of parsing prose (#510).
type ValidationError struct {
	Problems []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("publish artifact validation failed:\n- %s", strings.Join(e.Problems, "\n- "))
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
			if strings.HasPrefix(entry.Name(), stagingDirPrefix) {
				rel, err := filepath.Rel(root, filePath)
				if err != nil {
					return err
				}
				return fmt.Errorf("publish artifact contains temporary staging directory %q; a complete build output has no %s* leftovers", filepath.ToSlash(rel), stagingDirPrefix)
			}
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
			if isStubPage(data) {
				manifest.Stubs = append(manifest.Stubs, rel)
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

	for _, required := range coreArtifacts {
		if _, ok := files[required]; !ok {
			problems = append(problems, fmt.Sprintf("publish artifact is missing required file %q", required))
		}
	}

	// feed.xml is legitimately absent when no rendered page carries a date
	// (the generator deletes stale feeds), so its requirement is derived from
	// meta.json inside the artifact itself rather than assumed.
	if hasDatedRenderedPage(root) {
		if _, ok := files["feed.xml"]; !ok {
			problems = append(problems, `publish artifact is missing required file "feed.xml": meta.json lists a dated rendered page`)
		}
	}

	if _, ok := files["graph/index.html"]; ok {
		for _, required := range []string{"graph/data.json", "assets/graph/explorer.css", "assets/graph/explorer.js"} {
			if _, exists := files[required]; !exists {
				problems = append(problems, fmt.Sprintf("graph explorer is missing required artifact %q", required))
			}
		}
	}

	sort.Strings(manifest.Stubs)

	if len(problems) > 0 {
		return manifest, &ValidationError{Problems: problems}
	}
	return manifest, nil
}

// stubTitleRe matches the <title> a Missing Page stub is rendered with. The
// body marker below is written by the stub generator itself, so requiring both
// keeps ordinary pages that happen to be titled "Missing Page" out of the list.
var stubTitleRe = regexp.MustCompile(`(?i)<title>\s*Missing Page\b`)

const stubBodyMarker = "Under Construction"

func isStubPage(data []byte) bool {
	return stubTitleRe.Match(data) && bytes.Contains(data, []byte(stubBodyMarker))
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

// hasDatedRenderedPage reports whether meta.json inside the artifact lists at
// least one rendered page with a non-empty date — the exact condition under
// which the generator emits feed.xml. Unreadable or missing metadata returns
// false; a missing meta.json is reported separately as a required-file error.
func hasDatedRenderedPage(root string) bool {
	data, err := os.ReadFile(filepath.Join(root, "meta.json"))
	if err != nil {
		return false
	}
	var metaData map[string]struct {
		Date   string `json:"date"`
		Render bool   `json:"render"`
	}
	if err := json.Unmarshal(data, &metaData); err != nil {
		return false
	}
	for _, meta := range metaData {
		if meta.Render && strings.TrimSpace(meta.Date) != "" {
			return true
		}
	}
	return false
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
	} else if path.Ext(candidate) == ".html" {
		// The generator may emit content/foo.md as content/foo/index.html.
		// When a link says /foo.html, also accept /foo/index.html.
		dirCandidate := candidate[:len(candidate)-len(".html")] + "/index.html"
		if _, ok := files[dirCandidate]; ok {
			return dirCandidate, true
		}
	}
	_, ok := files[candidate]
	return candidate, ok
}
