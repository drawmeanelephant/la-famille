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

	"github.com/yuin/goldmark"

	"github.com/tbuddy/la-famille/internal/asset"
	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/discovery"
	"github.com/tbuddy/la-famille/internal/feed"
	"github.com/tbuddy/la-famille/internal/graph"
	"github.com/tbuddy/la-famille/internal/graphexplorer"
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

// buildLocks serializes builds publishing into the same output directory.
// replaceOutputDirectory renames directories in several steps, and the watcher
// starts rebuilds on a timer that cannot cancel a build already in progress, so
// two Build calls can otherwise interleave inside that window.
var (
	buildLocksMu sync.Mutex
	buildLocks   = make(map[string]*sync.Mutex)
)

// lockOutputDir blocks until this process owns the given output directory and
// returns the release function. The lock is process-local: it does not
// serialize two separate la-famille processes writing the same directory.
func lockOutputDir(outputDir string) func() {
	key := filepath.Clean(outputDir)
	if abs, err := filepath.Abs(key); err == nil {
		key = abs
	}

	buildLocksMu.Lock()
	lock, ok := buildLocks[key]
	if !ok {
		lock = &sync.Mutex{}
		buildLocks[key] = lock
	}
	buildLocksMu.Unlock()

	lock.Lock()
	return lock.Unlock
}

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
	Warnings   []string
	Health     ContentHealth
	Duration   time.Duration
	PageCount  int
	ErrorCount int
	CacheHit   bool
}

// Build generates the static site based on the given configuration.
func Build(cfg config.Config) (BuildResult, error) {
	start := time.Now()
	defer lockOutputDir(cfg.OutputDir)()

	outputDir, stagingDir, err := createStagingOutput(cfg.OutputDir)
	if err != nil {
		return BuildResult{}, err
	}

	committed := false
	defer func() {
		if !committed {
			if err := os.RemoveAll(stagingDir); err != nil {
				slog.Warn("Failed to remove build staging directory", "path", stagingDir, "error", err)
			}
		}
	}()
	fingerprint, err := cacheFingerprint(cfg, cfg.ContentDir, filepath.Dir(cfg.Template), cfg.AssetDir, filepath.Join(cfg.ProjectRoot, ".gitignore"))
	if err != nil {
		return BuildResult{}, fmt.Errorf("failed to fingerprint build inputs: %w", err)
	}
	if cache, cacheErr := loadBuildCache(cachePath(cfg)); cacheErr == nil && cacheUsable(cache, cfg.OutputDir, fingerprint) {
		if err := os.RemoveAll(stagingDir); err != nil {
			return BuildResult{}, fmt.Errorf("remove unused build staging directory: %w", err)
		}
		committed = true
		return BuildResult{
			Duration:  time.Since(start),
			PageCount: cache.PageCount,
			CacheHit:  true,
			Health:    cache.Health,
			Warnings:  cache.Warnings,
		}, nil
	}

	stagedCfg := cfg
	stagedCfg.OutputDir = stagingDir
	result, err := build(stagedCfg, cfg)
	if err != nil {
		return result, err
	}

	if err := replaceOutputDirectory(outputDir, stagingDir); err != nil {
		return result, err
	}
	committed = true
	return result, nil
}

