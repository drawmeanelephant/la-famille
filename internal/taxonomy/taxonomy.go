package taxonomy

import (
	"fmt"
	"html"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/microcosm-cc/bluemonday"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/page"
	"github.com/tbuddy/la-famille/internal/render"
	"github.com/tbuddy/la-famille/internal/transform"
)

type groupSpec struct {
	singular string
	plural   string
	prefix   string
	getItems func(meta *content.FileMeta) []string
}

var (
	tagsSpec = groupSpec{
		singular: "Tag",
		plural:   "Tags",
		prefix:   "tags",
		getItems: func(meta *content.FileMeta) []string {
			return meta.Tags
		},
	}
	categoriesSpec = groupSpec{
		singular: "Category",
		plural:   "Categories",
		prefix:   "categories",
		getItems: func(meta *content.FileMeta) []string {
			return meta.Categories
		},
	}
)

// GenerateTags generates rendered tag pages and tag index pages.
func GenerateTags(cfg, siteCfg config.Config, fileMap map[string]*content.FileMeta, renderer *render.Renderer, p *bluemonday.Policy) error {
	_, err := generateTaxonomyGroup(cfg, siteCfg, fileMap, renderer, p, tagsSpec)
	return err
}

// GenerateCategories generates rendered category pages and category index pages.
func GenerateCategories(cfg, siteCfg config.Config, fileMap map[string]*content.FileMeta, renderer *render.Renderer, p *bluemonday.Policy) error {
	_, err := generateTaxonomyGroup(cfg, siteCfg, fileMap, renderer, p, categoriesSpec)
	return err
}

// GenerateTaxonomies generates rendered pages for all supported taxonomies (tags, categories)
// and returns the relative output paths of all generated HTML pages.
func GenerateTaxonomies(cfg, siteCfg config.Config, fileMap map[string]*content.FileMeta, renderer *render.Renderer, p *bluemonday.Policy) ([]string, error) {
	tagPaths, err := generateTaxonomyGroup(cfg, siteCfg, fileMap, renderer, p, tagsSpec)
	if err != nil {
		return nil, err
	}
	catPaths, err := generateTaxonomyGroup(cfg, siteCfg, fileMap, renderer, p, categoriesSpec)
	if err != nil {
		return nil, err
	}
	allPaths := append(tagPaths, catPaths...)
	sort.Strings(allPaths)
	return allPaths, nil
}

