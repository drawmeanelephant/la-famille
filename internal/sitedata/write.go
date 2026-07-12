package sitedata

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tbuddy/la-famille/internal/jsonutil"
	"github.com/tbuddy/la-famille/internal/pathutil"
	"github.com/tbuddy/la-famille/internal/transform"
)

// Write writes the meta data to the output directory.
func Write(outputDir string, metaData map[string]map[string]interface{}) error {
	if err := jsonutil.WriteJSON(filepath.Join(outputDir, "meta.json"), metaData); err != nil {
		return fmt.Errorf("failed to write meta.json: %w", err)
	}

	// Generate sitemap.xml
	outDirClean := filepath.Clean(outputDir)
	sitemapPath := filepath.Join(outDirClean, "sitemap.xml")

	// Safeguard against path traversal using IsSafePath
	if !pathutil.IsSafePath(outDirClean, sitemapPath) {
		return fmt.Errorf("potential path traversal writing sitemap: %s", sitemapPath)
	}

	keys := make([]string, 0, len(metaData))
	for k := range metaData {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sitemapBuilder strings.Builder
	sitemapBuilder.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	sitemapBuilder.WriteString("<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")

	for _, k := range keys {
		// Calculate the output URL
		meta := metaData[k]
		slug := ""
		if slugVal, ok := meta["slug"].(string); ok {
			slug = slugVal
		}

		renderFlag := true
		if r, ok := meta["render"].(bool); ok {
			renderFlag = r
		}

		relPath := k
		if !strings.HasSuffix(relPath, ".md") {
			relPath += ".md"
		}
		relOut := transform.GetOutputURL(relPath, slug, renderFlag)
		urlPath := "/" + filepath.ToSlash(relOut)

		sitemapBuilder.WriteString(fmt.Sprintf("\t<url>\n\t\t<loc>%s</loc>\n\t</url>\n", urlPath))
	}
	sitemapBuilder.WriteString("</urlset>\n")

	if err := os.WriteFile(sitemapPath, []byte(sitemapBuilder.String()), 0600); err != nil {
		return fmt.Errorf("failed to write sitemap.xml: %w", err)
	}

	return nil
}
