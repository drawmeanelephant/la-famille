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

	// Find partials directory (relative to CWD, which might be deep during tests)
	partialsDir := filepath.Join("templates", "partials")
	if _, err := os.Stat(partialsDir); os.IsNotExist(err) {
		// Fallback for tests running in subdirectories (like cmd/la-famille)
		cwd, _ := os.Getwd()
		for i := 0; i < 3; i++ {
			cwd = filepath.Dir(cwd)
			testDir := filepath.Join(cwd, "templates", "partials")
			if _, err := os.Stat(testDir); err == nil {
				partialsDir = testDir
				break
			}
		}
	}

	// Collect all partials
	partialsGlob := filepath.Join(partialsDir, "*.html")
	partialFiles, _ := filepath.Glob(partialsGlob)

	// Combine main template and partials for single parsing call
	templateFiles := append([]string{templatePath}, partialFiles...)

	pageTmpl, err := template.ParseFiles(templateFiles...)
	if err != nil {
		return fmt.Errorf("failed to parse templates %v: %w", templateFiles, err)
	}

	if err := pageTmpl.Execute(outFile, p); err != nil {
		return err
	}
	return nil
}
