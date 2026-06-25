package generator

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/util"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/graph"
	"github.com/tbuddy/la-famille/internal/jsonutil"
	"github.com/tbuddy/la-famille/internal/page"
	"github.com/tbuddy/la-famille/internal/render"
	"github.com/tbuddy/la-famille/internal/stub"
	"github.com/tbuddy/la-famille/internal/transform"
)

// Build generates the static site based on the given configuration.
func Build(cfg config.Config) error {
	// 1. Pass 1: Walk content dir and gather metadata
	fileMap, err := content.GatherMetadata(cfg.ContentDir)
	if err != nil {
		return fmt.Errorf("failed to gather metadata: %w", err)
	}

	// Track missing files that need stubs. map[missingPath][]parentFiles
	missingFiles := make(map[string][]string)
	backlinks := make(map[string][]string)
	g := graph.Graph{
		Nodes: make(map[string]graph.Node),
		Edges: [][2]string{},
	}
	metaData := make(map[string]map[string]string)

	// 2. Pass 2: Process files in deterministic order
	var keys []string
	for k := range fileMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Reusable buffer for markdown conversion
	var buf bytes.Buffer

	p := bluemonday.UGCPolicy()
	for _, relPath := range keys {
		meta := fileMap[relPath]
		shouldRender := true
		if meta.Render != nil && !*meta.Render {
			shouldRender = false
		}

		id := strings.TrimSuffix(relPath, ".md")
		g.Nodes[id] = graph.Node{
			Type:   "page",
			Render: shouldRender,
		}

		m := make(map[string]string)
		title := meta.Title
		if title == "" {
			title = filepath.Base(relPath)
		}
		m["title"] = title
		if meta.Author != "" {
			m["author"] = meta.Author
		}
		if meta.Date != "" {
			m["date"] = meta.Date
		}
		metaData[id] = m

		outPath := filepath.Join(cfg.OutputDir, filepath.FromSlash(relPath))
		if shouldRender {
			outPath = strings.TrimSuffix(outPath, filepath.Ext(outPath)) + ".html"
		}

		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return err
		}

		if !shouldRender {
			// Just copy the file
			if err := os.WriteFile(outPath, meta.Content, 0644); err != nil {
				return err
			}
			continue
		}

		// Set up goldmark with AST transformer
		transformer := &transform.LinkTransformer{
			CurrentFile:  relPath,
			FileMap:      fileMap,
			MissingFiles: missingFiles,
			Backlinks:    backlinks,
			Graph:        &g,
		}

		md := goldmark.New(
			goldmark.WithParserOptions(
				parser.WithASTTransformers(
					util.Prioritized(transformer, 100),
				),
			),
		)

		buf.Reset()
		if err := md.Convert(meta.Rest, &buf); err != nil {
			log.Printf("Error converting %s: %v", relPath, err)
			continue
		}

		sanitizedHTML := p.SanitizeBytes(buf.Bytes())

		page := page.Page{
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

		if err := render.HTML(cfg, page, meta.Layout, outPath); err != nil {
			return err
		}
	}
	// 3. Generate stubs for missing files in deterministic order
	if err := stub.GenerateStubs(cfg, missingFiles, &g, p); err != nil {
		return err
	}

	// 4. Verbatim Asset Copy Step
	if cfg.AssetDir != "" {
		if _, err := os.Stat(cfg.AssetDir); err == nil {
			err = filepath.WalkDir(cfg.AssetDir, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}

				// Skip testdata subdirectories
				if d.IsDir() && d.Name() == "testdata" {
					return filepath.SkipDir
				}

				if d.IsDir() {
					return nil
				}

				relPath, err := filepath.Rel(cfg.AssetDir, path)
				if err != nil {
					return err
				}

				destPath := filepath.Join(cfg.OutputDir, "assets", relPath)
				if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
					return err
				}

				input, err := os.ReadFile(path)
				if err != nil {
					return err
				}

				return os.WriteFile(destPath, input, 0644)
			})
			if err != nil {
				return fmt.Errorf("failed to copy assets: %w", err)
			}
		}
	}

	// 5. Write JSON outputs
	for _, parents := range backlinks {
		sort.Strings(parents)
	}
	if err := jsonutil.WriteJSON(filepath.Join(cfg.OutputDir, "graph.json"), g); err != nil {
		return err
	}
	if err := jsonutil.WriteJSON(filepath.Join(cfg.OutputDir, "backlinks.json"), backlinks); err != nil {
		return err
	}
	if err := jsonutil.WriteJSON(filepath.Join(cfg.OutputDir, "meta.json"), metaData); err != nil {
		return err
	}

	return nil
}
