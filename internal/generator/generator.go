package generator

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/util"

	"github.com/tbuddy/la-famille/internal/asset"
	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/graph"
	"github.com/tbuddy/la-famille/internal/page"
	"github.com/tbuddy/la-famille/internal/render"
	"github.com/tbuddy/la-famille/internal/sitedata"
	"github.com/tbuddy/la-famille/internal/stub"
	"github.com/tbuddy/la-famille/internal/transform"
)

// convertMarkdown is a variable to allow mocking in tests.
var convertMarkdown = func(md goldmark.Markdown, source []byte, w *bytes.Buffer) error {
	return md.Convert(source, w)
}


// BuildResult contains statistics about the build process.
type BuildResult struct {
	Duration   time.Duration
	PageCount  int
	ErrorCount int
}

// Build generates the static site based on the given configuration.
func Build(cfg config.Config) (BuildResult, error) {
	start := time.Now()
	var result BuildResult

	// 1. Pass 1: Walk content dir and gather metadata
	fileMap, err := content.GatherMetadata(cfg.ContentDir)
	if err != nil {
		return result, fmt.Errorf("failed to gather metadata: %w", err)
	}

	// Track missing files that need stubs. map[missingPath][]parentFiles
	missingFiles := make(map[string][]string)
	backlinks := make(map[string][]string)
	g := graph.Graph{
		Nodes: make(map[string]graph.Node),
		Edges: [][2]string{},
	}
	metaData := make(map[string]map[string]interface{})

	// 2. Pass 2: Process files in deterministic order
	var keys []string
	for k := range fileMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Reusable buffer for markdown conversion
	renderer := render.New(filepath.Dir(cfg.Template))

	var errs []error

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

		m := make(map[string]interface{})
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
		if meta.Tags != nil {
			m["tags"] = meta.Tags
		}
		m["word_count"] = len(strings.Fields(string(meta.Rest)))
		metaData[id] = m

		outDirClean := filepath.Clean(cfg.OutputDir)
		outPath := filepath.Join(outDirClean, filepath.FromSlash(relPath))
		if !strings.HasPrefix(outPath, outDirClean+string(filepath.Separator)) && outPath != outDirClean {
			result.ErrorCount++
			log.Printf("Warning: Potential path traversal in page loading detected: %s. Skipping.", relPath)
			continue
		}
		if shouldRender {
			slug := meta.Slug
			if slug != "" {
				if !filepath.IsLocal(slug) || strings.Contains(slug, ".") || strings.Contains(slug, string(filepath.Separator)) || strings.Contains(slug, "/") {
					log.Printf("Warning: Invalid slug %q for %s. Ignoring.", slug, relPath)
					slug = ""
				}
			}
			relOut := transform.GetOutputURL(relPath, slug)
			outPath = filepath.Join(outDirClean, filepath.FromSlash(relOut))
		}

		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return result, err
		}

		if !shouldRender {
			// Just copy the file
			if err := os.WriteFile(outPath, meta.Content, 0644); err != nil {
				return result, err
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
		if err := convertMarkdown(md, meta.Rest, &buf); err != nil {
			result.ErrorCount++
			errs = append(errs, fmt.Errorf("error converting %s: %w", relPath, err))
			continue
		}

		sanitizedHTML := p.SanitizeBytes(buf.Bytes())


		desc := meta.Description
		if desc == "" {
			desc = cfg.DefaultDescription
		}
		img := meta.Image
		if img == "" {
			img = cfg.DefaultOGImage
		}

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
			Description:     desc,
			Image:           img,
		}

		if err := renderer.HTML(cfg, page, meta.Layout, outPath); err != nil {
			return result, err
		}
		result.PageCount++
	}
	if len(errs) > 0 {
		return result, errors.Join(errs...)
	}
	// 3. Generate stubs for missing files in deterministic order
	if err := stub.GenerateStubs(cfg, missingFiles, &g, p, fileMap); err != nil {
		return result, err
	}

	// 4. Verbatim Asset Copy Step
	if err := asset.CopyAssets(cfg); err != nil {
		return result, err
	}

	// 5. Write JSON outputs
	if err := sitedata.Write(cfg.OutputDir, g, backlinks, metaData); err != nil {
		return result, err
	}

	result.Duration = time.Since(start)
	return result, nil
}
