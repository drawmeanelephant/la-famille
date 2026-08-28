package taxonomy

import (
	"fmt"
	"html"
	"html/template"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/microcosm-cc/bluemonday"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/page"
	"github.com/tbuddy/la-famille/internal/render"
	"github.com/tbuddy/la-famille/internal/search"
	"github.com/tbuddy/la-famille/internal/transform"
)

type groupSpec struct {
	getItems func(meta *content.FileMeta) []string
	singular string
	plural   string
	prefix   string
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
func GenerateTags(cfg, siteCfg config.Config, fileMap map[string]*content.FileMeta, renderer *render.Renderer, p *bluemonday.Policy) ([]string, []search.Item, error) {
	return generateTaxonomyGroup(cfg, siteCfg, fileMap, renderer, p, tagsSpec)
}

// GenerateCategories generates rendered category pages and category index pages.
func GenerateCategories(cfg, siteCfg config.Config, fileMap map[string]*content.FileMeta, renderer *render.Renderer, p *bluemonday.Policy) ([]string, []search.Item, error) {
	return generateTaxonomyGroup(cfg, siteCfg, fileMap, renderer, p, categoriesSpec)
}

// GenerateTaxonomies generates rendered pages for all supported taxonomies (tags, categories)
// and returns the relative output paths and search items of all generated HTML pages.
func GenerateTaxonomies(cfg, siteCfg config.Config, fileMap map[string]*content.FileMeta, renderer *render.Renderer, p *bluemonday.Policy) ([]string, []search.Item, error) {
	tagPaths, tagItems, err := generateTaxonomyGroup(cfg, siteCfg, fileMap, renderer, p, tagsSpec)
	if err != nil {
		return nil, nil, err
	}
	catPaths, catItems, err := generateTaxonomyGroup(cfg, siteCfg, fileMap, renderer, p, categoriesSpec)
	if err != nil {
		return nil, nil, err
	}
	allPaths := append(tagPaths, catPaths...)
	sort.Strings(allPaths)
	allItems := append(tagItems, catItems...)
	return allPaths, allItems, nil
}

func generateTaxonomyGroup(cfg, siteCfg config.Config, fileMap map[string]*content.FileMeta, renderer *render.Renderer, p *bluemonday.Policy, spec groupSpec) ([]string, []search.Item, error) {
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
		return nil, nil, nil
	}

	outDirClean := filepath.Clean(cfg.OutputDir)
	var generatedPaths []string
	var searchItems []search.Item

	// Render main index page for the group (e.g. tags/index.html or categories/index.html)
	indexRelPath := fmt.Sprintf("%s/index.md", spec.prefix)
	indexOut := transform.GetOutputURL(indexRelPath, "", true)
	indexOutPath := filepath.Join(outDirClean, filepath.FromSlash(indexOut))

	if err := os.MkdirAll(filepath.Dir(indexOutPath), 0755); err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}
	generatedPaths = append(generatedPaths, indexOut)
	searchItems = append(searchItems, search.Item{
		Title: spec.plural,
		URL:   siteCfg.PublicPathForOutput(indexOut),
	})

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
			return nil, nil, err
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
			return nil, nil, err
		}
		generatedPaths = append(generatedPaths, itemOut)
		searchItems = append(searchItems, search.Item{
			Title:   fmt.Sprintf("%s: %s", spec.singular, item),
			URL:     siteCfg.PublicPathForOutput(itemOut),
			Tags:    []string{item},
			TagURLs: []string{siteCfg.PublicPathForOutput(itemOut)},
		})
	}

	sort.Strings(generatedPaths)
	return generatedPaths, searchItems, nil
}

