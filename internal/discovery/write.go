// Package discovery writes standard files that help crawlers discover a site.
package discovery

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/tbuddy/la-famille/internal/config"
)

const sitemapNamespace = "http://www.sitemaps.org/schemas/sitemap/0.9"

type sitemap struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Location string `xml:"loc"`
}

// Write creates sitemap.xml and robots.txt for the supplied rendered page
// output paths. A missing SiteURL is supported: the sitemap remains valid but
// contains no unverifiable absolute URLs, and robots.txt omits its Sitemap
// directive.
func Write(cfg config.Config, renderedPaths []string) error {
	if err := os.MkdirAll(cfg.OutputDir, 0700); err != nil {
		return fmt.Errorf("create discovery output directory: %w", err)
	}

	urls := sitemapURLs(cfg, renderedPaths)
	contents, err := xml.MarshalIndent(sitemap{XMLNS: sitemapNamespace, URLs: urls}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sitemap: %w", err)
	}
	contents = append([]byte(xml.Header), append(contents, '\n')...)
	if err := os.WriteFile(filepath.Join(cfg.OutputDir, "sitemap.xml"), contents, 0600); err != nil {
		return fmt.Errorf("write sitemap: %w", err)
	}

	robots := "User-agent: *\nAllow: /\n"
	if sitemapURL := cfg.URLForOutputPath("sitemap.xml"); sitemapURL != "" {
		robots += "\nSitemap: " + sitemapURL + "\n"
	}
	if err := os.WriteFile(filepath.Join(cfg.OutputDir, "robots.txt"), []byte(robots), 0600); err != nil {
		return fmt.Errorf("write robots: %w", err)
	}
	return nil
}

func sitemapURLs(cfg config.Config, renderedPaths []string) []sitemapURL {
	unique := make(map[string]struct{}, len(renderedPaths))
	for _, outputPath := range renderedPaths {
		if outputPath == "" {
			continue
		}
		if absoluteURL := cfg.URLForOutputPath(outputPath); absoluteURL != "" {
			unique[absoluteURL] = struct{}{}
		}
	}

	locations := make([]string, 0, len(unique))
	for location := range unique {
		locations = append(locations, location)
	}
	sort.Strings(locations)

	urls := make([]sitemapURL, 0, len(locations))
	for _, location := range locations {
		urls = append(urls, sitemapURL{Location: location})
	}
	return urls
}