func build(cfg, siteCfg config.Config) (BuildResult, error) {
	start := time.Now()
	var result BuildResult

	fingerprint, err := cacheFingerprint(siteCfg, siteCfg.ContentDir, filepath.Dir(siteCfg.Template), siteCfg.AssetDir, filepath.Join(siteCfg.ProjectRoot, ".gitignore"))
	if err != nil {
		return result, fmt.Errorf("failed to fingerprint build inputs: %w", err)
	}
	// 1. Pass 1: Walk content dir and gather metadata
	fileMap, err := content.GatherMetadata(cfg.ContentDir)
	if err != nil {
		return result, fmt.Errorf("failed to gather metadata: %w", err)
	}

	for _, meta := range fileMap {
		if len(meta.Warnings) > 0 {
			result.Warnings = append(result.Warnings, meta.Warnings...)
		}
	}
	sort.Strings(result.Warnings)

	// Track missing files that need stubs. map[missingPath][]parentFiles
	missingFiles := make(map[string][]string)
	backlinks := make(map[string][]string)
	g := graph.Graph{
		Nodes: make(map[string]graph.Node),
		Edges: [][2]string{},
	}
	metaData := make(map[string]map[string]interface{})
	// pageOutputs maps a page id to the output-relative path its HTML was
	// written to, so downstream consumers can build slug-aware public URLs
	// instead of guessing them back from the id.
	pageOutputs := make(map[string]string)
	var searchIndex []search.Item

	// 2. Pass 2: Process files in deterministic order
	keys := make([]string, 0, len(fileMap))
	for k := range fileMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Reusable buffer for markdown conversion
	renderer := render.New(filepath.Dir(cfg.Template))

	type indexedError struct {
		err   error
		index int
	}
	var errs []indexedError

	p := newContentSanitizer()

	taxPaths, taxSearchItems, err := taxonomy.GenerateTaxonomies(cfg, siteCfg, fileMap, renderer, p)
	if err != nil {
		return result, err
	}

	// Taxonomy listings share the output tree with content pages, so the guard
	// can only see the whole picture once their paths are known. claims keeps
	// the ownership map alive for the writers that run later in the build.
	claims, err := validateOutputPaths(fileMap, cfg.OutputDir, reservedOutputPaths(cfg), taxPaths, detectCaseSensitivity(cfg.OutputDir))
	if err != nil {
		return result, err
	}

	var mu sync.Mutex
	numWorkers := runtime.NumCPU()
	if numWorkers < 1 {
		numWorkers = 1
	}

	searchIndexItems := make([]search.Item, len(keys))
	renderedPaths := make([]string, len(keys))
	rssItems := make([]feed.Item, len(keys))

	type job struct {
		relPath string
		index   int
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
						meta      map[string]interface{}
						outputRel string
						errs      []error
						node      graph.Node
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
					if len(meta.Tags) > 0 {
						m["tags"] = meta.Tags
					}
					if len(meta.Categories) > 0 {
						m["categories"] = meta.Categories
					}
					m["word_count"] = len(strings.Fields(string(meta.Rest)))
					m["render"] = shouldRender

					update.meta = m

					defer func() {
						mu.Lock()
						g.Nodes[id] = update.node
						metaData[id] = update.meta
						if update.outputRel != "" {
							pageOutputs[id] = update.outputRel
						}
						if update.errCount > 0 {
							result.ErrorCount += update.errCount
						}
						if update.pageCount > 0 {
							result.PageCount += update.pageCount
						}
						if len(update.errs) > 0 {
							for _, e := range update.errs {
								errs = append(errs, indexedError{err: e, index: idx})
							}
						}
						mu.Unlock()
					}()

					outDirClean := filepath.Clean(cfg.OutputDir)
					outPath := filepath.Join(outDirClean, filepath.FromSlash(relPath))
					var relOut string

					if shouldRender {
						slug := meta.Slug
						if slug != "" && !usableSlug(slug) {
							slog.Warn("Invalid slug. Ignoring.", "slug", slug, "file", relPath)
							slug = ""
						}
						relOut = transform.GetOutputURL(relPath, slug, shouldRender)
						outPath = filepath.Join(outDirClean, filepath.FromSlash(relOut))

						var taxonomyTerms []string
						taxonomySeen := make(map[string]bool)
						for _, tag := range meta.Tags {
							if tag != "" && !taxonomySeen[tag] {
								taxonomySeen[tag] = true
								taxonomyTerms = append(taxonomyTerms, tag)
							}
						}
						for _, cat := range meta.Categories {
							if cat != "" && !taxonomySeen[cat] {
								taxonomySeen[cat] = true
								taxonomyTerms = append(taxonomyTerms, cat)
							}
						}

						// siteCfg, not cfg: cfg is the staging configuration.
						// PublicPathForOutput applies the siteurl base path and
						// drops index.html, so a search hit navigates to the
						// same URL the canonical link and sitemap advertise.
						urlPath := siteCfg.PublicPathForOutput(relOut)
						searchIndexItems[idx] = search.Item{
							Title:    title,
							URL:      urlPath,
							Tags:     taxonomyTerms,
							Snippet:  search.ExtractSnippet(meta.Rest),
							Headings: search.ExtractHeadings(meta.Rest),
						}
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

					// Set up goldmark with AST transformer. Unrendered pages are
					// converted too: the conversion is what walks their links
					// into the graph, the backlinks and the missing-file list.
					// Their generated HTML is then discarded.
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
					convertErr := getConvertMarkdown()(md, meta.Rest, &buf)

					if !shouldRender {
						// The verbatim copy does not depend on the conversion,
						// so a conversion failure here costs graph edges, not
						// output, and must not fail a build that previously
						// succeeded.
						if convertErr != nil {
							slog.Warn("Failed to scan links in unrendered page", "file", relPath, "error", convertErr)
						}
						if err := os.WriteFile(outPath, meta.Content, 0600); err != nil {
							update.errs = append(update.errs, err)
						}
						return
					}

					if convertErr != nil {
						update.errCount++
						update.errs = append(update.errs, fmt.Errorf("error converting %s: %w", relPath, convertErr))
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
						Site:            siteCfg,
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
						CanonicalURL:    cfg.URLForOutputPath(relOut),
					}

					if err := renderer.HTML(cfg, page, meta.Layout, outPath); err != nil {
						update.errs = append(update.errs, err)
						return
					}
					renderedPaths[idx] = relOut
					update.outputRel = relOut
					if meta.Date != "" {
						itemURL := cfg.URLForOutputPath(relOut)
						if itemURL == "" {
							itemURL = feed.LocalURL(relOut)
						}
						rssItems[idx] = feed.Item{
							Title:       title,
							URL:         itemURL,
							Date:        meta.Date,
							Description: search.ExtractSnippet(meta.Rest),
						}
					}
					update.pageCount++
				}()
			}
		}()
	}
	wg.Wait()

	for _, tp := range taxPaths {
		if tp != "" {
			renderedPaths = append(renderedPaths, tp)
		}
	}
	result.PageCount += len(taxPaths)

	for _, item := range searchIndexItems {
		if item.URL != "" {
			searchIndex = append(searchIndex, item)
		}
	}
	for _, item := range taxSearchItems {
		if item.URL != "" {
			searchIndex = append(searchIndex, item)
		}
	}

	// Sort searchIndex, edges, and other outputs to ensure deterministic output
	sort.SliceStable(searchIndex, func(i, j int) bool {
		if searchIndex[i].URL != searchIndex[j].URL {
			return searchIndex[i].URL < searchIndex[j].URL
		}
		return searchIndex[i].Title < searchIndex[j].Title
	})

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
	// 3. Generate stubs for missing files in deterministic order. Stubs run
	// last and write with os.Create, so without claiming their paths a dangling
	// link such as [all tags](tags.md) would overwrite the generated taxonomy
	// listing at exit 0.
	if err := stub.GenerateStubs(cfg, siteCfg, missingFiles, &g, p, fileMap, claims.stubClaimer()); err != nil {
		return result, err
	}

	// 4. Verbatim Asset Copy Step
	if err := asset.CopyAssets(cfg, claims.assetClaimer()); err != nil {
		return result, err
	}

	// Write graph structures via internal/graph
	// 5. Write JSON outputs
	if err := graph.WriteGraphFiles(cfg.OutputDir, g, backlinks); err != nil {
		return result, err
	}

	// Record each rendered page's public URL alongside its other metadata.
	// Consumers previously had to reconstruct it from the page id, which cannot
	// know about a frontmatter slug or a siteurl base path and so produced links
	// to pages that were never published there. This is the same pair the graph
	// explorer already uses, and it is additive: a new key inside an existing
	// object, leaving every reader of title/date/tags untouched. Raw
	// render:false pages and stubs get none, which is correct — they have no
	// published URL to cite.
	for id, out := range pageOutputs {
		if m := metaData[id]; m != nil && out != "" {
			m["url"] = siteCfg.PublicPathForOutput(out)
		}
	}

	if err := sitedata.Write(cfg.OutputDir, metaData); err != nil {
		return result, err
	}

	// 5b. Knowledge Graph Explorer page and payload (static, no extra deps).
	if _, err := graphexplorer.Write(graphexplorer.Input{
		Config:      cfg,
		Graph:       g,
		Meta:        metaData,
		PageOutputs: pageOutputs,
	}); err != nil {
		return result, err
	}

	if err := search.WriteMinifiedJSON(filepath.Join(cfg.OutputDir, "search.json"), searchIndex); err != nil {
		return result, err
	}
	var datedItems []feed.Item
	for _, item := range rssItems {
		if item.URL != "" {
			datedItems = append(datedItems, item)
		}
	}
	if err := feed.Write(cfg, datedItems); err != nil {
		return result, err
	}
	result.Health = ComputeContentHealth(fileMap, g, backlinks)

	if err := discovery.Write(cfg, renderedPaths); err != nil {
		return result, err
	}

	files, err := generatedFiles(cfg.OutputDir)
	if err != nil {
		return result, fmt.Errorf("failed to collect generated files: %w", err)
	}
	// Asset, stub, and page claims may have produced non-fatal case-only
	// warnings during the passes above; surface them alongside the metadata
	// warnings.
	result.Warnings = append(result.Warnings, claims.Warnings()...)
	sort.Strings(result.Warnings)
	if err := writeBuildCache(cachePath(siteCfg), fingerprint, files, result.PageCount, result.Health, result.Warnings); err != nil {
		return result, fmt.Errorf("failed to write build cache: %w", err)
	}

	result.Duration = time.Since(start)
	return result, nil
}

