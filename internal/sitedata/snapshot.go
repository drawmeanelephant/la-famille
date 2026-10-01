package sitedata

import (
	"crypto/sha256"
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// SnapshotOutput completes the source/graph projection with the bytes actually
// published and the actual sitemap, not predictions reconstructed from URLs.
func SnapshotOutput(manifest *Manifest, outputDir string, pageOutputs map[string]string) error {
	manifest.OutputCaptured = false
	pagePaths := make(map[string]int, len(manifest.Pages))
	for i, page := range manifest.Pages {
		path := pageOutputs[page.Identity]
		if !page.Rendered {
			path = page.SourcePath
		}
		if path != "" {
			pagePaths[filepath.ToSlash(path)] = i
		}
		if path == "" {
			return fmt.Errorf("published page %q has no output path", page.Identity)
		}
		sum, err := fingerprintOutputFile(filepath.Join(outputDir, filepath.FromSlash(path)))
		if err != nil {
			return fmt.Errorf("snapshot published page %q: %w", page.Identity, err)
		}
		manifest.Pages[i].OutputHash = sum
	}
	manifest.Files = nil
	err := filepath.WalkDir(outputDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(outputDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == ManifestFileName || rel == "diff.json" || rel == "diff.txt" {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("cannot fingerprint non-regular published file %q", rel)
		}
		if _, ok := pagePaths[rel]; ok {
			return nil
		}
		sum, err := fingerprintOutputFile(path)
		if err != nil {
			return err
		}
		manifest.Files = append(manifest.Files, ManifestFile{Path: rel, Hash: sum})
		return nil
	})
	if err != nil {
		return fmt.Errorf("snapshot published files: %w", err)
	}
	data, err := os.ReadFile(filepath.Join(outputDir, "sitemap.xml"))
	if err != nil {
		return fmt.Errorf("snapshot sitemap: %w", err)
	}
	var sitemap struct {
		URLs []struct {
			Location string `xml:"loc"`
		} `xml:"url"`
	}
	if err := xml.Unmarshal(data, &sitemap); err != nil {
		return fmt.Errorf("snapshot sitemap: %w", err)
	}
	manifest.Sitemap = nil
	for _, item := range sitemap.URLs {
		manifest.Sitemap = append(manifest.Sitemap, item.Location)
	}
	manifest.Sitemap = sortedUniqueStrings(manifest.Sitemap)
	for _, page := range manifest.Pages {
		if page.OutputHash == "" {
			return fmt.Errorf("published page %q is missing from the output snapshot", page.Identity)
		}
	}
	manifest.OutputCaptured = true
	return nil
}

func fingerprintOutputFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	_, copyErr := io.Copy(hash, f)
	closeErr := f.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
