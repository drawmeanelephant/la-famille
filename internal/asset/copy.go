package asset

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/pathutil"
	"github.com/tbuddy/la-famille/internal/runtimeassets"
	"golang.org/x/net/html"
)

// graphAssetDir is the asset subdirectory holding the knowledge graph
// explorer's bundle. It mirrors the path in internal/graphexplorer.AssetRel.
const graphAssetDir = "graph"

// resolveDir returns dir with symlinks resolved, falling back to the cleaned
// path when it cannot be resolved (it may not exist yet). Paths that are
// compared with or relativized against each other must all pass through this,
// or a platform where a parent directory is itself a symlink will produce
// paths that never match.
func resolveDir(dir string) string {
	if dir == "" {
		return ""
	}
	// Absolute first: EvalSymlinks keeps a relative path relative, and a
	// relative result later made absolute against the working directory would
	// not have its parents resolved — so a resolved path and an unresolved one
	// get compared and never match.
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = filepath.Clean(dir)
	}
	if resolved, resolveErr := filepath.EvalSymlinks(abs); resolveErr == nil {
		return resolved
	}
	return abs
}

// ClaimOutput reserves an output-relative path for the asset copier, returning
// a non-nil error when another producer already owns it. The caller builds the
// message, so an asset collision reads exactly like a page collision. A nil
// ClaimOutput disables ownership tracking, which is what the package's own
// tests want; the generator always supplies one.
type ClaimOutput func(relOut string) error