// reservedOutputPaths lists files the build generates itself, mapped to a
// description of what owns them. These are written after the content render
// pass, so a page rendering to one of them would be silently overwritten with
// no error and no missing-page report. Treating them as collisions up front
// turns that into an actionable build failure.
func reservedOutputPaths(cfg config.Config) map[string]string {
	reserved := make(map[string]string)
	if cfg.GraphExplorer {
		owner := "the knowledge graph explorer (disable with graph_explorer: false)"
		reserved[filepath.Clean(graphexplorer.IndexPath(cfg.OutputDir))] = owner
		reserved[filepath.Clean(graphexplorer.DataPath(cfg.OutputDir))] = owner
	}
	return reserved
}

// usableSlug reports whether a frontmatter slug can name an output directory.
// A slug that fails this check is discarded when the page is rendered, so every
// caller that predicts an output path has to discard it the same way, or the
// prediction names a file the generator never writes.
func usableSlug(slug string) bool {
	return transform.IsUsableSlug(slug)
}

// outputOwner is a single writer's claim on a path in the output tree.
type outputOwner struct {
	// source describes the writer, ready to drop into an error message.
	source string
	// relOut is the output-relative path as that writer spells it. Errors
	// report this rather than the absolute target, which during a build is a
	// throwaway staging path that tells the author nothing about their site.
	relOut string
}

