package render

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/page"
)

type Renderer struct {
	cache     map[string]*template.Template
	allowlist map[string]bool
	mu        sync.Mutex
}

func New(templateDir string) *Renderer {
	allowlist, err := DiscoverLayouts(templateDir)
	if err != nil {
		allowlist = make(map[string]bool)
	}
	return &Renderer{
		cache:     make(map[string]*template.Template),
		allowlist: allowlist,
	}
}

// DiscoverLayouts walks the templates directory to find available layouts.
func DiscoverLayouts(templateDir string) (map[string]bool, error) {
	allowlist := make(map[string]bool)
	entries, err := os.ReadDir(templateDir)
	if err != nil {
		return allowlist, err
	}
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".html" {
			allowlist[strings.TrimSuffix(e.Name(), ".html")] = true
		}
	}
	return allowlist, nil
}

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

// HTML renders a page struct using the specified layout template.
func (r *Renderer) HTML(cfg config.Config, p page.Page, layout, outPath string) error {
	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	templatePath := cfg.Template
	if layout != "" {
		if !r.allowlist[layout] {
			log.Printf("Warning: Layout %q not found in allowlist. Falling back to default %s", layout, cfg.Template)
			layout = ""
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

	r.mu.Lock()
	cachedTmpl, exists := r.cache[templatePath]
	if !exists {
		partials, _ := findPartials()

		b, err := os.ReadFile(templatePath)
		if err != nil {
			r.mu.Unlock()
			return fmt.Errorf("failed to read template %s: %w", templatePath, err)
		}

		parsedTmpl := template.New(filepath.Base(templatePath))
		parsedTmpl, err = parsedTmpl.Parse(string(b))
		if err != nil {
			r.mu.Unlock()
			return fmt.Errorf("failed to parse template %s: %w", templatePath, err)
		}

		for name, path := range partials {
			pb, err := os.ReadFile(path)
			if err != nil {
				r.mu.Unlock()
				return fmt.Errorf("failed to read partial %s: %w", path, err)
			}
			_, err = parsedTmpl.New(name).Parse(string(pb))
			if err != nil {
				r.mu.Unlock()
				return fmt.Errorf("failed to parse partial %s: %w", path, err)
			}
		}
		r.cache[templatePath] = parsedTmpl
		cachedTmpl = parsedTmpl
	}
	r.mu.Unlock()

	clonedTmpl, err := cachedTmpl.Clone()
	if err != nil {
		return fmt.Errorf("failed to clone template %s: %w", templatePath, err)
	}

	// Use ExecuteTemplate with the base name to avoid the ParseFiles name trap
	templateName := filepath.Base(templatePath)
	if cfg.WatchMode {
		var buf bytes.Buffer
		if err := clonedTmpl.ExecuteTemplate(&buf, templateName, p); err != nil {
			return err
		}

		script := `<script>
		if (window.EventSource) {
			var source = new EventSource('/livereload');
			source.onmessage = function(e) {
				if (e.data === 'reload') {
					window.location.reload();
				}
			};
		}
		</script>
</body>`

		htmlStr := buf.String()
		htmlStr = strings.Replace(htmlStr, "</body>", script, 1)

		_, err = outFile.WriteString(htmlStr)
		return err
	}

	if err := clonedTmpl.ExecuteTemplate(outFile, templateName, p); err != nil {
		return err
	}
	return nil
}