func CopyAssets(cfg config.Config, claim ClaimOutput) error {
	var ignoreRules []IgnoreRule
	if cfg.ProjectRoot != "" {
		ignoreRules = LoadIgnoreRules(cfg.ProjectRoot)
	}

	assetRootExists := cfg.AssetDir != ""
	if assetRootExists {
		info, err := os.Stat(cfg.AssetDir)
		if err != nil {
			if os.IsNotExist(err) {
				assetRootExists = false
			} else {
				return err
			}
		} else if !info.IsDir() {
			// Without this the walk treats the file as its own root and the
			// copy fails on the staging destination instead — blaming the
			// internal .staging path rather than the configured dir (#642).
			return fmt.Errorf("asset_dir %q is not a directory", cfg.AssetDir)
		}
	}

	outDirClean := filepath.Clean(filepath.Join(cfg.OutputDir, "assets"))
	if err := os.MkdirAll(outDirClean, 0755); err != nil {
		return err
	}

	// WalkDir lstats its root, so a symlinked asset directory matches the
	// inner-symlink skip below on the very first callback and the walk ends
	// having copied nothing — a silently unstyled site. Resolve the root once
	// so a symlinked asset directory behaves like a real one. Symlinks *inside*
	// the tree are still skipped.
	//
	// Every path compared against the walk root, or made relative to it, has to
	// be resolved the same way: on macOS /var is itself a symlink to
	// /private/var, so mixing resolved and unresolved paths silently breaks
	// both the containment check below and the ignore rules.
	walkRoot := resolveDir(cfg.AssetDir)
	outputResolved := resolveDir(cfg.OutputDir)
	projectRootResolved := cfg.ProjectRoot
	if projectRootResolved != "" {
		projectRootResolved = resolveDir(projectRootResolved)
	}

	// Refuse a resolved root that contains the output directory — `assets`
	// pointing at the project root, say. Walking it would copy the previous
	// build, and everything sitting beside it, into the new one and grow
	// without bound. Failing here beats publishing the whole project.
	if assetRootExists && pathutil.IsSafePath(walkRoot, outputResolved) {
		return fmt.Errorf("asset directory %q resolves to %q, which contains the output directory; copying it would publish the output into itself", cfg.AssetDir, walkRoot)
	}

	// Only bundled theme CSS/images are selective. init installs the whole
	// packet into a site's assets directory, so detect those byte-identical
	// copies as well as the embedded fallbacks. Edited site assets and all
	// non-theme files continue to be published unconditionally.
	files, err := runtimeassets.DefaultAssetFiles()
	if err != nil {
		return err
	}
	themeAssets := runtimeassets.ThemeAssetNames()
	needed := make(map[string]bool, len(themeAssets))
	if !cfg.IncludeUnusedThemeAssets {
		needed, err = referencedThemeAssets(outDirClean, walkRoot, assetRootExists, projectRootResolved, ignoreRules, themeAssets, files)
		if err != nil {
			return err
		}
	}
	publishThemeAsset := func(rel string) bool {
		return cfg.IncludeUnusedThemeAssets || needed[rel]
	}
	isThemeAsset := make(map[string]bool, len(themeAssets))
	for _, rel := range themeAssets {
		isThemeAsset[rel] = true
	}

	if assetRootExists {
		if err := filepath.WalkDir(walkRoot, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// Belt and braces: never descend into the output directory even if the
			// containment check above was somehow satisfied.
			if d.IsDir() && path == outputResolved {
				return filepath.SkipDir
			}

			if d.Type()&os.ModeSymlink != 0 {
				slog.Warn("Skipping symlink in assets", "path", path)
				return nil
			}
			relPath, err := filepath.Rel(walkRoot, path)
			if err != nil {
				return err
			}

			relSlash := filepath.ToSlash(relPath)

			// The knowledge graph explorer's bundle is only reachable from the
			// explorer page. When graph_explorer is off that page is never
			// generated, so copying the bundle would ship dead CSS and JS.
			if !cfg.GraphExplorer && (relSlash == graphAssetDir || strings.HasPrefix(relSlash, graphAssetDir+"/")) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// projectRootResolved, not cfg.ProjectRoot: walk paths are resolved, and
			// mixing the two makes every path non-local, which silently turns
			// .gitignore filtering into a no-op and publishes ignored files.
			if IsIgnoredAsset(path, d.IsDir(), relSlash, projectRootResolved, ignoreRules) {
				return nil
			}

			if d.IsDir() {
				return nil
			}

			if isThemeAsset[relSlash] && !publishThemeAsset(relSlash) {
				source, readErr := os.ReadFile(path)
				if readErr != nil {
					return readErr
				}
				if bytes.Equal(source, files[relSlash]) {
					return nil
				}
			}

			destPath := filepath.Join(outDirClean, filepath.FromSlash(relPath))
			if !pathutil.IsSafePath(outDirClean, destPath) {
				slog.Warn("Static asset sync boundary intervention blocked layout breakout", "path", relPath)
				return nil
			}

			// Assets are copied after the pages are rendered, so without an
			// ownership check an asset silently replaces a page that renders to the
			// same path — the site publishes the asset bytes while search, graph and
			// meta all still describe the page.
			if claim != nil {
				if claimErr := claim("assets/" + relSlash); claimErr != nil {
					return claimErr
				}
			}

			// Ensure directory structure is built first
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return err
			}

			srcStat, err := d.Info()
			if err != nil {
				return err
			}

			destStat, err := os.Stat(destPath)
			if err == nil {
				if srcStat.Size() == destStat.Size() && srcStat.ModTime().Equal(destStat.ModTime()) {
					return nil
				}
			}

			if err := CopyFile(path, destPath); err != nil {
				return err
			}

			return os.Chtimes(destPath, srcStat.ModTime(), srcStat.ModTime())
		}); err != nil {
			return err
		}
	}

	// Released binaries own a small fallback bundle so a fresh init/build does
	// not require an operator to discover repository assets. User-owned files
	// were copied first and remain authoritative: the embedded copy only fills
	// paths that are still absent from the staged output.
	assetNames := make([]string, 0, len(files))
	for relSlash := range files {
		assetNames = append(assetNames, relSlash)
	}
	sort.Strings(assetNames)
	for _, relSlash := range assetNames {
		data := files[relSlash]
		if !cfg.GraphExplorer && (relSlash == graphAssetDir || strings.HasPrefix(relSlash, graphAssetDir+"/")) {
			continue
		}
		if isThemeAsset[relSlash] && !publishThemeAsset(relSlash) {
			continue
		}

		destPath := filepath.Join(outDirClean, filepath.FromSlash(relSlash))
		if !pathutil.IsSafePath(outDirClean, destPath) {
			return fmt.Errorf("embedded asset %q escapes output directory", relSlash)
		}
		if info, statErr := os.Lstat(destPath); statErr == nil {
			if info.IsDir() {
				return fmt.Errorf("embedded asset destination is a directory: %s", destPath)
			}
			continue
		} else if !os.IsNotExist(statErr) {
			return statErr
		}

		if claim != nil {
			if err := claim("embedded asset " + relSlash); err != nil {
				return err
			}
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}
		// #nosec G306 -- published artifact must be readable by the web server (#637)
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return fmt.Errorf("write embedded asset %s: %w", relSlash, err)
		}
	}

	return nil
}