// outputClaims records which writer owns each path in the output tree. Every
// writer that emits into that tree - content pages, taxonomy listings, the
// knowledge graph explorer and the missing-page stubs - claims its paths here,
// so a second writer aiming at the same file is caught rather than silently
// overwriting the first.
//
// Keys are case-folded, because macOS and Windows collapse paths differing only
// in case onto one file. Whether the fold is enforced depends on the output
// filesystem: exact duplicates always collide, while two paths differing only
// in case are refused when that filesystem is case-insensitive and admitted
// with a warning when it is case-sensitive.
type outputClaims struct {
	owners        map[string][]outputOwner
	outputDir     string
	caseSensitive bool
	warnings      []string
	mu            sync.Mutex
}

func newOutputClaims(outputDir string, size int, caseSensitive bool) *outputClaims {
	return &outputClaims{
		outputDir:     outputDir,
		caseSensitive: caseSensitive,
		owners:        make(map[string][]outputOwner, size),
	}
}

// detectCaseSensitivity reports whether the output filesystem collapses paths
// differing only in case. It is a variable so tests can pin either branch on
// any host.
var detectCaseSensitivity = probeCaseSensitivity

// probeCaseSensitivity probes dir with one file: it creates a probe file and
// asks whether a differently-cased spelling of its name names the same file.
// A spelling that resolves to its own file means the filesystem is
// case-sensitive. When the probe cannot run it conservatively reports
// case-insensitive, preserving the historical behavior of refusing case-only
// collisions.
func probeCaseSensitivity(dir string) bool {
	probe := filepath.Join(dir, fmt.Sprintf("lafamille-case-probe-%d", os.Getpid()))
	f, err := os.Create(probe)
	if err != nil {
		return false
	}
	_ = f.Close()
	defer func() { _ = os.Remove(probe) }()

	_, err = os.Stat(filepath.Join(dir, flipProbeCase(filepath.Base(probe))))
	return errors.Is(err, os.ErrNotExist)
}

