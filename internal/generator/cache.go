package generator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tbuddy/la-famille/internal/config"
)

const cacheFileName = ".la-famille-cache.json"

type buildCache struct {
	Version        int           `json:"version"`
	Fingerprint    string        `json:"fingerprint"`
	GeneratedFiles []string      `json:"generated_files"`
	PageCount      int           `json:"page_count"`
	Health         ContentHealth `json:"health,omitempty"`
}

func cachePath(outputDir string) string { return filepath.Join(outputDir, cacheFileName) }

func cacheFingerprint(cfg config.Config, roots ...string) (string, error) {
	h := sha256.New()
	// WatchMode is operational state and must not invalidate generated output.
	cfg.WatchMode = false
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	_, _ = h.Write(data)

	for _, root := range roots {
		if err := hashTree(h, root); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func hashTree(h io.Writer, root string) error {
	entries := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.Type()&os.ModeSymlink != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		entries = append(entries, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	sort.Strings(entries)
	for _, rel := range entries {
		path := filepath.Join(root, filepath.FromSlash(rel))
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, _ = io.WriteString(h, filepath.ToSlash(root)+"\x00"+rel+"\x00")
		_, _ = h.Write(contents)
	}
	return nil
}

func loadBuildCache(path string) (buildCache, error) {
	var cache buildCache
	data, err := os.ReadFile(path)
	if err != nil {
		return cache, err
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		return cache, err
	}
	if cache.Version != 1 {
		return cache, fmt.Errorf("unsupported cache version %d", cache.Version)
	}
	return cache, nil
}

func cacheUsable(cache buildCache, outputDir, fingerprint string) bool {
	if cache.Fingerprint != fingerprint || len(cache.GeneratedFiles) == 0 {
		return false
	}
	actualFiles, err := generatedFiles(outputDir)
	if err != nil || len(actualFiles) != len(cache.GeneratedFiles) {
		return false
	}
	for i, rel := range cache.GeneratedFiles {
		if rel == cacheFileName || filepath.IsAbs(rel) || strings.Contains(rel, "..") {
			return false
		}
		if actualFiles[i] != rel {
			return false
		}
	}
	return true
}

func generatedFiles(outputDir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(outputDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.Type()&os.ModeSymlink != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(outputDir, path)
		if err != nil {
			return err
		}
		if filepath.Clean(rel) == cacheFileName {
			return nil
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func writeBuildCache(path, fingerprint string, files []string, pageCount int, health ContentHealth) error {
	cache := buildCache{Version: 1, Fingerprint: fingerprint, GeneratedFiles: files, PageCount: pageCount, Health: health}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
