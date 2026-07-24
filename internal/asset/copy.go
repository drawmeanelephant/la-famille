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

func CopyAssets(cfg config.Config) error {
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

	return filepath.WalkDir(cfg.AssetDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.Type()&os.ModeSymlink != 0 {
			slog.Warn("Skipping symlink in assets", "path", path)
			return nil
		}
		relPath, err := filepath.Rel(cfg.AssetDir, path)
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

		if IsIgnoredAsset(path, d.IsDir(), relSlash, cfg.ProjectRoot, ignoreRules) {
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
