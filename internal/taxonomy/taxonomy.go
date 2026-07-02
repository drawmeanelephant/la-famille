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

func GenerateTags(cfg config.Config, fileMap map[string]*content.FileMeta, renderer *render.Renderer, p *bluemonday.Policy) error {
	tagMap := make(map[string][]string)

	for relPath, meta := range fileMap {
		if meta.Render != nil && !*meta.Render {
			continue
		}
		for _, tag := range meta.Tags {
			tagMap[tag] = append(tagMap[tag], relPath)
		}
	}

	var tags []string
	for tag := range tagMap {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	outDirClean := filepath.Clean(cfg.OutputDir)

	for _, tag := range tags {
		pages := tagMap[tag]
		sort.Strings(pages)

		tagRelPath := fmt.Sprintf("tags/%s/index.md", tag)
		tagOut := transform.GetOutputURL(tagRelPath, "")
		outPath := filepath.Join(outDirClean, filepath.FromSlash(tagOut))

		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return err
		}

		var htmlContent strings.Builder
		htmlContent.WriteString(fmt.Sprintf("<h2>Tag: %s</h2>\n", html.EscapeString(tag)))
		htmlContent.WriteString("<ul>\n")

		for _, relPath := range pages {
			meta := fileMap[relPath]

			title := meta.Title
			if title == "" {
				title = filepath.Base(relPath)
			}

			pageOut := transform.GetOutputURL(relPath, meta.Slug)

			currDir := filepath.Dir(tagOut)
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
			Site:    cfg,
			Title:   fmt.Sprintf("Tag: %s", tag),
			Content: template.HTML(sanitizedHTML), // #nosec G203
		}

		if err := renderer.HTML(cfg, pageStruct, "", outPath); err != nil {
			return err
		}
	}
	return nil
}
