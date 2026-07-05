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
	"github.com/tbuddy/la-famille/internal/pathutil"
	"github.com/tbuddy/la-famille/internal/render"
	"github.com/tbuddy/la-famille/internal/transform"
)

func GenerateStubs(cfg config.Config, missingFiles map[string][]string, g *graph.Graph, p *bluemonday.Policy, fileMap map[string]*content.FileMeta) error {
	missingKeys := make([]string, 0, len(missingFiles))
	for k := range missingFiles {
		missingKeys = append(missingKeys, k)
	}
	sort.Strings(missingKeys)

	partials, _ := render.DiscoverPartials(filepath.Dir(cfg.Template))

	for _, missingRelPath := range missingKeys {
		if err := generateSingleStub(cfg, missingRelPath, missingFiles[missingRelPath], g, p, fileMap, partials); err != nil {
			return err
		}
	}
	return nil
}

func generateSingleStub(cfg config.Config, missingRelPath string, parents []string, g *graph.Graph, p *bluemonday.Policy, fileMap map[string]*content.FileMeta, partials map[string]string) error {
	outDirClean := filepath.Clean(cfg.OutputDir)

	relOut := transform.GetOutputURL(missingRelPath, "")
	outPath := filepath.Join(outDirClean, filepath.FromSlash(relOut))

	if !pathutil.IsSafePath(outDirClean, outPath) {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}

	sort.Strings(parents)
	id := strings.TrimSuffix(missingRelPath, ".md")
	g.Nodes[id] = graph.Node{
		Type:         "stub",
		Render:       true,
		Missing:      true,
		ReferencedBy: parents,
	}

	var htmlContent strings.Builder
	htmlContent.WriteString("<div class=\"alert alert-warning shadow-lg mb-8\">\n")
	htmlContent.WriteString("  <div>\n")
	htmlContent.WriteString("    <svg xmlns=\"http://www.w3.org/2000/svg\" class=\"stroke-current flex-shrink-0 h-6 w-6\" fill=\"none\" viewBox=\"0 0 24 24\"><path stroke-linecap=\"round\" stroke-linejoin=\"round\" stroke-width=\"2\" d=\"M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z\" /></svg>\n")
	htmlContent.WriteString("    <div>\n")
	htmlContent.WriteString("      <h3 class=\"font-bold\">🚧 Under Construction</h3>\n")
	htmlContent.WriteString("      <div class=\"text-xs\">We are still working on this content. Please check back later!</div>\n")
	htmlContent.WriteString("    </div>\n")
	htmlContent.WriteString("  </div>\n")
	htmlContent.WriteString("</div>\n")
	htmlContent.WriteString("<h3>Where did you come from?</h3>\n")
	htmlContent.WriteString("<p>You can return to the previous context by visiting one of these pages that link here:</p>\n")
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

		parentOut := transform.GetOutputURL(parent, parentSlug)
		currDir := filepath.Dir(relOut)
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
		Content: template.HTML(p.SanitizeBytes([]byte(htmlContent.String()))), // #nosec G203
	}

	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	b, err := os.ReadFile(cfg.Template)
	if err != nil {
		return fmt.Errorf("failed to read default template file for stubs: %w", err)
	}

	defaultTmpl := template.New(filepath.Base(cfg.Template))
	defaultTmpl, err = defaultTmpl.Parse(string(b))
	if err != nil {
		return fmt.Errorf("failed to parse default template file for stubs: %w", err)
	}

	for name, path := range partials {
		pb, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read partial %s for stubs: %w", path, err)
		}
		_, err = defaultTmpl.New(name).Parse(string(pb))
		if err != nil {
			return fmt.Errorf("failed to parse partial %s for stubs: %w", path, err)
		}
	}

	if err := defaultTmpl.ExecuteTemplate(outFile, filepath.Base(cfg.Template), pageStruct); err != nil {
		return err
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