// referencedThemeAssets scans the pages already rendered into the staging
// tree, including images in Markdown, metadata, inline styles/scripts and
// subpath-prefixed URLs. Plain page text does not make an asset necessary.
// Authored CSS/JS can use a theme image via url() or script without mentioning
// it in HTML, so scan those source files too. Do not scan the untouched bundled
// CSS: comments about other themes are not references in published pages.
// Only the known bundled names are subject to filtering; arbitrary user assets
// are never pruned.
func referencedThemeAssets(outDir, assetRoot string, assetRootExists bool, projectRoot string, ignoreRules []IgnoreRule, names []string, bundled map[string][]byte) (map[string]bool, error) {
	needed := make(map[string]bool, len(names))
	scanNames := func(data []byte) {
		for _, name := range names {
			// A stylesheet may import another CSS file or image relative to
			// itself (url("../img/mascot-default.jpeg")), so a filename
			// match is safer than requiring the full /assets/ URL prefix.
			if !needed[name] && bytes.Contains(data, []byte(filepath.Base(name))) {
				needed[name] = true
			}
		}
	}
	scanHTML := func(file string) error {
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		tokenizer := html.NewTokenizer(bytes.NewReader(data))
		inScriptOrStyle := false
		for {
			switch tokenizer.Next() {
			case html.ErrorToken:
				if err := tokenizer.Err(); err != io.EOF {
					return err
				}
				return nil
			case html.StartTagToken, html.SelfClosingTagToken:
				token := tokenizer.Token()
				for _, attr := range token.Attr {
					scanNames([]byte(attr.Val))
				}
				inScriptOrStyle = token.Data == "script" || token.Data == "style"
			case html.EndTagToken:
				inScriptOrStyle = false
			case html.TextToken:
				if inScriptOrStyle {
					scanNames(tokenizer.Text())
				}
			}
		}
	}
	if err := filepath.WalkDir(filepath.Dir(outDir), func(file string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// outDir is the staging /assets directory; only scan its sibling
		// generated pages, not any pre-existing files elsewhere.
		if file == outDir {
			return filepath.SkipDir
		}
		if !d.IsDir() && d.Type().IsRegular() && filepath.Ext(file) == ".html" {
			return scanHTML(file)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if assetRootExists {
		if err := filepath.WalkDir(assetRoot, func(file string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.Type()&os.ModeSymlink != 0 {
				return nil
			}
			rel, err := filepath.Rel(assetRoot, file)
			if err != nil {
				return err
			}
			relSlash := filepath.ToSlash(rel)
			if !d.IsDir() && IsIgnoredAsset(file, false, relSlash, projectRoot, ignoreRules) {
				return nil
			}
			if !d.IsDir() && (filepath.Ext(file) == ".css" || filepath.Ext(file) == ".js") {
				source, readErr := os.ReadFile(file)
				if readErr != nil {
					return readErr
				}
				if data, ok := bundled[relSlash]; ok && bytes.Equal(source, data) {
					return nil
				}
				scanNames(source)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return needed, nil
}

type IgnoreRule struct {
	pattern       []string
	anchored      bool
	directoryOnly bool
	negated       bool
}

func ParseIgnoreRules(contents string) []IgnoreRule {
	var rules []IgnoreRule
	for _, line := range strings.Split(contents, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		rule := IgnoreRule{}
		if strings.HasPrefix(line, "!") {
			rule.negated = true
			line = strings.TrimPrefix(line, "!")
		}
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "/") {
			rule.anchored = true
			line = strings.TrimPrefix(line, "/")
		}
		if strings.HasSuffix(line, "/") {
			rule.directoryOnly = true
			line = strings.TrimRight(line, "/")
		}
		if line == "" {
			continue
		}

		rule.pattern = strings.Split(filepath.ToSlash(line), "/")
		rules = append(rules, rule)
	}
	return rules
}

func LoadIgnoreRules(projectRoot string) []IgnoreRule {
	if projectRoot == "" {
		return nil
	}
	gitignore, err := os.ReadFile(filepath.Join(projectRoot, ".gitignore"))
	if err != nil {
		return nil
	}
	return ParseIgnoreRules(string(gitignore))
}

// IsIgnored applies rules in file order, matching the final applicable rule.
// Paths are slash-separated and relative to the directory containing .gitignore.
func IsIgnored(slashPath string, isDir bool, rules []IgnoreRule) bool {
	segments := strings.Split(strings.Trim(slashPath, "/"), "/")
	ignored := false
	for _, rule := range rules {
		if rule.matches(segments, isDir) {
			ignored = !rule.negated
		}
	}
	return ignored
}

func isIgnored(slashPath string, isDir bool, rules []IgnoreRule) bool {
	return IsIgnored(slashPath, isDir, rules)
}

func IsIgnoredAsset(path string, isDir bool, relSlash string, projectRoot string, ignoreRules []IgnoreRule) bool {
	if filepath.Ext(path) == ".go" || strings.Contains(relSlash, "/testdata/") || strings.HasPrefix(relSlash, "testdata/") || relSlash == "testdata" {
		return true
	}
	if len(ignoreRules) > 0 && projectRoot != "" {
		projectRel, err := filepath.Rel(projectRoot, path)
		if err == nil {
			projectSlash := filepath.ToSlash(projectRel)
			if projectSlash != "." && filepath.IsLocal(projectRel) && IsIgnored(projectSlash, isDir, ignoreRules) {
				return true
			}
		}
	}
	return false
}

func (rule IgnoreRule) matches(segments []string, isDir bool) bool {
	if len(rule.pattern) == 1 && !rule.anchored {
		for i, segment := range segments {
			candidateIsDir := i < len(segments)-1 || isDir
			if (!rule.directoryOnly || candidateIsDir) && matchSegment(rule.pattern[0], segment) {
				return true
			}
		}
		return false
	}

	for end := 1; end <= len(segments); end++ {
		candidateIsDir := end < len(segments) || isDir
		if rule.directoryOnly && !candidateIsDir {
			continue
		}
		if matchPath(rule.pattern, segments[:end]) {
			return true
		}
	}
	return false
}

func matchPath(pattern, candidate []string) bool {
	if len(pattern) == 0 {
		return len(candidate) == 0
	}
	if pattern[0] == "**" {
		return matchPath(pattern[1:], candidate) || (len(candidate) > 0 && matchPath(pattern, candidate[1:]))
	}
	return len(candidate) > 0 && matchSegment(pattern[0], candidate[0]) && matchPath(pattern[1:], candidate[1:])
}

func matchSegment(pattern, candidate string) bool {
	matched, err := path.Match(pattern, candidate)
	return err == nil && matched
}

func CopyFile(src, dst string) (err error) {
	source, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer source.Close()

	destination, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("failed to establish destination: %w", err)
	}
	defer func() {
		cerr := destination.Close()
		if err == nil {
			err = cerr
		}
	}()

	if _, err = io.Copy(destination, source); err != nil {
		return fmt.Errorf("payload copy error: %w", err)
	}

	return nil
}
