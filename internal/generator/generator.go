package generator

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"

	"github.com/tbuddy/la-famille/internal/asset"
	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/graph"
	"github.com/tbuddy/la-famille/internal/markdown"
	"github.com/tbuddy/la-famille/internal/page"
	"github.com/tbuddy/la-famille/internal/pathutil"
	"github.com/tbuddy/la-famille/internal/render"
	"github.com/tbuddy/la-famille/internal/search"
	"github.com/tbuddy/la-famille/internal/sitedata"
	"github.com/tbuddy/la-famille/internal/stub"
	"github.com/tbuddy/la-famille/internal/taxonomy"
	"github.com/tbuddy/la-famille/internal/transform"
)

// convertMarkdown is a variable to allow mocking in tests.
var (
	convertMu       sync.RWMutex
	convertMarkdown = func(md goldmark.Markdown, source []byte, w *bytes.Buffer) error {
		return md.Convert(source, w)
	}
)

func getConvertMarkdown() func(goldmark.Markdown, []byte, *bytes.Buffer) error {
	convertMu.RLock()
	defer convertMu.RUnlock()
	return convertMarkdown
}

func setConvertMarkdown(fn func(goldmark.Markdown, []byte, *bytes.Buffer) error) {
	convertMu.Lock()
	defer convertMu.Unlock()
	convertMarkdown = fn
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
	var searchIndex []search.Item

	// 2. Pass 2: Process files in deterministic order
	if err := validateOutputPaths(fileMap, cfg.OutputDir); err != nil {
		return result, err
	}
	keys := make([]string, 0, len(fileMap))
	for k := range fileMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Reusable buffer for markdown conversion
	renderer := render.New(filepath.Dir(cfg.Template))

	type indexedError struct {
		index int
		err   error
	}
	var errs []indexedError

	p := bluemonday.UGCPolicy()
	p.AllowAttrs("class").Globally()
	p.AllowElements("svg", "path")
	p.AllowAttrs("xmlns", "fill", "viewBox", "stroke-linecap", "stroke-linejoin", "stroke-width", "d", "stroke", "class").OnElements("svg", "path")

	if err := taxonomy.GenerateTags(cfg, fileMap, renderer, p); err != nil {
		return result, err
	}

	var mu sync.Mutex
	numWorkers := runtime.NumCPU()
	if numWorkers < 1 {
		numWorkers = 1
	}

	searchIndexItems := make([]search.Item, len(keys))

	type job struct {
		index   int
		relPath string
	}

	jobs := make(chan job, len(keys))
	for i, k := range keys {
		jobs <- job{index: i, relPath: k}
	}
	close(jobs)

	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var buf bytes.Buffer
			for j := range jobs {
				func() {
					type jobUpdate struct {
						node      graph.Node
						meta      map[string]interface{}
						errs      []error
						errCount  int
						pageCount int
					}
					var update jobUpdate

					relPath := j.relPath
					idx := j.index
					meta := fileMap[relPath]
					shouldRender := true
					if meta.Render != nil && !*meta.Render {
						shouldRender = false
					}

					id := strings.TrimSuffix(relPath, ".md")
					if !shouldRender {
						id = relPath
					}

					update.node = graph.Node{
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
					m["render"] = shouldRender

					update.meta = m

					defer func() {
						mu.Lock()
						g.Nodes[id] = update.node
						metaData[id] = update.meta
						if update.errCount > 0 {
							result.ErrorCount += update.errCount
						}
						if update.pageCount > 0 {
							result.PageCount += update.pageCount
						}
						if len(update.errs) > 0 {
							for _, e := range update.errs {
								errs = append(errs, indexedError{idx, e})
							}
						}
						mu.Unlock()
					}()

					if shouldRender {
						urlOut := transform.GetOutputURL(relPath, meta.Slug, shouldRender)
						urlPath := "/" + filepath.ToSlash(urlOut)

						searchIndexItems[idx] = search.Item{
							Title:   title,
							URL:     urlPath,
							Tags:    meta.Tags,
							Snippet: search.ExtractSnippet(meta.Rest),
						}
					}

					outDirClean := filepath.Clean(cfg.OutputDir)
					outPath := filepath.Join(outDirClean, filepath.FromSlash(relPath))

					if shouldRender {
						slug := meta.Slug
						if slug != "" {
							if !filepath.IsLocal(slug) || strings.Contains(slug, ".") || strings.Contains(slug, string(filepath.Separator)) || strings.Contains(slug, "/") {
								slog.Warn("Invalid slug. Ignoring.", "slug", slug, "file", relPath)
								slug = ""
							}
						}
						relOut := transform.GetOutputURL(relPath, slug, shouldRender)
						outPath = filepath.Join(outDirClean, filepath.FromSlash(relOut))
					}

					// Validate the final outPath against directory escapes using IsSafePath
					if !pathutil.IsSafePath(outDirClean, outPath) {
						update.errCount++
						slog.Warn("Potential path traversal in page loading detected. Skipping.", "path", outPath)
						return
					}

					if err := os.MkdirAll(filepath.Dir(outPath), 0700); err != nil {
						update.errs = append(update.errs, err)
						return
					}

					if !shouldRender {
						// Just copy the file
						if err := os.WriteFile(outPath, meta.Content, 0600); err != nil {
							update.errs = append(update.errs, err)
						}
						return
					}

					// Set up goldmark with AST transformer
					transformer := &transform.LinkTransformer{
						CurrentFile:  relPath,
						FileMap:      fileMap,
						MissingFiles: missingFiles,
						Backlinks:    backlinks,
						Graph:        &g,
						Mu:           &mu,
					}

					md := markdown.NewEngine(transformer)

					buf.Reset()
					if err := getConvertMarkdown()(md, meta.Rest, &buf); err != nil {
						update.errCount++
						update.errs = append(update.errs, fmt.Errorf("error converting %s: %w", relPath, err))
						return
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
						ComplianceModal: meta.ComplianceModal,
						Content:         template.HTML(sanitizedHTML), // #nosec G203
						Description:     desc,
						Image:           img,
					}

					if err := renderer.HTML(cfg, page, meta.Layout, outPath); err != nil {
						update.errs = append(update.errs, err)
						return
					}
					update.pageCount++
				}()
			}
		}()
	}
	wg.Wait()

	for _, item := range searchIndexItems {
		if item.URL != "" {
			searchIndex = append(searchIndex, item)
		}
	}

	// Sort searchIndex, edges, and other outputs to ensure deterministic output
	sort.SliceStable(g.Edges, func(i, j int) bool {
		return g.Edges[i][0] < g.Edges[j][0]
	})

	for k := range backlinks {
		sort.Strings(backlinks[k])
	}

	// Sort errs for deterministic order
	if len(errs) > 0 {
		sort.SliceStable(errs, func(i, j int) bool {
			return errs[i].index < errs[j].index
		})

		var joinErrs []error
		for _, ie := range errs {
			joinErrs = append(joinErrs, ie.err)
		}
		return result, errors.Join(joinErrs...)
	}
	// 3. Generate stubs for missing files in deterministic order
	if err := stub.GenerateStubs(cfg, missingFiles, &g, p, fileMap); err != nil {
		return result, err
	}

	// 4. Verbatim Asset Copy Step
	if err := asset.CopyAssets(cfg); err != nil {
		return result, err
	}

	// Write graph structures via internal/graph
	// 5. Write JSON outputs
	if err := graph.WriteGraphFiles(cfg.OutputDir, g, backlinks); err != nil {
		return result, err
	}

	if err := sitedata.Write(cfg.OutputDir, metaData); err != nil {
		return result, err
	}

	if err := search.WriteMinifiedJSON(filepath.Join(cfg.OutputDir, "search.json"), searchIndex); err != nil {
		return result, err
	}

	result.Duration = time.Since(start)
	return result, nil
}

func validateOutputPaths(fileMap map[string]*content.FileMeta, outputDir string) error {
	owners := make(map[string]string, len(fileMap))

	for relPath, meta := range fileMap {
		if meta.Render != nil && !*meta.Render {
			continue
		}

		relOut := transform.GetOutputURL(relPath, meta.Slug, true)
		target := filepath.Clean(filepath.Join(
			outputDir,
			filepath.FromSlash(relOut),
		))

		if previous, exists := owners[target]; exists {
			return fmt.Errorf("output path collision: %q and %q both map to %q", previous, relPath, target)
		}
		owners[target] = relPath
	}

	return nil
}