// flipProbeCase toggles the case of the first ASCII letter in s, so the probe
// name "lafamille-..." is re-checked as "Lafamille-...".
func flipProbeCase(s string) string {
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] >= 'a' && s[i] <= 'z':
			return s[:i] + string(s[i]-('a'-'A')) + s[i+1:]
		case s[i] >= 'A' && s[i] <= 'Z':
			return s[:i] + string(s[i]+('a'-'A')) + s[i+1:]
		}
	}
	return s
}

func (c *outputClaims) absPath(relOut string) string {
	return filepath.Clean(filepath.Join(c.outputDir, filepath.FromSlash(relOut)))
}

func (c *outputClaims) key(relOut string) string {
	return strings.ToLower(c.absPath(relOut))
}

// claim reserves relOut for source. It returns the writer that already holds
// the path, and false, when the path is taken. An exact duplicate is always
// refused; a pair differing only in case is refused on case-insensitive
// filesystems and admitted with a warning on case-sensitive ones.
func (c *outputClaims) claim(source, relOut string) (outputOwner, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.key(relOut)
	var firstCaseDiff outputOwner
	haveCaseDiff := false
	for _, previous := range c.owners[key] {
		if previous.relOut == relOut {
			return previous, false
		}
		if !haveCaseDiff {
			firstCaseDiff = previous
			haveCaseDiff = true
		}
	}
	if haveCaseDiff && !c.caseSensitive {
		return firstCaseDiff, false
	}
	if haveCaseDiff {
		c.warnings = append(c.warnings, fmt.Sprintf(
			"output path warning: %s maps to %q and %s maps to %q, which would be the same file on a case-insensitive filesystem",
			firstCaseDiff.source, firstCaseDiff.relOut, source, relOut))
	}
	c.owners[key] = append(c.owners[key], outputOwner{source: source, relOut: relOut})
	return outputOwner{}, true
}

// Warnings returns the non-fatal collision notices collected while claiming.
func (c *outputClaims) Warnings() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, len(c.warnings))
	copy(out, c.warnings)
	return out
}