// NavLinks returns the site links with taxonomy archive entries appended for
// every group that produced archive pages: Tags → /tags/ and Categories →
// /categories/ (#529). A group is surfaced only when at least one rendered
// page carries a non-blank term, mirroring generateTaxonomyGroup's filtering,
// so the link never points at an archive that does not exist. Links the
// operator already configured (by label or by URL) are never duplicated.
func NavLinks(links []config.SiteLink, fileMap map[string]*content.FileMeta) []config.SiteLink {
	groups := []struct {
		label    string
		prefix   string
		getItems func(meta *content.FileMeta) []string
	}{
		{label: "Tags", prefix: "tags", getItems: func(meta *content.FileMeta) []string { return meta.Tags }},
		{label: "Categories", prefix: "categories", getItems: func(meta *content.FileMeta) []string { return meta.Categories }},
	}
	for _, g := range groups {
		if !groupHasTerms(fileMap, g.getItems) {
			continue
		}
		href := "/" + g.prefix + "/"
		if siteLinkExists(links, g.label, href) {
			continue
		}
		links = append(links, config.SiteLink{Label: g.label, URL: href})
	}
	return links
}

// PageTagLinks renders the linked tag list appended to an article's body so
// its tag archives are reachable from the post itself, not only from the nav
// and sitemap (#529). Hrefs are relative to the page's output file, matching
// how archive pages link to each other and how markdown links are rewritten,
// so they resolve at any depth and under subpath deploys without the
// base-path rewrite. The block is sanitized with p. Returns nil when the page
// has no tags.
func PageTagLinks(tags []string, pageOut string, p *bluemonday.Policy) []byte {
	names := dedupeTrimmed(tags)
	if len(names) == 0 {
		return nil
	}

	var b strings.Builder
	b.WriteString(`<p class="page-tags"><strong>Tags:</strong>`)
	for _, name := range names {
		itemOut := transform.GetOutputURL(fmt.Sprintf("tags/%s/index.md", name), "", true)
		b.WriteString(` <a class="tag-link" href="`)
		b.WriteString(html.EscapeString(relativeHref(pageOut, itemOut)))
		b.WriteString(`">`)
		b.WriteString(html.EscapeString(name))
		b.WriteString(`</a>`)
	}
	b.WriteString(`</p>`)
	return p.SanitizeBytes([]byte(b.String()))
}

// dedupeTrimmed returns tags with whitespace trimmed, blanks dropped, and
// duplicates removed, preserving the order they appear in the frontmatter.
func dedupeTrimmed(tags []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		out = append(out, tag)
	}
	return out
}

// relativeHref renders a link from one generated output file to another,
// relative to the source's directory, with the trailing index.html stripped so
// the target resolves to its directory URL. It returns "" when the two paths
// share no usable relative link.
func relativeHref(fromOut, toOut string) string {
	currDir := filepath.Dir(fromOut)
	if currDir == "." {
		currDir = ""
	}
	rel, err := filepath.Rel(currDir, toOut)
	if err != nil {
		return ""
	}
	relSlash := filepath.ToSlash(rel)
	if strings.HasSuffix(relSlash, "index.html") {
		if relSlash == "index.html" {
			relSlash = "./"
		} else {
			relSlash = strings.TrimSuffix(relSlash, "index.html")
		}
	}
	return relSlash
}

// groupHasTerms reports whether any rendered page carries a non-blank term
// for the group, the same filter generateTaxonomyGroup applies before it
// writes archive pages.
func groupHasTerms(fileMap map[string]*content.FileMeta, getItems func(meta *content.FileMeta) []string) bool {
	for _, meta := range fileMap {
		if meta.Render != nil && !*meta.Render {
			continue
		}
		for _, item := range getItems(meta) {
			if strings.TrimSpace(item) != "" {
				return true
			}
		}
	}
	return false
}

// siteLinkExists reports whether the configured links already name the label
// or the URL, so the generator never duplicates an operator's own nav entry.
func siteLinkExists(links []config.SiteLink, label, href string) bool {
	wantPath := pathForLink(href)
	for _, l := range links {
		if strings.EqualFold(strings.TrimSpace(l.Label), label) {
			return true
		}
		if pathForLink(l.URL) == wantPath {
			return true
		}
	}
	return false
}

// pathForLink extracts the URL path from a configured link, accepting both
// absolute URLs and root-relative paths, with trailing slashes normalized
// away so /tags/ and /tags compare equal.
func pathForLink(raw string) string {
	raw = strings.TrimSpace(raw)
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		raw = u.Path
	}
	return strings.TrimRight(raw, "/")
}
