package stub

import (
	"fmt"
	"html"
	"html/template"
	"log/slog"
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

// ClaimOutput reserves the output-relative path a stub is about to write.
// Stubs are generated last and write with os.Create, so without a reservation a
// single dangling link is enough to overwrite a taxonomy listing, a rendered
// page or another stub at exit 0.
//
// It returns ("", true) when the path was free and is now the stub's, or the
// description of the writer that already owns it and false. A nil ClaimOutput
// means "nothing else writes here" and every stub is written.
type ClaimOutput func(missingRelPath, relOut string) (owner string, ok bool)

func GenerateStubs(cfg, siteCfg config.Config, missingFiles map[string][]string, g *graph.Graph, p *bluemonday.Policy, fileMap map[string]*content.FileMeta, claim ClaimOutput) error {
	missingKeys := make([]string, 0, len(missingFiles))
	for k := range missingFiles {
		missingKeys = append(missingKeys, k)
	}
	sort.Strings(missingKeys)

	partials, _ := render.DiscoverPartials(filepath.Dir(cfg.Template))

	for _, missingRelPath := range missingKeys {
		if err := generateSingleStub(cfg, siteCfg, missingRelPath, missingFiles[missingRelPath], g, p, fileMap, partials, claim); err != nil {
			return err
		}
	}
	return nil
}

func generateSingleStub(cfg, siteCfg config.Config, missingRelPath string, parents []string, g *graph.Graph, p *bluemonday.Policy, fileMap map[string]*content.FileMeta, partials map[string]string, claim ClaimOutput) error {
	outDirClean := filepath.Clean(cfg.OutputDir)
	relOut := transform.GetOutputURL(missingRelPath, "", true)
	outPath := filepath.Join(outDirClean, filepath.FromSlash(relOut))

	if !pathutil.IsSafePath(outDirClean, outPath) {
		return nil
	}

	sort.Strings(parents)

	// The node is recorded either way: the link that produced it is already an
	// edge in the graph, and dropping the node would leave that edge dangling.
	writeStub := true
	if claim != nil {
		owner, ok := claim(missingRelPath, relOut)
		if !ok {
			slog.Warn("Not writing a stub over generated output",
				"missing", missingRelPath, "path", relOut, "owner", owner)
			writeStub = false
		}
	}
	id := strings.TrimSuffix(missingRelPath, ".md")
	g.Nodes[id] = graph.Node{
		Type:         "stub",
		Render:       true,
		Missing:      true,
		ReferencedBy: parents,
	}

	if !writeStub {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}

	var htmlContent strings.Builder
	htmlContent.WriteString("<div class=\"alert alert-warning shadow-lg mb-8\">\n  <div>\n")
	htmlContent.WriteString("    <svg xmlns=\"http://www.w3.org/2000/svg\" class=\"stroke-current flex-shrink-0 h-6 w-6\" fill=\"none\" viewBox=\"0 0 24 24\"><path stroke-linecap=\"round\" stroke-linejoin=\"round\" stroke-width=\"2\" d=\"M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z\" /></svg>\n")
	htmlContent.WriteString("    <div>\n      <h3 class=\"font-bold\">🚧 Under Construction</h3>\n")
	htmlContent.WriteString("      <div class=\"text-xs\">We are still working on this content. Please check back later!</div>\n    </div>\n  </div>\n</div>\n")
	htmlContent.WriteString("<h3>Where did you come from?</h3>\n<ul class=\"menu bg-base-100 border border-base-300 rounded-box w-full\">\n")

	for _, parent := range parents {
		parentSlug := ""
		if meta, ok := fileMap[parent]; ok && meta != nil && usableSlug(meta.Slug) {
			// An unusable slug is discarded when the parent is rendered, so
			// honouring it here would link to a page that does not exist.
			parentSlug = meta.Slug
		}

		parentRender := true
		if meta, ok := fileMap[parent]; ok && meta != nil && meta.Render != nil && !*meta.Render {
			parentRender = false
		}

		parentOut := transform.GetOutputURL(parent, parentSlug, parentRender)
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
		Site:         siteCfg,
		Title:        "Missing Page",
		Content:      template.HTML(p.SanitizeBytes([]byte(htmlContent.String()))), // #nosec G203
		CanonicalURL: siteCfg.URLForOutputPath(relOut),
	}

	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	b, err := os.ReadFile(cfg.Template)
	if err != nil {
		return fmt.Errorf("failed to read layout: %w", err)
	}

	defaultTmpl := template.New(filepath.Base(cfg.Template))
	defaultTmpl, err = defaultTmpl.Parse(string(b))
	if err != nil {
		return err
	}

	for name, path := range partials {
		pb, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = defaultTmpl.New(name).Parse(string(pb))
		if err != nil {
			return err
		}
	}

	return defaultTmpl.ExecuteTemplate(outFile, filepath.Base(cfg.Template), pageStruct)
}

// usableSlug mirrors the check the render worker applies before it computes a
// page's output path: a slug that fails it is discarded, so any code predicting
// where that page landed has to discard it too. The durable home for this rule
// is transform.GetOutputURL, which every predictor already calls.
func usableSlug(slug string) bool {
	if slug == "" {
		return false
	}
	return filepath.IsLocal(slug) &&
		!strings.Contains(slug, ".") &&
		!strings.Contains(slug, string(filepath.Separator)) &&
		!strings.Contains(slug, "/")
}

func RelPathFromTo(base, target string) (string, error) {
	baseDir := filepath.Dir(base)
	rel, err := filepath.Rel(baseDir, target)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}