// stubClaimer adapts the registry to the reservation callback internal/stub
// uses. Paths are claimed by their output-relative form, so the output
// directory itself is not needed here.
// assetClaimer lets the asset copier reserve its destinations in the same
// registry the pages use. Assets are copied after rendering, so without this an
// asset quietly replaces a page that renders to the same path.
func (c *outputClaims) assetClaimer() asset.ClaimOutput {
	return func(relOut string) error {
		source := fmt.Sprintf("the asset %q", relOut)
		previous, ok := c.claim(source, relOut)
		if ok {
			return nil
		}
		// The shared builder, so an asset clash is described the same way a
		// page clash is — including the case-only wording, which the asset
		// path used to get wrong by claiming both wrote the same file.
		return collisionError(previous, source, relOut)
	}
}

func (c *outputClaims) stubClaimer() stub.ClaimOutput {
	return func(missingRelPath, relOut string) (string, bool) {
		source := fmt.Sprintf("the generated stub for %q", missingRelPath)
		previous, ok := c.claim(source, relOut)
		if ok {
			return "", true
		}
		return previous.source, false
	}
}

func collisionError(previous outputOwner, source, relOut string) error {
	if previous.relOut == relOut {
		return fmt.Errorf("output path collision: %s and %s both map to %q", previous.source, source, relOut)
	}
	return fmt.Errorf(
		"output path collision: %s maps to %q and %s maps to %q, which are the same file on case-insensitive filesystems",
		previous.source, previous.relOut, source, relOut,
	)
}

// validateOutputPaths claims every output path known before the render pass and
// returns the registry so later writers can claim theirs too.
func validateOutputPaths(fileMap map[string]*content.FileMeta, outputDir string, reserved map[string]string, taxonomyPaths []string, caseSensitive bool) (*outputClaims, error) {
	claims := newOutputClaims(outputDir, len(fileMap)+len(taxonomyPaths)+len(reserved), caseSensitive)

	// Content pages claim first so that an author's own file is the first party
	// named in any error and the generated owner that follows explains itself.
	//
	// Sort so a site with several collisions always reports the same one,
	// rather than whichever the map happened to yield first.
	relPaths := make([]string, 0, len(fileMap))
	for relPath := range fileMap {
		relPaths = append(relPaths, relPath)
	}
	sort.Strings(relPaths)

	for _, relPath := range relPaths {
		meta := fileMap[relPath]
		if meta.Render != nil && !*meta.Render {
			// Unrendered pages are copied verbatim to their own .md path, which
			// no renderer can reach: GatherMetadata only walks .md files while
			// every rendered output is .html.
			continue
		}

		// The worker discards an unusable slug before computing the real output
		// path, so predicting with the raw slug names a file that is never
		// written - and misses the collision that really happens.
		slug := meta.Slug
		if slug != "" && !usableSlug(slug) {
			slug = ""
		}

		relOut := transform.GetOutputURL(relPath, slug, true)
		source := fmt.Sprintf("%q", relPath)
		if previous, ok := claims.claim(source, relOut); !ok {
			return nil, collisionError(previous, source, relOut)
		}
	}

	// Paths the build reserves for itself, such as the knowledge graph explorer.
	reservedTargets := make([]string, 0, len(reserved))
	for target := range reserved {
		reservedTargets = append(reservedTargets, target)
	}
	sort.Strings(reservedTargets)
	for _, target := range reservedTargets {
		rel, err := filepath.Rel(filepath.Clean(outputDir), target)
		if err != nil {
			return nil, fmt.Errorf("resolve reserved output path %q: %w", target, err)
		}
		relOut := filepath.ToSlash(rel)
		if previous, ok := claims.claim(reserved[target], relOut); !ok {
			return nil, collisionError(previous, reserved[target], relOut)
		}
	}

	// Taxonomy listings are written before the content workers start, so a
	// content page mapping onto one of them overwrites it without a warning.
	seen := make(map[string]bool, len(taxonomyPaths))
	for _, relOut := range taxonomyPaths {
		if relOut == "" || seen[relOut] {
			continue
		}
		seen[relOut] = true
		const source = "the generated taxonomy page"
		if previous, ok := claims.claim(source, relOut); !ok {
			return nil, collisionError(previous, source, relOut)
		}
	}

	return claims, nil
}

