package render

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/page"
)

// HTMLPage writes the sanitized HTML content into a template and saves it to outPath.
func HTMLPage(cfg config.Config, meta *content.FileMeta, title string, sanitizedHTML []byte, outPath string) error {
	p := page.Page{
		Site:            cfg,
		Title:           title,
		Author:          meta.Author,
		Date:            meta.Date,
		VideoScript:     meta.VideoScript,
		AnimationCues:   meta.AnimationCues,
		SoundtrackTheme: meta.SoundtrackTheme,
		Layout:          meta.Layout,
		Content:         template.HTML(sanitizedHTML),
	}

	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	templatePath := cfg.Template
	if meta.Layout != "" {
		if !filepath.IsLocal(meta.Layout + ".html") {
			log.Printf("Warning: Potential path traversal in layout template loading detected: %s. Falling back to default %s", meta.Layout, cfg.Template)
		} else {
			layoutPath := filepath.Join("templates", meta.Layout+".html")
			// If we are running tests, the templates directory is relative to the root, but the test might run from cmd/la-famille
			if _, err := os.Stat(layoutPath); os.IsNotExist(err) {
				layoutPathFallback := filepath.Join("..", "..", "templates", meta.Layout+".html")
				if _, err2 := os.Stat(layoutPathFallback); err2 == nil {
					layoutPath = layoutPathFallback
				}
			}
			if _, err := os.Stat(layoutPath); err == nil {
				templatePath = layoutPath
			} else {
				log.Printf("Warning: layout template %s not found, falling back to %s", layoutPath, cfg.Template)
			}
		}
	}

	pageTmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", templatePath, err)
	}

	if err := pageTmpl.Execute(outFile, p); err != nil {
		return err
	}

	return nil
}
