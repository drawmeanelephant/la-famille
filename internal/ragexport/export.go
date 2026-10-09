package ragexport

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/ragfmt"
)

// RunExport exports project files into RAG-friendly markdown bundles
func RunExport(cfg config.Config) error {
	outDir := cfg.RagDir
	if outDir == "" {
		outDir = "rag-archive"
	}
	absOut, err := filepath.Abs(outDir)
	if err != nil {
		return fmt.Errorf("failed to resolve output directory: %w", err)
	}
	outDir = absOut

	// A RAG directory at or above the project root makes the walk's own
	// output exclusion match every source file, producing empty bundles that
	// look successful — refuse it outright (#643).
	if isWithinDir(cfg.ProjectRoot, outDir) {
		return fmt.Errorf("RagDir (%s) must not be the project root or contain it; the archive walk would exclude every source file", cfg.RagDir)
	}

	// Only a real directory holding regular (or absent) bundle files may be
	// replaced: a symlink planted in a cloned repository must not be
	// followed (#646).
	if err := checkArchiveDestination(outDir); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(outDir), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	// Bundles are written into a fresh staging sibling and swapped into
	// place only once all three are complete, so a failed run leaves the
	// previous archive untouched rather than a mixed generation (#645).
	stagingDir, err := os.MkdirTemp(filepath.Dir(outDir), "."+filepath.Base(outDir)+".staging-")
	if err != nil {
		return fmt.Errorf("failed to create staging directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(stagingDir) }()

	// Walks exclude the real archive dir (it may hold a stale generation
	// inside the project) and the staging dir (it sits beside the archive,
	// inside the project in the common public/rag-archive layout, and would
	// otherwise leak its half-written bundles into the next generation).
	excludeDirs := []string{outDir, stagingDir}

	contentDir := bundleDir(cfg.ContentDir, "content", cfg.ProjectRoot)
	assetDir := bundleDir(cfg.AssetDir, "assets", cfg.ProjectRoot)
	templateDir, templateFile := templateBundleTarget(cfg)

	// 1. System Bundle
	if err := writeBundle(
		filepath.Join(stagingDir, "rag-system.md"),
		[]string{
			"cmd/**/*.go",
			"internal/**/*.go",
			"pkg/**/*.go",
			"*.go",
			"go.mod",
			"go.sum",
			"README.md",
			"playwright_test.js",
			".github/workflows/*.yml",
		},
		[]string{"internal/config"},
		nil,
		excludeDirs,
		cfg.ProjectRoot,
	); err != nil {
		return fmt.Errorf("failed to write system bundle: %w", err)
	}

	// 2. Config/Templates Bundle
	if err := writeBundle(
		filepath.Join(stagingDir, "rag-config.md"),
		[]string{
			"internal/config/**/*.go",
			".jules/**/*.md",
		},
		nil,
		nil,
		excludeDirs,
		cfg.ProjectRoot,
	); err != nil {
		return fmt.Errorf("failed to write config bundle: %w", err)
	}

	// Append assets listing to Config/Templates Bundle
	cfgFile, err := os.OpenFile(filepath.Join(stagingDir, "rag-config.md"), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open config bundle for appending assets: %w", err)
	}

	_, _ = cfgFile.WriteString(fmt.Sprintf("<file path=\"%s/\">\n<content>\n", assetDir))
	_ = filepath.WalkDir(filepath.Join(cfg.ProjectRoot, assetDir), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // ignore missing assets dir
		}
		if withinAnyDir(path, excludeDirs) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		// if it's a directory, just print the path with a trailing slash
		if d.IsDir() {
			_, _ = cfgFile.WriteString(filepath.ToSlash(getRel(cfg.ProjectRoot, path)) + "/\n")
		} else {
			// for files, print size and name
			info, err := d.Info()
			size := int64(0)
			if err == nil {
				size = info.Size()
			}
			_, _ = cfgFile.WriteString(fmt.Sprintf("%s (size: %d bytes)\n", filepath.ToSlash(getRel(cfg.ProjectRoot, path)), size))
		}
		return nil
	})
	_, _ = cfgFile.WriteString("</content>\n</file>\n\n")

	if templateFile != "" {
		// The template sits at the project root, so there is no directory to
		// walk — listing "." would dump the whole project.
		_, _ = cfgFile.WriteString(fmt.Sprintf("<file path=\"%s\">\n<content>\n", templateFile))
		size := int64(0)
		if info, err := os.Stat(filepath.Join(cfg.ProjectRoot, filepath.FromSlash(templateFile))); err == nil {
			size = info.Size()
		}
		_, _ = cfgFile.WriteString(fmt.Sprintf("%s (size: %d bytes)\n", templateFile, size))
	} else {
		_, _ = cfgFile.WriteString(fmt.Sprintf("<file path=\"%s/\">\n<content>\n", templateDir))
		_ = filepath.WalkDir(filepath.Join(cfg.ProjectRoot, templateDir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // ignore missing templates dir
			}
			if withinAnyDir(path, excludeDirs) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			// if it's a directory, just print the path with a trailing slash
			if d.IsDir() {
				_, _ = cfgFile.WriteString(filepath.ToSlash(getRel(cfg.ProjectRoot, path)) + "/\n")
			} else {
				// for files, print size and name
				info, err := d.Info()
				size := int64(0)
				if err == nil {
					size = info.Size()
				}
				_, _ = cfgFile.WriteString(fmt.Sprintf("%s (size: %d bytes)\n", filepath.ToSlash(getRel(cfg.ProjectRoot, path)), size))
			}
			return nil
		})
	}
	_, _ = cfgFile.WriteString("</content>\n</file>\n\n")

	// The config bundle must be closed before the swap: renaming a
	// directory that still holds an open file fails on Windows.
	if err := cfgFile.Close(); err != nil {
		return fmt.Errorf("failed to finish config bundle: %w", err)
	}

	// 3. Content Bundle
	contentExcludes, err := unpublishedContentExcludes(cfg, contentDir)
	if err != nil {
		return fmt.Errorf("failed to find unpublished content: %w", err)
	}
	contentExcludes = append(contentExcludes, contentDir+"/jules")
	if err :=
		writeBundle(
			filepath.Join(stagingDir, "rag-content.md"),
			[]string{
				contentDir + "/**/*.md",
			},
			contentExcludes,
			nil, // Default formatting is verbatim with XML tags, which preserves the YAML frontmatter
			excludeDirs,
			cfg.ProjectRoot,
		); err != nil {
		return fmt.Errorf("failed to write content bundle: %w", err)
	}

	if err := swapArchiveDir(outDir, stagingDir); err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("RAG archive directory created at %s", outDir))
	slog.Info("Created rag-system.md")
	slog.Info("Created rag-config.md")
	slog.Info("Created rag-content.md")

	return nil
}

