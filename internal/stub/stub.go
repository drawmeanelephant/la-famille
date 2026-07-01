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
	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/graph"
	"github.com/tbuddy/la-famille/internal/page"
	"github.com/tbuddy/la-famille/internal/transform"
)

func findPartials() (map[string]string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	var templatesDir string
	for {
		potential := filepath.Join(wd, "templates")
		if stat, err := os.Stat(potential); err == nil && stat.IsDir() {
			templatesDir = potential
			break
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			// Reached root without finding it, just return empty to not break existing flow
			return nil, nil
		}
		wd = parent
	}

	partialsDir := filepath.Join(templatesDir, "partials")
	if _, err := os.Stat(partialsDir); os.IsNotExist(err) {
		return nil, nil
	}

	partials := make(map[string]string)
	err = filepath.WalkDir(partialsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(d.Name()) == ".html" {
			rel, err := filepath.Rel(templatesDir, path)
			if err != nil {
				return err
			}
			partials[filepath.ToSlash(rel)] = path
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return partials, nil
}

func GenerateStubs(cfg config.Config, missingFiles map[string][]string, g *graph.Graph, p *bluemonday.Policy, fileMap map[string]*content.FileMeta) error {
	var missingKeys []string
	for k := range missingFiles {
		missingKeys = append(missingKeys, k)
	}
	sort.Strings(missingKeys)

	for _, missingRelPath := range missingKeys {
		outDirClean := filepath.Clean(cfg.OutputDir)
		outPath := filepath.Join(outDirClean, filepath.FromSlash(missingRelPath))
		if !strings.HasPrefix(outPath, outDirClean+string(filepath.Separator)) && outPath != outDirClean {
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

		// derive outPath using clean URL logic
		relOut := transform.GetOutputURL(missingRelPath, "")
		outPath = filepath.Join(outDirClean, filepath.FromSlash(relOut))

		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return err
		}

		// build simple HTML stub
		var htmlContent strings.Builder
		htmlContent.WriteString("<div class=\"alert alert-warning shadow-lg mb-8\">\n")
		htmlContent.WriteString("  <div>\n")
		htmlContent.WriteString("    <svg xmlns=\"http://www.w3.org/2000/svg\" class=\"stroke-current flex-shrink-0 h-6 w-6\" fill=\"none\" viewBox=\"0 0 24 24\"><path stroke-linecap=\"round\" stroke-linejoin=\"round\" stroke-width=\"2\" d=\"M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z\" /></svg>\n")
		htmlContent.WriteString("    <div>\n")
		htmlContent.WriteString("      <h3 class=\"font-bold\">🌱 This page is a stub</h3>\n")
		htmlContent.WriteString("      <div class=\"text-xs\">The content for this page hasn't been written yet.</div>\n")
		htmlContent.WriteString("    </div>\n")
		htmlContent.WriteString("  </div>\n")
		htmlContent.WriteString("</div>\n")
		htmlContent.WriteString("<h3>Return paths</h3>\n")
		htmlContent.WriteString("<p>This missing page was referenced by the following pages:</p>\n")
		htmlContent.WriteString("<ul class=\"menu bg-base-100 border border-base-300 rounded-box w-full\">\n")
		for _, parent := range parents {
			parentSlug := ""
			if meta, ok := fileMap[parent]; ok && meta != nil {
				parentSlug = meta.Slug
				if parentSlug != "" {
					if !filepath.IsLocal(parentSlug) || strings.Contains(parentSlug, ".") || strings.Contains(parentSlug, string(filepath.Separator)) || strings.Contains(parentSlug, "/") {
						parentSlug = ""
					}
				}
			}

			currOut := transform.GetOutputURL(missingRelPath, "")
			parentOut := transform.GetOutputURL(parent, parentSlug)

			currDir := filepath.Dir(currOut)
			if currDir == "." {
				currDir = ""
			}

			relParent, err := filepath.Rel(currDir, parentOut)
			if err == nil {
				relParentSlash := filepath.ToSlash(relParent)
				if strings.HasSuffix(relParentSlash, "index.html") {
					if relParentSlash == "index.html" {
						relParentSlash = "./"
					} else {
						relParentSlash = strings.TrimSuffix(relParentSlash, "index.html")
					}
				}
				htmlContent.WriteString(fmt.Sprintf("<li><a href=\"%s\">%s</a></li>\n", html.EscapeString(relParentSlash), html.EscapeString(parent)))
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

		partials, _ := findPartials()
		b, err := os.ReadFile(cfg.Template)
		if err != nil {
			outFile.Close()
			return fmt.Errorf("failed to read default template file for stubs: %w", err)
		}

		defaultTmpl := template.New(filepath.Base(cfg.Template))
		defaultTmpl, err = defaultTmpl.Parse(string(b))
		if err != nil {
			outFile.Close()
			return fmt.Errorf("failed to parse default template file for stubs: %w", err)
		}

		for name, path := range partials {
			pb, err := os.ReadFile(path)
			if err != nil {
				outFile.Close()
				return fmt.Errorf("failed to read partial %s for stubs: %w", path, err)
			}
			_, err = defaultTmpl.New(name).Parse(string(pb))
			if err != nil {
				outFile.Close()
				return fmt.Errorf("failed to parse partial %s for stubs: %w", path, err)
			}
		}

		if err := defaultTmpl.ExecuteTemplate(outFile, filepath.Base(cfg.Template), pageStruct); err != nil {
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
