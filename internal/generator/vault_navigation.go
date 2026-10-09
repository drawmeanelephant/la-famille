package generator

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/graphexplorer"
	"github.com/tbuddy/la-famille/internal/page"
	"github.com/tbuddy/la-famille/internal/transform"
)

const (
	unresolvedNotesOutput = "unresolved-notes/index.html"
	unresolvedNotesURL    = "/unresolved-notes/"
)

func (bc *buildContext) addUnresolvedNotesNavLink() {
	for _, link := range bc.siteCfg.SiteLinks {
		if link.URL == unresolvedNotesURL {
			return
		}
	}
	bc.siteCfg.SiteLinks = append(bc.siteCfg.SiteLinks, config.SiteLink{
		Label: "Unresolved Notes",
		URL:   unresolvedNotesURL,
	})
}

// renderBacklinksPanels adds inbound links from the backlinks collected by
// LinkTransformer to each rendered note page. Stubs already render their
// inbound references as part of the stub page.
func (bc *buildContext) renderBacklinksPanels() error {
	ids := make([]string, 0, len(bc.pageOutputs))
	for id := range bc.pageOutputs {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		output := bc.pageOutputs[id]
		outputPath := filepath.Join(bc.cfg.OutputDir, filepath.FromSlash(output))
		data, err := os.ReadFile(outputPath)
		if err != nil {
			return fmt.Errorf("read rendered note %q for backlinks: %w", output, err)
		}

		panel := []byte(bc.backlinksPanel(id, output))
		bodyEnd := bytes.LastIndex(data, []byte("</body>"))
		if bodyEnd < 0 {
			data = append(data, panel...)
		} else {
			updated := make([]byte, 0, len(data)+len(panel))
			updated = append(updated, data[:bodyEnd]...)
			updated = append(updated, panel...)
			data = append(updated, data[bodyEnd:]...)
		}
		// #nosec G306 -- published artifact must be readable by the web server (#637)
		if err := os.WriteFile(outputPath, data, 0644); err != nil {
			return fmt.Errorf("write rendered note %q with backlinks: %w", output, err)
		}
	}
	return nil
}

func (bc *buildContext) backlinksPanel(id, currentOutput string) string {
	var panel strings.Builder
	panel.WriteString(`<section class="backlinks-panel" aria-labelledby="backlinks-heading">`)
	panel.WriteString(`<h2 id="backlinks-heading">Linked from</h2>`)

	parents := bc.backlinks[id]
	if len(parents) == 0 {
		panel.WriteString(`<p>No backlinks yet.</p>`)
		panel.WriteString(`</section>`)
		return panel.String()
	}

	parents = append([]string(nil), parents...)
	sort.Strings(parents)
	panel.WriteString("<ul>\n")
	lastParent := ""
	for _, parent := range parents {
		if parent == lastParent {
			continue
		}
		lastParent = parent

		title, parentOutput := bc.backlinkSource(parent)
		if parentOutput == "" {
			fmt.Fprintf(&panel, "<li>%s</li>\n", html.EscapeString(title))
			continue
		}
		href := relativeOutputURL(currentOutput, parentOutput)
		fmt.Fprintf(&panel, `<li><a href="%s">%s</a></li>`+"\n",
			html.EscapeString(href), html.EscapeString(title))
	}
	panel.WriteString("</ul></section>")
	return panel.String()
}

func (bc *buildContext) backlinkSource(id string) (title, output string) {
	sourcePath := id
	if !strings.HasSuffix(sourcePath, ".md") {
		sourcePath += ".md"
	}
	if meta := bc.fileMap[sourcePath]; meta != nil {
		title = strings.TrimSpace(meta.Title)
		if meta.Render != nil && !*meta.Render {
			output = sourcePath
		} else {
			output = bc.pageOutputs[strings.TrimSuffix(sourcePath, ".md")]
		}
	}
	if title == "" {
		title = graphexplorer.TitleFromID(id)
	}
	return title, output
}

func relativeOutputURL(fromOutput, toOutput string) string {
	relative, err := filepath.Rel(filepath.Dir(filepath.FromSlash(fromOutput)), filepath.FromSlash(toOutput))
	if err != nil {
		return ""
	}
	href := filepath.ToSlash(relative)
	if strings.HasSuffix(href, "index.html") {
		href = strings.TrimSuffix(href, "index.html")
		if href == "" {
			href = "./"
		}
	}
	return (&url.URL{Path: href}).String()
}

func (bc *buildContext) writeUnresolvedNotesIndex() error {
	sort.Strings(bc.generatedStubIDs)
	var contentHTML strings.Builder
	contentHTML.WriteString(`<section class="unresolved-notes" aria-labelledby="unresolved-notes-heading">`)
	contentHTML.WriteString(`<h2 id="unresolved-notes-heading">Notes to write</h2>`)
	if len(bc.generatedStubIDs) == 0 {
		contentHTML.WriteString("<p>No unresolved notes.</p></section>")
	} else {
		contentHTML.WriteString("<ul>\n")
		for _, id := range bc.generatedStubIDs {
			title := strings.TrimSpace(bc.missingTitles[id+".md"])
			if title == "" {
				title = graphexplorer.TitleFromID(id)
			}
			stubOutput := transform.GetOutputURL(id+".md", "", true)
			href := relativeOutputURL(unresolvedNotesOutput, stubOutput)
			fmt.Fprintf(&contentHTML, `<li><a href="%s">%s</a></li>`+"\n",
				html.EscapeString(href), html.EscapeString(title))
		}
		contentHTML.WriteString("</ul></section>")
	}

	relOutput := unresolvedNotesOutput
	outPath := filepath.Join(bc.cfg.OutputDir, filepath.FromSlash(relOutput))
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return fmt.Errorf("create unresolved-notes index directory: %w", err)
	}
	contentBytes := bc.sanitizer.SanitizeBytes([]byte(contentHTML.String()))
	indexPage := page.Page{
		Site:         bc.siteCfg,
		Title:        "Unresolved Notes",
		Content:      template.HTML(contentBytes), // #nosec G203 -- content is built from escaped names and sanitized
		CanonicalURL: bc.siteCfg.URLForOutputPath(relOutput),
	}
	if err := bc.renderer.HTML(bc.cfg, indexPage, "", outPath); err != nil {
		return fmt.Errorf("render unresolved-notes index: %w", err)
	}

	bc.renderedPaths = append(bc.renderedPaths, relOutput)
	bc.result.PageCount++
	return nil
}