func generateTaxonomyGroup(cfg, siteCfg config.Config, fileMap map[string]*content.FileMeta, renderer *render.Renderer, p *bluemonday.Policy, spec groupSpec) ([]string, error) {
	itemMap := make(map[string][]string)

	for relPath, meta := range fileMap {
		if meta.Render != nil && !*meta.Render {
			continue
		}
		for _, item := range spec.getItems(meta) {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			itemMap[item] = append(itemMap[item], relPath)
		}
	}

	items := make([]string, 0, len(itemMap))
	for item := range itemMap {
		items = append(items, item)
	}
	sort.Strings(items)

	if len(items) == 0 {
		return nil, nil
	}

	outDirClean := filepath.Clean(cfg.OutputDir)
	var generatedPaths []string

	// Render main index page for the group (e.g. tags/index.html or categories/index.html)
	indexRelPath := fmt.Sprintf("%s/index.md", spec.prefix)
	indexOut := transform.GetOutputURL(indexRelPath, "", true)
	indexOutPath := filepath.Join(outDirClean, filepath.FromSlash(indexOut))

	if err := os.MkdirAll(filepath.Dir(indexOutPath), 0755); err != nil {
		return nil, err
	}

	var indexHTML strings.Builder
	indexHTML.WriteString(fmt.Sprintf("<h2>%s</h2>\n<ul>\n", html.EscapeString(spec.plural)))

	for _, item := range items {
		itemRelPath := fmt.Sprintf("%s/%s/index.md", spec.prefix, item)
		itemOut := transform.GetOutputURL(itemRelPath, "", true)

		currDir := filepath.Dir(indexOut)
		if currDir == "." {
			currDir = ""
		}

		relOut, err := filepath.Rel(currDir, itemOut)
		if err == nil {
			relOutSlash := filepath.ToSlash(relOut)
			if strings.HasSuffix(relOutSlash, "index.html") {
				if relOutSlash == "index.html" {
					relOutSlash = "./"
				} else {
					relOutSlash = strings.TrimSuffix(relOutSlash, "index.html")
				}
			}
			indexHTML.WriteString(fmt.Sprintf("<li><a href=\"%s\">%s</a></li>\n", html.EscapeString(relOutSlash), html.EscapeString(item)))
		}
	}
	indexHTML.WriteString("</ul>\n")

	sanitizedIndex := p.SanitizeBytes([]byte(indexHTML.String()))
	indexPageStruct := page.Page{
		Site:         siteCfg,
		Title:        spec.plural,
		Content:      template.HTML(sanitizedIndex), // #nosec G203
		CanonicalURL: siteCfg.URLForOutputPath(indexOut),
	}

	if err := renderer.HTML(cfg, indexPageStruct, "", indexOutPath); err != nil {
		return nil, err
	}
	generatedPaths = append(generatedPaths, indexOut)

	// Render individual taxonomy item pages (e.g., tags/go/index.html)
	for _, item := range items {
		rawPages := itemMap[item]
		// Deduplicate pages per taxonomy item
		seenPages := make(map[string]bool)
		var pages []string
		for _, pagePath := range rawPages {
			if !seenPages[pagePath] {
				seenPages[pagePath] = true
				pages = append(pages, pagePath)
			}
		}
		sort.Strings(pages)

		itemRelPath := fmt.Sprintf("%s/%s/index.md", spec.prefix, item)
		itemOut := transform.GetOutputURL(itemRelPath, "", true)
		outPath := filepath.Join(outDirClean, filepath.FromSlash(itemOut))

		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return nil, err
		}

		var htmlContent strings.Builder
		htmlContent.WriteString(fmt.Sprintf("<h2>%s: %s</h2>\n", html.EscapeString(spec.singular), html.EscapeString(item)))
		htmlContent.WriteString("<ul>\n")

		for _, relPath := range pages {
			meta := fileMap[relPath]
			title := meta.Title
			if title == "" {
				title = filepath.Base(relPath)
			}

			pageRender := true
			if meta.Render != nil && !*meta.Render {
				pageRender = false
			}
			pageOut := transform.GetOutputURL(relPath, meta.Slug, pageRender)

			currDir := filepath.Dir(itemOut)
			if currDir == "." {
				currDir = ""
			}

			relOut, err := filepath.Rel(currDir, pageOut)
			if err == nil {
				relOutSlash := filepath.ToSlash(relOut)
				if strings.HasSuffix(relOutSlash, "index.html") {
					if relOutSlash == "index.html" {
						relOutSlash = "./"
					} else {
						relOutSlash = strings.TrimSuffix(relOutSlash, "index.html")
					}
				}
				htmlContent.WriteString(fmt.Sprintf("<li><a href=\"%s\">%s</a></li>\n", html.EscapeString(relOutSlash), html.EscapeString(title)))
			}
		}
		htmlContent.WriteString("</ul>\n")

		sanitizedHTML := p.SanitizeBytes([]byte(htmlContent.String()))

		pageStruct := page.Page{
			Site:         siteCfg,
			Title:        fmt.Sprintf("%s: %s", spec.singular, item),
			Content:      template.HTML(sanitizedHTML), // #nosec G203
			CanonicalURL: siteCfg.URLForOutputPath(itemOut),
		}

		if err := renderer.HTML(cfg, pageStruct, "", outPath); err != nil {
			return nil, err
		}
		generatedPaths = append(generatedPaths, itemOut)
	}

	sort.Strings(generatedPaths)
	return generatedPaths, nil
}
