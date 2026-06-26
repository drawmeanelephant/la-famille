package stub

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
	"github.com/tbuddy/la-famille/internal/graph"
	"github.com/tbuddy/la-famille/internal/page"
)

func GenerateStubs(cfg config.Config, missingFiles map[string][]string, g *graph.Graph, p *bluemonday.Policy) error {
	var missingKeys []string
	for k := range missingFiles {
		missingKeys = append(missingKeys, k)
	}
	sort.Strings(missingKeys)

	for _, missingRelPath := range missingKeys {
		if !filepath.IsLocal(filepath.FromSlash(missingRelPath)) {
			continue
		}

		parents := missingFiles[missingRelPath]
		sort.Strings(parents)
		id := strings.TrimSuffix(missingRelPath, ".md")
		g.Nodes[id] = graph.Node{
			Type:         "stub",
			Render:       true,
			Missing:      true,
			ReferencedBy: parents,
		}

		outPath := filepath.Join(cfg.OutputDir, filepath.FromSlash(missingRelPath))
		// ensure the missing relative path has .html
		outPath = strings.TrimSuffix(outPath, ".md") + ".html"

		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return err
		}

		// build simple HTML stub
		var htmlContent strings.Builder
		htmlContent.WriteString("<h2>🌱 This page is a stub</h2>\n")
		htmlContent.WriteString("<p>The content for this page hasn't been written yet.</p>\n<hr>\n")
		htmlContent.WriteString("<h3>Return paths</h3>\n")
		htmlContent.WriteString("<p>This missing page was referenced by the following pages:</p>\n<ul>\n")
		for _, parent := range parents {
			parentHtml := strings.TrimSuffix(parent, ".md") + ".html"
			// determine relative path from missing file to parent file for linking
			relParent, err := RelPathFromTo(missingRelPath, parentHtml)
			if err == nil {
				htmlContent.WriteString(fmt.Sprintf("<li><a href=\"%s\">%s</a></li>\n", html.EscapeString(relParent), html.EscapeString(parent)))
			} else {
				htmlContent.WriteString(fmt.Sprintf("<li>%s</li>\n", html.EscapeString(parent)))
			}
		}
		htmlContent.WriteString("</ul>\n")

		pageStruct := page.Page{
			Site:    cfg,
			Title:   "Missing Page",
			Content: template.HTML(p.SanitizeBytes([]byte(htmlContent.String()))),
		}

		outFile, err := os.Create(outPath)
		if err != nil {
			return err
		}

		defaultTmpl, err := template.ParseFiles(cfg.Template)
		if err != nil {
			outFile.Close()
			return fmt.Errorf("failed to parse default template file for stubs: %w", err)
		}

		if err := defaultTmpl.Execute(outFile, pageStruct); err != nil {
			outFile.Close()
			return err
		}
		outFile.Close()
	}
	return nil
}

// RelPathFromTo computes the relative URL path from base (e.g. dir1/missing.md) to target (e.g. index.html)
func RelPathFromTo(base, target string) (string, error) {
	baseDir := filepath.Dir(base)
	rel, err := filepath.Rel(baseDir, target)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}
