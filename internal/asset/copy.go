package asset

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/pathutil"
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

// ClaimOutput reserves an output-relative path for the asset copier. It returns
// the existing owner and false when another producer has already claimed that
// path. A nil ClaimOutput disables ownership tracking, which is what the
// package's own tests want; the generator always supplies one.
type ClaimOutput func(relOut string) (owner string, ok bool)

func CopyAssets(cfg config.Config, claim ClaimOutput) error {
	if cfg.AssetDir == "" {
		return nil
	}

	var ignoreRules []IgnoreRule
	if cfg.ProjectRoot != "" {
		ignoreRules = LoadIgnoreRules(cfg.ProjectRoot)
	}

	if _, err := os.Stat(cfg.AssetDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
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
	if pathutil.IsSafePath(walkRoot, outputResolved) {
		return fmt.Errorf("asset directory %q resolves to %q, which contains the output directory; copying it would publish the output into itself", cfg.AssetDir, walkRoot)
	}

	return filepath.WalkDir(walkRoot, func(path string, d os.DirEntry, err error) error {
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
			relOut := "assets/" + relSlash
			if owner, ok := claim(relOut); !ok {
				return fmt.Errorf("output path collision: the asset %q and %s both write %q", relSlash, owner, relOut)
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
	})
}

type IgnoreRule struct {
	pattern       []string
	anchored      bool
	directoryOnly bool
	negated       bool
}

type ignoreRule = IgnoreRule

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

func parseIgnoreRules(contents string) []IgnoreRule {
	return ParseIgnoreRules(contents)
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

func (rule ignoreRule) matches(segments []string, isDir bool) bool {
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

	destination, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
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
