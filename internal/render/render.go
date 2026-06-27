package render

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/page"
)

func findPartials() ([]string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	var partialsDir string
	for {
		potential := filepath.Join(wd, "templates", "partials")
		if stat, err := os.Stat(potential); err == nil && stat.IsDir() {
			partialsDir = potential
			break
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			// Reached root without finding it, just return empty to not break existing flow
			return nil, nil
		}
		wd = parent
	}

	var partials []string
	entries, err := os.ReadDir(partialsDir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".html" {
			partials = append(partials, filepath.Join(partialsDir, e.Name()))
		}
	}
	return partials, nil
}

// HTML renders a page struct using the specified layout template.
func HTML(cfg config.Config, p page.Page, layout, outPath string) error {
	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	templatePath := cfg.Template
	if layout != "" {
		if !filepath.IsLocal(layout + ".html") {
			log.Printf("Warning: Potential path traversal in layout template loading detected: %s. Falling back to default %s", layout, cfg.Template)
		} else {
			layoutPath := filepath.Join("templates", layout+".html")
			// If we are running tests, the templates directory is relative to the root, but the test might run from cmd/la-famille
			if _, err := os.Stat(layoutPath); os.IsNotExist(err) {
				layoutPathFallback := filepath.Join("..", "..", "templates", layout+".html")
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

		partials, _ := findPartials()
	allTmpls := append([]string{templatePath}, partials...)
	pageTmpl, err := template.ParseFiles(allTmpls...)
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", templatePath, err)
	}

	if err := pageTmpl.Execute(outFile, p); err != nil {
		return err
	}
	return nil
}