func unpublishedContentExcludes(cfg config.Config, contentDir string) ([]string, error) {
	contentPath := filepath.Join(cfg.ProjectRoot, filepath.FromSlash(contentDir))
	if _, err := os.Stat(contentPath); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	files, err := content.GatherMetadata(contentPath)
	if err != nil {
		return nil, err
	}
	excludes := make([]string, 0)
	for relPath, meta := range files {
		if !content.IsPublished(meta) {
			excludes = append(excludes, filepath.ToSlash(filepath.Join(contentDir, relPath)))
		}
	}
	sort.Strings(excludes)
	return excludes, nil
}

func writeBundle(outPath string, patterns []string, excludes []string, formatFunc func(path string, content []byte) string, excludeDirs []string, projectRoot string) error {
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	var matchedFiles []string
	for _, pattern := range patterns {
		err := filepath.WalkDir(projectRoot, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if withinAnyDir(path, excludeDirs) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				if d.Name() == ".git" || d.Name() == "test-results" || d.Name() == "public" || d.Name() == "vendor" || d.Name() == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}

			relPath := getRel(projectRoot, path)
			if pathMatch(pattern, filepath.ToSlash(relPath)) {
				// Check excludes
				isExcluded := false
				for _, exclude := range excludes {
					excludedPath := strings.TrimSuffix(filepath.ToSlash(exclude), "/")
					if filepath.ToSlash(relPath) == excludedPath || strings.HasPrefix(filepath.ToSlash(relPath), excludedPath+"/") {
						isExcluded = true
						break
					}
				}
				if isExcluded {
					return nil
				}
				found := false
				for _, mf := range matchedFiles {
					if mf == path { // keep path for reading file later
						found = true
						break
					}
				}
				if !found {
					matchedFiles = append(matchedFiles, path)
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	sort.Strings(matchedFiles)

	for _, path := range matchedFiles {
		body, err := os.ReadFile(path)
		if err != nil {
			// A matched file that cannot be read must not vanish from the
			// bundle silently: the archive would look complete while
			// missing content the matched set promised (#644).
			return fmt.Errorf("failed to read %s: %w", filepath.ToSlash(getRel(projectRoot, path)), err)
		}

		var output string
		if formatFunc != nil {
			output = formatFunc(path, body)
		} else {
			// Escape any line of the file body that would otherwise read as
			// archive structure, so a source file or Markdown page that
			// documents this format cannot corrupt the bundle.
			output = fmt.Sprintf("<file path=\"%s\">\n<content>\n%s\n</content>\n</file>\n\n", filepath.ToSlash(getRel(projectRoot, path)), ragfmt.EscapeContent(string(body)))
		}
		if _, err := f.WriteString(output); err != nil {
			return err
		}
	}

	return nil
}

// checkArchiveDestination verifies that an existing archive directory can be
// replaced safely: it must be a real directory — never a symlink — and every
// bundle path it contains must be a regular file. A symlink planted in a
// cloned repository at one of these paths would otherwise redirect the
// export's writes outside the project (#646).
func checkArchiveDestination(outDir string) error {
	info, err := os.Lstat(outDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to inspect RAG archive directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("RAG archive path %s is not a directory", outDir)
	}
	for _, name := range []string{"rag-system.md", "rag-config.md", "rag-content.md"} {
		bundlePath := filepath.Join(outDir, name)
		info, err := os.Lstat(bundlePath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("failed to inspect RAG bundle %s: %w", bundlePath, err)
		}
		if !info.Mode().IsRegular() {
			kind := info.Mode().Type().String()
			if info.Mode()&os.ModeSymlink != 0 {
				kind = "symlink"
			}
			return fmt.Errorf("refusing to replace %s: expected a regular file, found %s", bundlePath, kind)
		}
	}
	return nil
}

// swapArchiveDir installs a fully written staging directory as the archive,
// keeping a failed swap recoverable: the previous archive is renamed aside
// first, so nothing is lost if the staging rename fails. It mirrors
// replaceOutputDirectory in internal/generator.
func swapArchiveDir(outDir, stagingDir string) error {
	parent := filepath.Dir(outDir)
	if filepath.Dir(stagingDir) != parent {
		return fmt.Errorf("staging directory must be a sibling of the RAG archive directory")
	}

	archiveExists := false
	if info, err := os.Lstat(outDir); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("RAG archive path %s is not a directory", outDir)
		}
		archiveExists = true
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect RAG archive directory: %w", err)
	}

	backupDir, err := os.MkdirTemp(parent, "."+filepath.Base(outDir)+".previous-")
	if err != nil {
		return fmt.Errorf("failed to create archive backup path: %w", err)
	}
	if err := os.Remove(backupDir); err != nil {
		return fmt.Errorf("failed to prepare archive backup path: %w", err)
	}

	if archiveExists {
		if err := os.Rename(outDir, backupDir); err != nil {
			return fmt.Errorf("failed to move existing archive aside: %w", err)
		}
	}
	if err := os.Rename(stagingDir, outDir); err != nil {
		if archiveExists {
			if restoreErr := os.Rename(backupDir, outDir); restoreErr != nil {
				return fmt.Errorf("failed to install RAG archive: %w; restoring previous archive from %s also failed: %v", err, backupDir, restoreErr)
			}
		}
		return fmt.Errorf("failed to install RAG archive: %w", err)
	}
	if archiveExists {
		if err := os.RemoveAll(backupDir); err != nil {
			slog.Warn("Failed to remove replaced RAG archive", "path", backupDir, "error", err)
		}
	}
	return nil
}

// withinAnyDir reports whether path is one of the given directories or a
// descendant of one of them.
func withinAnyDir(path string, dirs []string) bool {
	for _, dir := range dirs {
		if isWithinDir(path, dir) {
			return true
		}
	}
	return false
}

// templateBundleTarget decides what the config bundle should list for
// templates. A template configured at the project root (template:
// layout.html) has no directory of its own, and walking "." would dump the
// entire project, so the single file is listed instead. Exactly one of the
// two return values is non-empty.
func templateBundleTarget(cfg config.Config) (dir string, singleFile string) {
	tmpl := strings.TrimSpace(cfg.Template)
	if tmpl == "" {
		return "templates", ""
	}
	if d := bundleDir(filepath.Dir(tmpl), "", cfg.ProjectRoot); d != "" {
		return d, ""
	}
	if f := bundleDir(tmpl, "", cfg.ProjectRoot); f != "" {
		return "", f
	}
	return "templates", ""
}

// bundleDir normalises a configured directory for use in bundle patterns and
// walks, falling back to the historical default when nothing usable is set.
//
// The patterns and walks below are all resolved against projectRoot, so an
// absolute configured directory has to be brought back into that frame first —
// filepath.Join(root, "/abs/dir") silently yields root + "/abs/dir" rather than
// the directory the author configured. A directory outside projectRoot cannot
// be expressed as a bundle pattern at all, so it falls back to the default.
func bundleDir(configured, fallback, projectRoot string) string {
	dir := strings.TrimSpace(configured)
	if dir == "" {
		return fallback
	}
	if filepath.IsAbs(dir) {
		rel, err := filepath.Rel(projectRoot, dir)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fallback
		}
		dir = rel
	}
	dir = filepath.ToSlash(filepath.Clean(dir))
	if dir == "." || dir == "/" {
		return fallback
	}
	return strings.TrimSuffix(dir, "/")
}

// isWithinDir reports whether path is dir itself or a descendant of dir.
// Both paths are made absolute so a configured RAG output directory is
// excluded correctly regardless of how ProjectRoot or RagDir were written.
func isWithinDir(path, dir string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}

	rel, err := filepath.Rel(absDir, absPath)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel))
}

func pathMatch(pattern, path string) bool {
	if strings.Contains(pattern, "**/") {
		prefix := strings.Split(pattern, "**/")[0]
		suffix := strings.Split(pattern, "**/")[1]
		if prefix != "" && !strings.HasPrefix(path, prefix) {
			return false
		}
		match, _ := filepath.Match(suffix, filepath.Base(path))
		return match
	}
	match, _ := filepath.Match(pattern, path)
	return match
}

func getRel(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}
	return rel
}
