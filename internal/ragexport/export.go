package ragexport

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tbuddy/la-famille/internal/config"
)

// RunExport exports project files into RAG-friendly markdown bundles
func RunExport(cfg config.Config) error {
	outDir := cfg.RagDir
	if outDir == "" {
		outDir = "rag-archive"
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	fmt.Printf("RAG archive directory created at %s\n", outDir)

	// 1. System Bundle
	if err := writeBundle(
		filepath.Join(outDir, "rag-system.md"),
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
		outDir,
		cfg.ProjectRoot,
	); err != nil {
		return fmt.Errorf("failed to write system bundle: %w", err)
	}
	fmt.Println("Created rag-system.md")

	// 2. Config/Templates Bundle
	if err := writeBundle(
		filepath.Join(outDir, "rag-config.md"),
		[]string{
			"internal/config/**/*.go",
			".jules/**/*.md",
		},
		nil,
		nil,
		outDir,
		cfg.ProjectRoot,
	); err != nil {
		return fmt.Errorf("failed to write config bundle: %w", err)
	}

	// Append assets listing to Config/Templates Bundle
	cfgFile, err := os.OpenFile(filepath.Join(outDir, "rag-config.md"), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open config bundle for appending assets: %w", err)
	}
	defer cfgFile.Close()

	cfgFile.WriteString("<file path=\"assets/\">\n<content>\n")
	filepath.WalkDir(filepath.Join(cfg.ProjectRoot, "assets"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // ignore missing assets dir
		}
		// if it's a directory, just print the path with a trailing slash
		if d.IsDir() {
			cfgFile.WriteString(filepath.ToSlash(getRel(cfg.ProjectRoot, path)) + "/\n")
		} else {
			// for files, print size and name
			info, err := d.Info()
			size := int64(0)
			if err == nil {
				size = info.Size()
			}
			cfgFile.WriteString(fmt.Sprintf("%s (size: %d bytes)\n", filepath.ToSlash(getRel(cfg.ProjectRoot, path)), size))
		}
		return nil
	})
	cfgFile.WriteString("</content>\n</file>\n\n")

	cfgFile.WriteString("<file path=\"templates/\">\n<content>\n")
	filepath.WalkDir(filepath.Join(cfg.ProjectRoot, "templates"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // ignore missing templates dir
		}
		// if it's a directory, just print the path with a trailing slash
		if d.IsDir() {
			cfgFile.WriteString(filepath.ToSlash(getRel(cfg.ProjectRoot, path)) + "/\n")
		} else {
			// for files, print size and name
			info, err := d.Info()
			size := int64(0)
			if err == nil {
				size = info.Size()
			}
			cfgFile.WriteString(fmt.Sprintf("%s (size: %d bytes)\n", filepath.ToSlash(getRel(cfg.ProjectRoot, path)), size))
		}
		return nil
	})
	cfgFile.WriteString("</content>\n</file>\n\n")

	fmt.Println("Created rag-config.md")

	// 3. Content Bundle
	if err :=
		writeBundle(
			filepath.Join(outDir, "rag-content.md"),
			[]string{
				"content/**/*.md",
			},
			[]string{"content/jules"},
			nil, // Default formatting is verbatim with XML tags, which preserves the YAML frontmatter
			outDir,
			cfg.ProjectRoot,
		); err != nil {
		return fmt.Errorf("failed to write content bundle: %w", err)
	}
	fmt.Println("Created rag-content.md")

	return nil
}

func writeBundle(outPath string, patterns []string, excludes []string, formatFunc func(path string, content []byte) string, outDir string, projectRoot string) error {
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
			if d.IsDir() {
				if d.Name() == ".git" || d.Name() == "test-results" || d.Name() == "public" || d.Name() == "vendor" || d.Name() == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}

			relPath := getRel(projectRoot, path)
			if pathMatch(pattern, filepath.ToSlash(relPath)) {
				if strings.Contains(filepath.ToSlash(relPath), filepath.ToSlash(outDir)) {
					return nil
				}
				// Check excludes
				isExcluded := false
				for _, exclude := range excludes {
					if strings.HasPrefix(filepath.ToSlash(relPath), filepath.ToSlash(exclude)) {
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
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var output string
		if formatFunc != nil {
			output = formatFunc(path, content)
		} else {
			output = fmt.Sprintf("<file path=\"%s\">\n<content>\n%s\n</content>\n</file>\n\n", filepath.ToSlash(getRel(projectRoot, path)), string(content))
		}
		if _, err := f.WriteString(output); err != nil {
			return err
		}
	}

	return nil
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