// createStagingOutput creates an empty sibling directory for a build. Keeping the
// staging directory beside the final output means os.Rename can replace the
// completed site without a cross-filesystem copy.
func createStagingOutput(configuredOutput string) (string, string, error) {
	cleanOutput := filepath.Clean(configuredOutput)
	if cleanOutput == "." || cleanOutput == string(filepath.Separator) {
		return "", "", fmt.Errorf("output directory must not be the current directory or filesystem root")
	}
	if !filepath.IsAbs(cleanOutput) && !filepath.IsLocal(cleanOutput) {
		return "", "", fmt.Errorf("output directory must be a local path: %q", configuredOutput)
	}

	outputDir, err := filepath.Abs(cleanOutput)
	if err != nil {
		return "", "", fmt.Errorf("resolve output directory: %w", err)
	}
	parentDir := filepath.Dir(outputDir)
	if parentDir == outputDir {
		return "", "", fmt.Errorf("output directory must not be filesystem root")
	}
	if err := os.MkdirAll(parentDir, 0700); err != nil {
		return "", "", fmt.Errorf("create output parent directory: %w", err)
	}

	if info, err := os.Lstat(outputDir); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return "", "", fmt.Errorf("output directory must not be a symlink: %q", configuredOutput)
		}
		if !info.IsDir() {
			return "", "", fmt.Errorf("output path is not a directory: %q", configuredOutput)
		}
	} else if !os.IsNotExist(err) {
		return "", "", fmt.Errorf("inspect output directory: %w", err)
	}

	stagingDir, err := os.MkdirTemp(parentDir, "."+filepath.Base(outputDir)+".staging-")
	if err != nil {
		return "", "", fmt.Errorf("create build staging directory: %w", err)
	}
	return outputDir, stagingDir, nil
}

// replaceOutputDirectory swaps a completed staging tree into place. The previous
// output is renamed rather than deleted first so a failed replacement can be
// restored without exposing a partially generated site.
func replaceOutputDirectory(outputDir, stagingDir string) error {
	parentDir := filepath.Dir(outputDir)
	if filepath.Dir(stagingDir) != parentDir {
		return fmt.Errorf("staging directory must be a sibling of the output directory")
	}

	outputExists := false
	if info, err := os.Lstat(outputDir); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("output directory changed while building: %q", outputDir)
		}
		outputExists = true
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect output directory before replacement: %w", err)
	}

	backupDir, err := os.MkdirTemp(parentDir, "."+filepath.Base(outputDir)+".previous-")
	if err != nil {
		return fmt.Errorf("create output backup path: %w", err)
	}
	if err := os.Remove(backupDir); err != nil {
		return fmt.Errorf("prepare output backup path: %w", err)
	}

	if outputExists {
		if err := os.Rename(outputDir, backupDir); err != nil {
			return fmt.Errorf("move existing output aside: %w", err)
		}
	}

	if err := os.Rename(stagingDir, outputDir); err != nil {
		if !outputExists {
			return fmt.Errorf("replace output directory: %w", err)
		}
		if restoreErr := os.Rename(backupDir, outputDir); restoreErr != nil {
			return fmt.Errorf("replace output directory: %w; restore previous output: %v", err, restoreErr)
		}
		return fmt.Errorf("replace output directory: %w", err)
	}

	if outputExists {
		if err := os.RemoveAll(backupDir); err != nil {
			slog.Warn("Failed to remove replaced output directory", "path", backupDir, "error", err)
		}
	}
	return nil
}
