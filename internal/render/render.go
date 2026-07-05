package render

import (
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/page"
)

type cacheEntry struct {
	tmpl *template.Template
	err  error
}

type Renderer struct {
	cache       map[string]*cacheEntry
	onces       map[string]*sync.Once
	allowlist   map[string]bool
	mu          sync.RWMutex
	templateDir string
}

func New(templateDir string) *Renderer {
	allowlist, err := DiscoverLayouts(templateDir)
	if err != nil {
		allowlist = make(map[string]bool)
	}
	return &Renderer{
		cache:       make(map[string]*cacheEntry),
		onces:       make(map[string]*sync.Once),
		allowlist:   allowlist,
		templateDir: templateDir,
	}
}

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

func DiscoverPartials(templateDir string) (map[string]string, error) {
	partialsDir := filepath.Join(templateDir, "partials")
	if _, err := os.Stat(partialsDir); os.IsNotExist(err) {
		return nil, nil
	}

	partials := make(map[string]string)
	err := filepath.WalkDir(partialsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(d.Name()) == ".html" {
			rel, err := filepath.Rel(templateDir, path)
			if err != nil {
				return err
			}
			partials[filepath.ToSlash(rel)] = path
		}
		return nil
	})
	return partials, err
}

func (r *Renderer) HTML(cfg config.Config, p page.Page, layout, outPath string) error {
	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	templatePath := cfg.Template
	if layout != "" {
		if !r.allowlist[layout] {
			slog.Warn("Layout not found in allowlist. Falling back to default", "layout", layout, "default", cfg.Template)
		} else {
			layoutPath := filepath.Join(r.templateDir, layout+".html")
			if _, err := os.Stat(layoutPath); err == nil {
				templatePath = layoutPath
			}
		}
	}

	r.mu.Lock()
	once, onceExists := r.onces[templatePath]
	if !onceExists {
		once = &sync.Once{}
		r.onces[templatePath] = once
	}
	entry, entryExists := r.cache[templatePath]
	if !entryExists {
		entry = &cacheEntry{}
		r.cache[templatePath] = entry
	}
	r.mu.Unlock()

	once.Do(func() {
		partials, err := DiscoverPartials(r.templateDir)
		if err != nil {
			entry.err = fmt.Errorf("partials lookup error: %w", err)
			return
		}

		b, err := os.ReadFile(templatePath)
		if err != nil {
			entry.err = fmt.Errorf("failed to read template: %w", err)
			return
		}

		parsedTmpl := template.New(filepath.Base(templatePath))
		parsedTmpl, err = parsedTmpl.Parse(string(b))
		if err != nil {
			entry.err = fmt.Errorf("failed to parse template: %w", err)
			return
		}

		for name, path := range partials {
			pb, err := os.ReadFile(path)
			if err != nil {
				entry.err = fmt.Errorf("failed to read partial: %w", err)
				return
			}
			_, err = parsedTmpl.New(name).Parse(string(pb))
			if err != nil {
				entry.err = fmt.Errorf("failed to sync partial layout: %w", err)
				return
			}
		}
		entry.tmpl = parsedTmpl
	})

	if entry.err != nil {
		r.mu.Lock()
		delete(r.onces, templatePath)
		delete(r.cache, templatePath)
		r.mu.Unlock()
		return entry.err
	}

	clonedTmpl, err := entry.tmpl.Clone()
	if err != nil {
		return fmt.Errorf("template clone failure: %w", err)
	}

	templateName := filepath.Base(templatePath)
	if cfg.WatchMode {
		var sb strings.Builder
		if err := clonedTmpl.ExecuteTemplate(&sb, templateName, p); err != nil {
			return err
		}

		s := sb.String()
		idx := strings.LastIndex(s, "</body>")
		if idx != -1 {
			var final strings.Builder
			final.Grow(len(s) + 250)
			final.WriteString(s[:idx])
			final.WriteString(`<script>
			if (window.EventSource) {
				var source = new EventSource('/livereload');
				source.onmessage = function(e) { if (e.data === 'reload') window.location.reload(); };
			}
			</script>
</body>`)
			final.WriteString(s[idx+7:])
			_, err = outFile.WriteString(final.String())
			return err
		}
	}

	return clonedTmpl.ExecuteTemplate(outFile, templateName, p)
}
