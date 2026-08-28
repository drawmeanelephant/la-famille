package render

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
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
	templateDir string
	mu          sync.RWMutex
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

	templatePath = filepath.Clean(templatePath)
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
	var buf bytes.Buffer
	if err := clonedTmpl.ExecuteTemplate(&buf, templateName, p); err != nil {
		return err
	}
	rendered := applyBasePath(cfg.BasePath(), buf.Bytes())
	// The live-reload snippet belongs only to `serve --watch`; production
	// builds must not have it injected.
	if cfg.WatchMode {
		return writeWithLiveReload(outFile, rendered)
	}
	_, err = outFile.Write(rendered)
	return err
}

// applyBasePath makes a rendered page deploy correctly under a siteurl subpath
// (for example a GitHub Pages project site at https://user.github.io/repo). The
// bundled themes and hand-written templates emit root-relative URLs (/assets/…,
// /search.json, the "//" home link); on a project site those resolve at the
// domain root and every subresource 404s (#528).
//
// When base is non-empty this prefixes every root-relative URL-bearing
// attribute (href, src, poster, action, and srcset entries) with the base
// path, and injects a <meta name="la-famille-base-path"> so client-side scripts
// can resolve fetch() targets the same way. Absolute and protocol-relative URLs
// are left untouched, as are fragments, query-only targets, data: URLs, and
// values already bearing the prefix.
//
// When base is empty (a root deploy, or siteurl unset) the page passes through
// byte for byte, so this never disturbs the default build output.
func applyBasePath(base string, page []byte) []byte {
	base = strings.TrimSpace(base)
	if base == "" {
		return page
	}
	base = strings.TrimRight(base, "/")
	if base == "" {
		return page
	}

	s := string(page)

	// Scan attribute positions and rebuild only the URLs we rebase, leaving
	// every other byte (tags, attributes, text) untouched.
	var b strings.Builder
	b.Grow(len(s) + 64)
	i := 0
	for i < len(s) {
		eq := strings.IndexByte(s[i:], '=')
		if eq < 0 {
			b.WriteString(s[i:])
			break
		}
		eq += i
		name := attributeName(s[i:eq])
		if !isURLAttr(name) {
			// Not an attribute we rebase. Copy through the '=' and keep going;
			// the brace keeps the scan O(n) rather than re-indexing each time.
			b.WriteString(s[i : eq+1])
			i = eq + 1
			continue
		}
		if eq+1 >= len(s) || s[eq+1] != '"' {
			b.WriteString(s[i : eq+1])
			i = eq + 1
			continue
		}
		end := strings.IndexByte(s[eq+2:], '"') + eq + 2
		if end < eq+2 {
			b.WriteString(s[i:])
			break
		}
		value := s[eq+2 : end]
		rewritten := rebaseAttrValue(name, value, base)
		b.WriteString(s[i : eq+2])
		b.WriteString(rewritten)
		b.WriteString(s[end : end+1])
		i = end + 1
	}

	// Inject the base-path meta for client-side scripts.
	meta := []byte(`<meta name="la-famille-base-path" content="` + base + `">`)
	out := []byte(b.String())
	headIdx := bytes.LastIndex(out, []byte("</head>"))
	if headIdx < 0 {
		// No </head> (a fragment or minimal template): lead with the meta.
		return append(meta, out...)
	}
	dst := make([]byte, 0, len(out)+len(meta))
	dst = append(dst, out[:headIdx]...)
	dst = append(dst, meta...)
	dst = append(dst, out[headIdx:]...)
	return dst
}

// attributeName extracts the attribute name immediately before an '=' by
// stepping back over letters, digits, and - _ : so we skip opening '<' or
// whitespace regardless of attribute spacing.
func attributeName(prefix string) string {
	i := len(prefix) - 1
	for i >= 0 && isNameByte(prefix[i]) {
		i--
	}
	return prefix[i+1:]
}

func isNameByte(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == ':'
}

func isURLAttr(name string) bool {
	switch strings.ToLower(name) {
	case "href", "src", "poster", "action", "srcset", "data-src":
		return true
	default:
		return false
	}
}

// rebaseAttrValue returns the attribute value with each root-relative, single
// URL it names prefixed by base. Simple attributes rewrite directly; srcset
// entries are rewritten one candidate URL at a time.
func rebaseAttrValue(name, value, base string) string {
	if strings.EqualFold(name, "srcset") {
		fields := strings.Fields(value)
		for k, f := range fields {
			if f != "" && !isImageDescriptor(f) {
				fields[k] = rebaseURL(f, base)
			}
		}
		return strings.Join(fields, " ")
	}
	return rebaseURL(value, base)
}

// isImageDescriptor reports whether a srcset token is a width/density descriptor
// ("1024w", "1.5x") rather than a URL. Because srcset fields are space-split,
// a descriptor arrives as its own token and never starts with '/'; that is the
// distinguishing signal instead.
func isImageDescriptor(token string) bool {
	return !strings.HasPrefix(token, "/") && !strings.Contains(token, "//")
}

// rebaseURL prefixes base to a single root-relative URL if it is eligible, and
// reports it unchanged otherwise.
func rebaseURL(value, base string) string {
	if value == "" {
		return value
	}
	for _, p := range []string{"#", "?"} {
		if strings.HasPrefix(value, p) {
			return value
		}
	}
	if !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return value
	}
	lower := strings.ToLower(value)
	for _, scheme := range []string{"http:", "https:", "data:", "javascript:", "mailto:"} {
		if strings.HasPrefix(lower, scheme) {
			return value
		}
	}
	if strings.HasPrefix(value, base) {
		return value
	}
	return base + value
}

const liveReloadScript = `<script>
(function() {
	if (!window.EventSource) return;
	var base = '';
	var el = document.querySelector('meta[name="la-famille-base-path"]');
	if (el) base = el.getAttribute('content') || '';
	var source = new EventSource(base + '/livereload');
	source.onmessage = function(e) { if (e.data === 'reload') window.location.reload(); };
})();
</script>
`

func writeWithLiveReload(w io.Writer, rendered []byte) error {
	marker := []byte("</body>")
	index := bytes.LastIndex(rendered, marker)

	if index < 0 {
		_, err := w.Write(rendered)
		return err
	}

	if bytes.Contains(rendered, []byte("EventSource(")) {
		_, err := w.Write(rendered)
		return err
	}

	if _, err := w.Write(rendered[:index]); err != nil {
		return err
	}
	if _, err := io.WriteString(w, liveReloadScript); err != nil {
		return err
	}
	_, err := w.Write(rendered[index:])
	return err
}
