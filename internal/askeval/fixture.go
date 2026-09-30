package askeval

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/generator"
	"github.com/tbuddy/la-famille/internal/graph"
	"github.com/tbuddy/la-famille/internal/ragexport"
	"github.com/tbuddy/la-famille/internal/retrieval"
)

func prepareSite(projectRoot, destination string, site Site) (config.Config, error) {
	project, err := os.OpenRoot(projectRoot)
	if err != nil {
		return config.Config{}, err
	}
	defer project.Close()
	// OpenRoot follows only links that stay inside the project, including
	// when resolving the source directory itself. Content reads remain
	// confined after that directory is opened, even during concurrent edits.
	source, err := project.OpenRoot(filepath.Join(site.Fixture, site.ContentDir))
	if err != nil {
		return config.Config{}, fmt.Errorf("open fixture content: %w", err)
	}
	defer source.Close()
	if err := copyContent(source.FS(), filepath.Join(destination, "content")); err != nil {
		return config.Config{}, fmt.Errorf("copy content: %w", err)
	}
	templateDir := filepath.Join(destination, "templates")
	if err := os.MkdirAll(templateDir, 0700); err != nil {
		return config.Config{}, err
	}
	layout := filepath.Join(templateDir, "layout.html")
	if err := os.WriteFile(layout, []byte("<!doctype html><html><body><h1>{{.Title}}</h1>{{.Content}}</body></html>"), 0600); err != nil {
		return config.Config{}, err
	}
	cfg := config.DefaultConfig()
	cfg.ProjectRoot = destination
	cfg.ContentDir = filepath.Join(destination, "content")
	cfg.OutputDir = filepath.Join(destination, "public")
	cfg.RagDir = filepath.Join(destination, "rag-archive")
	cfg.AssetDir = filepath.Join(destination, "assets")
	cfg.Template = layout
	cfg.GraphExplorer = false
	if err := cfg.ValidateResolved(); err != nil {
		return config.Config{}, err
	}
	if _, err := generator.Build(cfg); err != nil {
		return config.Config{}, fmt.Errorf("build: %w", err)
	}
	if err := ragexport.RunExport(cfg); err != nil {
		return config.Config{}, fmt.Errorf("export: %w", err)
	}
	// Content-only projects have no repository source. Omit empty optional
	// bundles rather than weakening the loader or inventing fixture evidence.
	for _, name := range []string{"rag-system.md", "rag-config.md"} {
		path := filepath.Join(cfg.RagDir, name)
		info, err := os.Stat(path)
		if err != nil {
			return config.Config{}, err
		}
		if info.Size() == 0 {
			if err := os.Remove(path); err != nil {
				return config.Config{}, err
			}
		}
	}
	return cfg, nil
}

// copyContent rejects stable file/directory symlinks (including internal ones)
// to avoid importing duplicate evidence. The supplied root FS also confines
// reads if a link is introduced after the walk, preventing external reads.
func copyContent(source fs.FS, destination string) error {
	if err := fs.WalkDir(source, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("fixture content contains symlink %q", path)
		}
		return nil
	}); err != nil {
		return err
	}
	return os.CopyFS(destination, source)
}

func validateLabels(q Question, corpus retrieval.Corpus, g graph.Graph) error {
	pages := make(map[string]bool)
	for _, ch := range corpus.Chunks {
		pages[ch.PageID] = true
	}
	for _, page := range append(slices.Clone(q.AcceptablePages), q.PathPages...) {
		if !pages[page] {
			return fmt.Errorf("question %s labels page %q absent from published corpus", q.ID, page)
		}
	}
	adj := graph.Adjacency(g)
	for i := 1; i < len(q.PathPages); i++ {
		a, b := q.PathPages[i-1], q.PathPages[i]
		if !slices.Contains(adj[a].Outbound, b) && !slices.Contains(adj[a].Inbound, b) {
			return fmt.Errorf("question %s labels nonexistent graph hop %s -> %s", q.ID, a, b)
		}
	}
	return nil
}
