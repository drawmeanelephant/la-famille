package checker

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/adrg/frontmatter"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"

	"github.com/tbuddy/la-famille/internal/asset"
	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/markdown"
	"github.com/tbuddy/la-famille/internal/pathutil"
	"github.com/tbuddy/la-famille/internal/runtimeassets"
	"github.com/tbuddy/la-famille/internal/sitedata"
	"github.com/tbuddy/la-famille/internal/transform"
)

type Level string

const (
	LevelError Level = "ERROR"
	LevelWarn  Level = "WARN"
)

// Categories for findings — used to feed the publish-check summary footer (#483).
const (
	CategoryBrokenLink      = "broken_links"
	CategoryMissingMetadata = "missing_metadata"
	CategoryAssetHealth     = "asset_health"
	CategoryOrphan          = "orphan"
)

type Finding struct {
	File     string
	Level    Level
	Message  string
	Line     int
	Category string
}

func (f Finding) String() string {
	if f.Line > 0 {
		return fmt.Sprintf("[%s] %s:%d: %s", f.Level, f.File, f.Line, f.Message)
	}
	if f.File != "" {
		return fmt.Sprintf("[%s] %s: %s", f.Level, f.File, f.Message)
	}
	return fmt.Sprintf("[%s] %s", f.Level, f.Message)
}

type Result struct {
	Findings []Finding
}

func (r *Result) ErrorCount() int {
	count := 0
	for _, f := range r.Findings {
		if f.Level == LevelError {
			count++
		}
	}
	return count
}

func (r *Result) WarnCount() int {
	count := 0
	for _, f := range r.Findings {
		if f.Level == LevelWarn {
			count++
		}
	}
	return count
}

// CountByCategory returns the number of findings in the given category.
func (r *Result) CountByCategory(category string) int {
	count := 0
	for _, f := range r.Findings {
		if f.Category == category {
			count++
		}
	}
	return count
}

func taxonomyFindings(relPath, kind string, values []string, line int) []Finding {
	var findings []Finding
	for _, value := range values {
		norm, usable := content.NormalizeTaxonomyValue(value)
		if usable && norm == value {
			continue
		}
		finding := Finding{
			File:     relPath,
			Line:     line,
			Level:    LevelWarn,
			Category: CategoryMissingMetadata,
		}
		switch {
		case len(norm) > content.MaxTaxonomyValueLen:
			finding.Level = LevelError
			finding.Message = fmt.Sprintf("%s %q cannot be published: normalized value is %d bytes (limit %d); shorten it", kind, value, len(norm), content.MaxTaxonomyValueLen)
		case !usable:
			finding.Level = LevelError
			finding.Message = fmt.Sprintf("%s %q cannot be published: normalization removes every character; use letters, digits, or hyphens", kind, value)
		default:
			finding.Message = fmt.Sprintf("malformed %s %q (normalized to %q)", kind, value, norm)
		}
		findings = append(findings, finding)
	}
	return findings
}

// Validate checks content files for frontmatter errors, invalid dates, malformed tags/categories,
// missing metadata (title/description), invalid render/slug combinations, path collisions,
// broken internal links, orphaned pages, and optional asset health.
func Validate(cfg config.Config) (*Result, error) {
	return ValidateWithManifest(cfg, "")
}

// ValidateWithManifest performs the same content checks as Validate, using
// the supplied build manifest for internal-link and orphan checks when one is
// provided. Other validation continues to use the current source tree.
func ValidateWithManifest(cfg config.Config, manifestPath string) (*Result, error) {
	allFileMap, err := content.GatherMetadata(cfg.ContentDir)
	if err != nil {
		return nil, fmt.Errorf("failed to gather metadata: %w", err)
	}
	fileMap := content.PublishedFiles(allFileMap)

	var manifestPages map[string]sitedata.ManifestPage
	if manifestPath != "" {
		manifest, err := sitedata.ReadManifest(manifestPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read site manifest: %w", err)
		}
		if len(manifest.Pages) != len(fileMap) {
			return nil, fmt.Errorf("site manifest has %d pages; current content has %d", len(manifest.Pages), len(fileMap))
		}
		manifestPages = make(map[string]sitedata.ManifestPage, len(manifest.Pages))
		for _, page := range manifest.Pages {
			if _, exists := fileMap[page.SourcePath]; !exists {
				return nil, fmt.Errorf("site manifest contains unknown source page %q", page.SourcePath)
			}
			if _, duplicate := manifestPages[page.SourcePath]; duplicate {
				return nil, fmt.Errorf("site manifest contains duplicate source page %q", page.SourcePath)
			}
			manifestPages[page.SourcePath] = page
		}
		for sourcePath := range fileMap {
			if _, exists := manifestPages[sourcePath]; !exists {
				return nil, fmt.Errorf("site manifest does not contain source page %q", sourcePath)
			}
		}
	}

	var findings []Finding

	// A missing siteurl is only a planning concern locally, but it becomes a
	// ship-blocking omission when discovery files are published: without it the
	// generated sitemap.xml carries root-relative <loc> entries, which the
	// sitemaps.org protocol requires to be absolute. Surface it as a site-wide
	// warning so `check` flags what the quickstart deploy section warns about
	// (#535).
	if strings.TrimSpace(cfg.SiteURL) == "" && strings.TrimSpace(cfg.LegacySiteURL) == "" {
		findings = append(findings, Finding{
			Level:    LevelWarn,
			Category: CategoryMissingMetadata,
			Message:  "siteurl is unset: sitemap.xml <loc> entries will be root-relative, which the sitemaps.org protocol requires to be absolute — set siteurl before deploying publicly",
		})
	}

	// Sort file keys for deterministic evaluation order
	keys := make([]string, 0, len(fileMap))
	for k := range fileMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	mdEngine := markdown.NewEngine(nil)

	// Output-tree links (extension-less and .html) are validated against where
	// a build will actually write, so compute that once up front (#506). Stub
	// targets extend the set: the build publishes a generated "Missing Page"
	// for every missing .md/wiki destination, so a link landing there is a
	// warning at most (#647). Skipped when a manifest drives the link checks —
	// its recorded resolution already encodes the build-time outcome.
	var expectedOutputs, stubOnlyOutputs map[string]bool
	if manifestPages == nil {
		stubTargets := collectStubTargets(fileMap, allFileMap)
		expectedOutputs, stubOnlyOutputs = buildExpectedOutputs(fileMap, cfg.GraphExplorer, stubTargets)
	}

	for _, relPath := range keys {
		meta := fileMap[relPath]

		// 1. Frontmatter syntax check
		var rawMatter map[string]interface{}
		_, fmErr := frontmatter.Parse(bytes.NewReader(meta.Content), &rawMatter)
		if fmErr != nil {
			findings = append(findings, Finding{
				File:     relPath,
				Line:     1,
				Level:    LevelError,
				Category: CategoryMissingMetadata,
				Message:  fmt.Sprintf("invalid frontmatter: %v", fmErr),
			})
		}

		if rawMatter != nil {
			normalizedMatter := make(map[string]interface{})
			for k, v := range rawMatter {
				normalizedMatter[strings.ToLower(k)] = v
			}
			yamlBytes, yErr := yaml.Marshal(normalizedMatter)
			if yErr == nil {
				var matter struct {
					Render      *bool              `yaml:"render"`
					Publish     *bool              `yaml:"publish"`
					Date        string             `yaml:"date"`
					Slug        string             `yaml:"slug"`
					Tags        content.StringList `yaml:"tags"`
					Categories  content.StringList `yaml:"categories"`
					Category    content.StringList `yaml:"category"`
					Title       string             `yaml:"title"`
					Description string             `yaml:"description"`
				}
				_ = yaml.Unmarshal(yamlBytes, &matter)

				// Date validation
				if matter.Date != "" {
					if _, err := time.Parse(time.DateOnly, matter.Date); err != nil {
						line := findFieldLine(meta.Content, "date")
						findings = append(findings, Finding{
							File:     relPath,
							Line:     line,
							Level:    LevelError,
							Category: CategoryMissingMetadata,
							Message:  fmt.Sprintf("invalid date format %q: must be YYYY-MM-DD", matter.Date),
						})
					}
				}

				// Validate with the same normalization and byte limit as
				// GatherMetadata. A term that cannot produce an archive must
				// not pass check with only a warning.
				findings = append(findings, taxonomyFindings(relPath, "tag", []string(matter.Tags), findFieldLine(meta.Content, "tags"))...)
				findings = append(findings, taxonomyFindings(relPath, "category", []string(matter.Categories), findFieldLine(meta.Content, "categories"))...)
				findings = append(findings, taxonomyFindings(relPath, "category", []string(matter.Category), findFieldLine(meta.Content, "category"))...)

				// Render & Slug combination check
				if matter.Render != nil && !*matter.Render && matter.Slug != "" {
					line := findFieldLine(meta.Content, "slug")
					findings = append(findings, Finding{
						File:     relPath,
						Line:     line,
						Level:    LevelError,
						Category: CategoryMissingMetadata,
						Message:  fmt.Sprintf("invalid render/slug combination: slug %q specified when render is false", matter.Slug),
					})
				}

				// Slug validity check
				if matter.Slug != "" {
					slug := matter.Slug
					if !filepath.IsLocal(slug) || strings.Contains(slug, ".") || strings.Contains(slug, string(filepath.Separator)) || strings.Contains(slug, "/") {
						line := findFieldLine(meta.Content, "slug")
						findings = append(findings, Finding{
							File:     relPath,
							Line:     line,
							Level:    LevelError,
							Category: CategoryMissingMetadata,
							Message:  fmt.Sprintf("invalid slug %q: slug must be a simple local name without slashes or dots", slug),
						})
					}
				}
			}
		}

		// 1b. Missing metadata warnings (title, description) — only for rendered pages
		shouldRender := true
		if meta.Render != nil && !*meta.Render {
			shouldRender = false
		}
		if shouldRender {
			if strings.TrimSpace(meta.Title) == "" {
				findings = append(findings, Finding{
					File:     relPath,
					Level:    LevelWarn,
					Category: CategoryMissingMetadata,
					Message:  "missing title: <meta property=\"og:title\"> will fallback to filename",
				})
			}
			if strings.TrimSpace(meta.Description) == "" {
				findings = append(findings, Finding{
					File:     relPath,
					Level:    LevelWarn,
					Category: CategoryMissingMetadata,
					Message:  "missing description: page will use default_description if configured",
				})
			}
			// A filename-derived slug becomes a URL directory verbatim, so an
			// unsafe name ships broken URLs and malformed sitemap entries that
			// nothing else flags before deploy (#509).
			if suggestion := slugSuggestion(relPath); suggestion != "" {
				findings = append(findings, Finding{
					File:     relPath,
					Level:    LevelWarn,
					Category: CategoryMissingMetadata,
					Message:  fmt.Sprintf("slug contains characters unsafe for URLs (spaces, uppercase) — consider renaming to %s", suggestion),
				})
			}
		}

		// 2. Internal Markdown links validation
		if manifestPages != nil {
			page := manifestPages[relPath]
			for _, link := range page.Links {
				if link.Resolved {
					continue
				}
				finding := Finding{
					File:     relPath,
					Line:     link.Line,
					Level:    LevelError,
					Category: CategoryBrokenLink,
					Message:  fmt.Sprintf("broken internal link %q -> %q", link.Destination, link.Target),
				}
				switch {
				case !filepath.IsLocal(filepath.FromSlash(link.Target)):
					// The link's target climbs out of the content root; it can
					// only ship verbatim and 404 (#648).
					finding.Message = fmt.Sprintf("internal link %q escapes the content root (-> %q)", link.Destination, link.Target)
				case strings.HasSuffix(strings.ToLower(link.Target), ".md"):
					// A local .md target the build could not resolve becomes a
					// generated "Missing Page" stub — a warning at most (#647).
					finding.Level = LevelWarn
					finding.Message = fmt.Sprintf("internal link %q -> %q resolves to a generated \"Missing Page\" stub", link.Destination, link.Target)
				}
				findings = append(findings, finding)
			}
		} else if len(meta.Rest) > 0 {
			doc := mdEngine.Parser().Parse(text.NewReader(meta.Rest))
			_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
				if !entering {
					return ast.WalkContinue, nil
				}

				link, ok := n.(*ast.Link)
				if !ok {
					return ast.WalkContinue, nil
				}

				dest := string(link.Destination)
				if wikiTarget, _, isWikiLink := transform.ParseWikiLinkDestination(dest); isWikiLink {
					if _, _, resolved := transform.ResolveWikiTarget(relPath, wikiTarget, allFileMap); resolved {
						return ast.WalkContinue, nil
					}
					targetRelPath := transform.UnresolvedWikiTargetPath(relPath, wikiTarget)
					lineNo := findLinkLine(meta.Content, meta.Rest, n, dest)
					if f, ok := stubLinkFinding(relPath, lineNo, wikiTarget, targetRelPath, stubOnlyOutputs); ok {
						findings = append(findings, f)
					}
					return ast.WalkContinue, nil
				}
				u, err := url.Parse(dest)
				if err != nil || u.IsAbs() || strings.HasPrefix(dest, "//") || u.Path == "" {
					return ast.WalkContinue, nil
				}

				ext := strings.ToLower(path.Ext(u.Path))
				isSourceRef := ext == ".md"
				// Extension-less and .html links address the generated output
				// tree rather than the source tree; they pass through a build
				// verbatim, so the only way to catch a typo is to resolve them
				// against where the build will actually write.
				isOutputRef := ext == "" || ext == ".html"
				if !isSourceRef && !isOutputRef {
					return ast.WalkContinue, nil
				}

				var targetRelPath string
				if isSourceRef {
					targetRelPath = sourceTreeTarget(relPath, u.Path)

					// A target that climbs out of the content root ships
					// verbatim into the artifact and 404s (#648) — report it
					// instead of skipping it. Locality is judged on the
					// decoded path, which covers %-encoded ".." segments too.
					if !filepath.IsLocal(filepath.FromSlash(targetRelPath)) {
						findings = append(findings, Finding{
							File:     relPath,
							Line:     findLinkLine(meta.Content, meta.Rest, n, dest),
							Level:    LevelError,
							Category: CategoryBrokenLink,
							Message:  fmt.Sprintf("internal link %q escapes the content root (-> %q)", dest, targetRelPath),
						})
						return ast.WalkContinue, nil
					}

					if targetMeta, exists := allFileMap[targetRelPath]; exists {
						if !content.IsPublished(targetMeta) {
							return ast.WalkContinue, nil
						}
						return ast.WalkContinue, nil
					}

					// The target does not exist. When the link spells it the
					// way LinkTransformer records missing files (".md" suffix),
					// the build publishes a generated stub there — a warning
					// at most (#647). If some other writer already owns the
					// output path the link lands on real content, not a stub.
					if strings.HasSuffix(u.Path, ".md") {
						lineNo := findLinkLine(meta.Content, meta.Rest, n, dest)
						if f, ok := stubLinkFinding(relPath, lineNo, dest, targetRelPath, stubOnlyOutputs); ok {
							findings = append(findings, f)
						}
						return ast.WalkContinue, nil
					}
				} else {
					candidate := normalizeOutputCandidate(outputTreeTarget(relPath, u.Path, meta))
					if expectedOutputFor(expectedOutputs, candidate) {
						return ast.WalkContinue, nil
					}
					targetRelPath = candidate
				}

				lineNo := findLinkLine(meta.Content, meta.Rest, n, dest)
				findings = append(findings, Finding{
					File:     relPath,
					Line:     lineNo,
					Level:    LevelError,
					Category: CategoryBrokenLink,
					Message:  fmt.Sprintf("broken internal link %q -> %q", dest, targetRelPath),
				})

				return ast.WalkContinue, nil
			})
		}
	}

	// 3. Output path collisions (duplicate/conflicting metadata)
	owners := make(map[string]string)
	for _, relPath := range keys {
		meta := fileMap[relPath]
		if meta.Render != nil && !*meta.Render {
			continue
		}
		slug := meta.Slug
		if slug != "" && (!filepath.IsLocal(slug) || strings.Contains(slug, ".") || strings.Contains(slug, string(filepath.Separator)) || strings.Contains(slug, "/")) {
			slug = ""
		}
		relOut := transform.GetOutputURL(relPath, slug, true)
		if prev, exists := owners[relOut]; exists {
			findings = append(findings, Finding{
				File:     relPath,
				Line:     0,
				Level:    LevelError,
				Category: CategoryBrokenLink,
				Message:  fmt.Sprintf("output path collision: %q and %q both map to %q", prev, relPath, relOut),
			})
		} else {
			owners[relOut] = relPath
		}
	}

	// 4. Orphan detection — zero-inbound rendered pages, exempting index (Explorer Orphan Rule)
	var orphanFindings []Finding
	if manifestPages != nil {
		orphanFindings = detectManifestOrphans(fileMap, manifestPages)
	} else {
		orphanFindings = detectOrphans(fileMap, allFileMap)
	}
	findings = append(findings, orphanFindings...)

	// 5. Asset health diagnostics (optional)
	if cfg.CheckAssetHealth {
		assetFindings, aErr := validateAssets(cfg, fileMap)
		if aErr != nil {
			return nil, fmt.Errorf("asset health check failed: %w", aErr)
		}
		findings = append(findings, assetFindings...)
	}

	// Sort findings deterministically by File, Line, Level, Message
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].File != findings[j].File {
			return findings[i].File < findings[j].File
		}
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		if findings[i].Level != findings[j].Level {
			return findings[i].Level < findings[j].Level
		}
		return findings[i].Message < findings[j].Message
	})

	return &Result{Findings: findings}, nil
}

func detectManifestOrphans(fileMap map[string]*content.FileMeta, pages map[string]sitedata.ManifestPage) []Finding {
	// The manifest's inbound counts only cover links the link transformer
	// graphs (.md and wiki); output-style links are recorded per page with an
	// empty GraphTarget, so count their resolved targets here too (#652).
	owners := outputOwners(fileMap)
	outputInbound := make(map[string]int)
	for _, page := range pages {
		for _, link := range page.Links {
			if !link.Resolved || link.GraphTarget != "" {
				continue
			}
			if id, ok := owners[link.Target]; ok {
				outputInbound[id]++
				continue
			}
			if strings.HasSuffix(link.Target, ".html") {
				if id, ok := owners[strings.TrimSuffix(link.Target, ".html")+"/index.html"]; ok {
					outputInbound[id]++
				}
			}
		}
	}

	var findings []Finding
	for relPath, meta := range fileMap {
		if meta == nil || (meta.Render != nil && !*meta.Render) {
			continue
		}
		identity := strings.TrimSuffix(relPath, ".md")
		page := pages[relPath]
		if page.InboundLinkCount+outputInbound[identity] == 0 && identity != "index" {
			findings = append(findings, Finding{
				File:     relPath,
				Level:    LevelWarn,
				Category: CategoryOrphan,
				Message:  "orphaned page: no inbound links",
			})
		}
	}
	return findings
}

// detectOrphans reuses the markdown link graph to find rendered pages with zero inbound links.
// It mirrors the logic in generator.ComputeContentHealth but without requiring a full build.
// The rendered homepage (id "index") is exempt per content/docs/publishing.md Explorer Orphan Rule.
func detectOrphans(fileMap, allFileMap map[string]*content.FileMeta) []Finding {
	// Build inbound count map for rendered pages.
	ids := make(map[string]string) // id -> relPath
	inbound := make(map[string]int)
	for relPath, meta := range fileMap {
		if meta.Render != nil && !*meta.Render {
			continue
		}
		id := strings.TrimSuffix(relPath, ".md")
		ids[id] = relPath
		inbound[id] = 0
	}
	if len(ids) == 0 {
		return nil
	}

	mdEngine := markdown.NewEngine(nil)
	// Output-style internal links (extension-less and .html) land on a page's
	// output path, so they count as inbound references too (#652).
	owners := outputOwners(fileMap)
	for relPath, meta := range fileMap {
		if len(meta.Rest) == 0 {
			continue
		}
		doc := mdEngine.Parser().Parse(text.NewReader(meta.Rest))
		_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}
			link, ok := n.(*ast.Link)
			if !ok {
				return ast.WalkContinue, nil
			}
			dest := string(link.Destination)
			if wikiTarget, _, isWikiLink := transform.ParseWikiLinkDestination(dest); isWikiLink {
				targetRelPath, targetMeta, resolved := transform.ResolveWikiTarget(relPath, wikiTarget, allFileMap)
				if !resolved {
					return ast.WalkContinue, nil
				}
				if !content.IsPublished(targetMeta) {
					return ast.WalkContinue, nil
				}
				targetID := strings.TrimSuffix(targetRelPath, ".md")
				if targetMeta != nil && targetMeta.Render != nil && !*targetMeta.Render {
					targetID = targetRelPath
				}
				if _, ok := inbound[targetID]; ok {
					inbound[targetID]++
				}
				return ast.WalkContinue, nil
			}
			u, err := url.Parse(dest)
			if err != nil || u.IsAbs() || strings.HasPrefix(dest, "//") || u.Path == "" {
				return ast.WalkContinue, nil
			}
			if !strings.HasSuffix(u.Path, ".md") {
				ext := strings.ToLower(path.Ext(u.Path))
				if ext != "" && ext != ".html" {
					return ast.WalkContinue, nil
				}
				candidate := normalizeOutputCandidate(outputTreeTarget(relPath, u.Path, meta))
				if id, ok := owners[candidate]; ok {
					if _, tracked := inbound[id]; tracked {
						inbound[id]++
					}
					return ast.WalkContinue, nil
				}
				if strings.HasSuffix(candidate, ".html") {
					if id, ok := owners[strings.TrimSuffix(candidate, ".html")+"/index.html"]; ok {
						if _, tracked := inbound[id]; tracked {
							inbound[id]++
						}
					}
				}
				return ast.WalkContinue, nil
			}
			targetRelPath := sourceTreeTarget(relPath, u.Path)
			if !filepath.IsLocal(filepath.FromSlash(targetRelPath)) {
				return ast.WalkContinue, nil
			}
			targetMeta, exists := allFileMap[targetRelPath]
			if !exists {
				return ast.WalkContinue, nil
			}
			if !content.IsPublished(targetMeta) {
				return ast.WalkContinue, nil
			}
			targetID := strings.TrimSuffix(targetRelPath, ".md")
			if targetMeta.Render != nil && !*targetMeta.Render {
				targetID = targetRelPath
			}
			if _, ok := inbound[targetID]; ok {
				inbound[targetID]++
			}
			return ast.WalkContinue, nil
		})
	}

	var findings []Finding
	for id, count := range inbound {
		if count == 0 && id != "index" {
			relPath := ids[id]
			findings = append(findings, Finding{
				File:     relPath,
				Level:    LevelWarn,
				Category: CategoryOrphan,
				Message:  "orphaned page: no inbound links",
			})
		}
	}
	return findings
}

var rasterExts = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".gif":  true,
	".webp": true,
	".bmp":  true,
	".tiff": true,
	".tif":  true,
	".ico":  true,
	".avif": true,
}

var suspiciousImageExts = map[string]bool{
	".psd":  true,
	".ai":   true,
	".eps":  true,
	".tiff": true,
	".tif":  true,
	".raw":  true,
	".cr2":  true,
	".nef":  true,
	".heic": true,
	".heif": true,
	".xcf":  true,
	".indd": true,
	".bmp":  true,
	".jp2":  true,
	".j2k":  true,
	".jpx":  true,
	".pnm":  true,
	".pbm":  true,
	".pgm":  true,
	".ppm":  true,
}

func validateAssets(cfg config.Config, fileMap map[string]*content.FileMeta) ([]Finding, error) {
	var findings []Finding
	if cfg.AssetDir == "" {
		return nil, nil
	}

	ignoreRules := asset.LoadIgnoreRules(cfg.ProjectRoot)

	maxSize := cfg.MaxAssetSizeBytes
	if maxSize <= 0 {
		maxSize = 5 * 1024 * 1024
	}

	validAssets := make(map[string]bool)
	assetCaseMap := make(map[string]string)

	assetDirStat, err := os.Stat(cfg.AssetDir)
	assetDirExists := err == nil && assetDirStat.IsDir()

	if assetDirExists {
		err := filepath.WalkDir(cfg.AssetDir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			relPath, err := filepath.Rel(cfg.AssetDir, path)
			if err != nil {
				return err
			}
			relSlash := filepath.ToSlash(relPath)

			if relSlash == "." {
				return nil
			}

			if d.Type()&os.ModeSymlink != 0 {
				findings = append(findings, Finding{
					File:     relSlash,
					Line:     0,
					Level:    LevelWarn,
					Category: CategoryAssetHealth,
					Message:  fmt.Sprintf("symlink in asset directory skipped: %s", relSlash),
				})
				return nil
			}

			// Boundary breakout check for asset paths
			if !filepath.IsLocal(relPath) || strings.HasPrefix(relSlash, "..") || !pathutil.IsSafePath(cfg.AssetDir, path) {
				findings = append(findings, Finding{
					File:     relSlash,
					Line:     0,
					Level:    LevelWarn,
					Category: CategoryAssetHealth,
					Message:  fmt.Sprintf("asset path %q escapes configured asset root %q", relSlash, cfg.AssetDir),
				})
				return nil
			}

			if asset.IsIgnoredAsset(path, d.IsDir(), relSlash, cfg.ProjectRoot, ignoreRules) {
				return nil
			}

			if d.IsDir() {
				return nil
			}

			// Valid, non-ignored asset file
			validAssets[relSlash] = true

			// Case-collision check
			lowerRel := strings.ToLower(relSlash)
			if prev, exists := assetCaseMap[lowerRel]; exists && prev != relSlash {
				findings = append(findings, Finding{
					File:     relSlash,
					Line:     0,
					Level:    LevelWarn,
					Category: CategoryAssetHealth,
					Message:  fmt.Sprintf("asset case-collision / duplicate destination risk: %q and %q map to the same destination %q", prev, relSlash, lowerRel),
				})
			} else {
				assetCaseMap[lowerRel] = relSlash
			}

			// Unsupported or suspicious extension check
			ext := strings.ToLower(filepath.Ext(relSlash))
			if suspiciousImageExts[ext] {
				findings = append(findings, Finding{
					File:     relSlash,
					Line:     0,
					Level:    LevelWarn,
					Category: CategoryAssetHealth,
					Message:  fmt.Sprintf("unsupported or suspicious image extension %q: prefer web-optimized formats (.png, .jpg, .webp, .svg, .avif)", ext),
				})
			}

			// Large raster asset check
			info, infoErr := d.Info()
			if infoErr == nil && rasterExts[ext] {
				if info.Size() > maxSize {
					findings = append(findings, Finding{
						File:     relSlash,
						Line:     0,
						Level:    LevelWarn,
						Category: CategoryAssetHealth,
						Message:  fmt.Sprintf("unusually large raster asset (%s > %s threshold)", formatBytes(info.Size()), formatBytes(maxSize)),
					})
				}
			}

			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to scan assets: %w", err)
		}
	}

	// Scan content files for missing referenced assets
	keys := make([]string, 0, len(fileMap))
	for k := range fileMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	mdEngine := markdown.NewEngine(nil)

	for _, relPath := range keys {
		meta := fileMap[relPath]
		if len(meta.Rest) == 0 {
			continue
		}

		doc := mdEngine.Parser().Parse(text.NewReader(meta.Rest))
		_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}

			var dest string
			isAssetRef := false

			switch node := n.(type) {
			case *ast.Image:
				dest = string(node.Destination)
				isAssetRef = true
			case *ast.Link:
				dest = string(node.Destination)
				ext := strings.ToLower(filepath.Ext(dest))
				if strings.HasPrefix(dest, "/assets/") || strings.HasPrefix(dest, "assets/") || rasterExts[ext] || suspiciousImageExts[ext] || ext == ".svg" {
					isAssetRef = true
				}
			}

			if !isAssetRef || dest == "" {
				return ast.WalkContinue, nil
			}

			u, err := url.Parse(dest)
			if err != nil || u.IsAbs() || strings.HasPrefix(dest, "//") {
				return ast.WalkContinue, nil
			}

			refPath := u.Path
			if refPath == "" {
				return ast.WalkContinue, nil
			}

			var assetRel string
			if strings.HasPrefix(refPath, "/assets/") {
				assetRel = strings.TrimPrefix(refPath, "/assets/")
			} else if strings.HasPrefix(refPath, "assets/") {
				assetRel = strings.TrimPrefix(refPath, "assets/")
			} else if strings.HasPrefix(refPath, "/") {
				assetRel = strings.TrimPrefix(refPath, "/")
			} else {
				dir := filepath.Dir(relPath)
				if dir == "." {
					assetRel = refPath
				} else {
					assetRel = dir + "/" + refPath
				}
				if strings.HasPrefix(assetRel, "assets/") {
					assetRel = strings.TrimPrefix(assetRel, "assets/")
				}
			}

			assetRel = filepath.ToSlash(filepath.Clean(assetRel))

			// Check for asset path escaping root
			if !filepath.IsLocal(filepath.FromSlash(assetRel)) || strings.HasPrefix(assetRel, "..") || strings.Contains(dest, "%2E%2E") {
				lineNo := findLinkLine(meta.Content, meta.Rest, n, dest)
				findings = append(findings, Finding{
					File:     relPath,
					Line:     lineNo,
					Level:    LevelWarn,
					Category: CategoryAssetHealth,
					Message:  fmt.Sprintf("referenced asset path %q escapes asset root", dest),
				})
				return ast.WalkContinue, nil
			}

			// Check existence in AssetDir
			if !validAssets[assetRel] {
				lineNo := findLinkLine(meta.Content, meta.Rest, n, dest)
				if actual, caseMismatch := assetCaseMap[strings.ToLower(assetRel)]; caseMismatch {
					findings = append(findings, Finding{
						File:     relPath,
						Line:     lineNo,
						Level:    LevelWarn,
						Category: CategoryAssetHealth,
						Message:  fmt.Sprintf("referenced asset %q has case mismatch with existing asset %q", dest, actual),
					})
				} else {
					findings = append(findings, Finding{
						File:     relPath,
						Line:     lineNo,
						Level:    LevelWarn,
						Category: CategoryAssetHealth,
						Message:  fmt.Sprintf("missing referenced asset %q", dest),
					})
				}
			}

			return ast.WalkContinue, nil
		})
	}

	// Scan installed templates for local /assets/ references that no build
	// would satisfy (#515): a layout referencing a missing image passed plain
	// check silently and only surfaced at publish-check, after the artifact.
	findings = append(findings, validateTemplateAssets(cfg, validAssets)...)

	return findings, nil
}

var templateAssetRef = regexp.MustCompile(`(?i)(?:src|href)\s*=\s*["'](/assets/[^"'#?]*)["']`)

func validateTemplateAssets(cfg config.Config, validAssets map[string]bool) []Finding {
	var findings []Finding
	if cfg.Template == "" {
		return nil
	}
	templateDir := filepath.Dir(cfg.Template)
	if info, err := os.Stat(templateDir); err != nil || !info.IsDir() {
		return nil
	}

	// The released binary fills absent paths in public/assets from its
	// embedded fallback bundle, so those references resolve even without a
	// matching file in the project's assets directory.
	embedded := make(map[string]bool)
	files, err := runtimeassets.DefaultAssetFiles()
	if err == nil {
		for name := range files {
			if !cfg.GraphExplorer && strings.HasPrefix(name, "graph/") {
				continue
			}
			embedded[name] = true
		}
	}

	err = filepath.WalkDir(templateDir, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(p), ".html") {
			return nil
		}
		data, readErr := os.ReadFile(p)
		if readErr != nil {
			return readErr
		}
		relTemplate, err := filepath.Rel(templateDir, p)
		if err != nil {
			return err
		}
		relTemplate = filepath.ToSlash(relTemplate)

		for _, match := range templateAssetRef.FindAllStringSubmatch(string(data), -1) {
			ref := match[1]
			assetRel := filepath.ToSlash(filepath.Clean(strings.TrimPrefix(ref, "/assets/")))
			if validAssets[assetRel] || embedded[assetRel] {
				continue
			}
			findings = append(findings, Finding{
				File:     relTemplate,
				Level:    LevelWarn,
				Category: CategoryAssetHealth,
				Message:  fmt.Sprintf("template references missing local asset %q", ref),
			})
		}
		return nil
	})
	if err != nil {
		return append(findings, Finding{
			File:     filepath.ToSlash(templateDir),
			Level:    LevelWarn,
			Category: CategoryAssetHealth,
			Message:  fmt.Sprintf("failed to scan templates for asset references: %v", err),
		})
	}
	sort.Slice(findings, func(i, j int) bool { return findings[i].File < findings[j].File })
	return findings
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func findFieldLine(content []byte, fieldName string) int {
	lines := strings.Split(string(content), "\n")
	prefix := strings.ToLower(fieldName) + ":"
	for i, line := range lines {
		trimmed := strings.ToLower(strings.TrimSpace(line))
		if strings.HasPrefix(trimmed, prefix) {
			return i + 1
		}
	}
	return 1
}

func findLinkLine(fullContent []byte, restBytes []byte, node ast.Node, dest string) int {
	restOffset := len(fullContent) - len(restBytes)
	curr := node.Parent()
	startOffset := -1
	for curr != nil {
		if curr.Type() == ast.TypeBlock {
			if lines := curr.Lines(); lines != nil && lines.Len() > 0 {
				startOffset = lines.At(0).Start
				break
			}
		}
		curr = curr.Parent()
	}

	if startOffset >= 0 {
		searchFrom := restOffset + startOffset
		if searchFrom < len(fullContent) {
			if target, _, ok := transform.ParseWikiLinkDestination(dest); ok {
				if idx := bytes.Index(fullContent[searchFrom:], []byte("[["+target)); idx >= 0 {
					return lineFromOffset(fullContent, searchFrom+idx)
				}
			}
			if idx := bytes.Index(fullContent[searchFrom:], []byte(dest)); idx >= 0 {
				return lineFromOffset(fullContent, searchFrom+idx)
			}
		}
	}

	if idx := bytes.Index(fullContent, []byte(dest)); idx >= 0 {
		return lineFromOffset(fullContent, idx)
	}

	return 1
}

func lineFromOffset(content []byte, offset int) int {
	if offset <= 0 || offset > len(content) {
		return 1
	}
	line := 1
	for i := 0; i < offset && i < len(content); i++ {
		if content[i] == '\n' {
			line++
		}
	}
	return line
}

// buildExpectedOutputs returns the set of output-relative paths a build is
// expected to emit: every rendered and unrendered page, taxonomy listings for
// terms actually present, the graph explorer when enabled, the unresolved-notes
// index, generated "Missing Page" stubs for the supplied missing targets, and
// the homepage. It mirrors transform.GetOutputURL so a link that will resolve
// after build is never flagged as broken.
//
// The second return value is the subset of outputs that exist only because a
// stub will be written — the paths where a link lands on generated placeholder
// content rather than a real page.
func buildExpectedOutputs(fileMap map[string]*content.FileMeta, graphExplorer bool, stubTargets map[string]bool) (map[string]bool, map[string]bool) {
	outputs := make(map[string]bool)
	tags := make(map[string]bool)
	categories := make(map[string]bool)

	outputs["index.html"] = true
	// The build writes the unresolved-notes index even when it is empty
	// (generator.writeUnresolvedNotesIndex).
	outputs["unresolved-notes/index.html"] = true

	addTerm := func(kind string, term string) {
		if term == "" {
			return
		}
		outputs[path.Join(kind, term, "index.html")] = true
	}

	for relPath, meta := range fileMap {
		out := pageOutputPath(relPath, meta)
		outputs[out] = true

		render := meta.Render == nil || *meta.Render
		if !render {
			continue
		}
		for _, tag := range meta.Tags {
			if tag != "" {
				tags[tag] = true
			}
		}
		for _, cat := range meta.Categories {
			if cat != "" {
				categories[cat] = true
			}
		}
	}

	if len(tags) > 0 {
		outputs["tags/index.html"] = true
		for term := range tags {
			addTerm("tags", term)
		}
	}
	if len(categories) > 0 {
		outputs["categories/index.html"] = true
		for term := range categories {
			addTerm("categories", term)
		}
	}
	if graphExplorer {
		outputs["graph/index.html"] = true
	}

	// Stubs are generated last and lose to any writer that already claimed
	// their path, so only a previously-unclaimed output is a stub write.
	stubOnly := make(map[string]bool, len(stubTargets))
	for target := range stubTargets {
		out := filepath.ToSlash(filepath.Clean(transform.GetOutputURL(target, "", true)))
		if !outputs[out] {
			stubOnly[out] = true
			outputs[out] = true
		}
	}
	return outputs, stubOnly
}

// sourceTreeTarget resolves a .md link target to its content-tree path the way
// LinkTransformer does: root-relative paths anchor at the content root, others
// join onto the linking page's source directory.
func sourceTreeTarget(relPath, urlPath string) string {
	if strings.HasPrefix(urlPath, "/") {
		return filepath.ToSlash(filepath.Clean(strings.TrimPrefix(urlPath, "/")))
	}
	if dir := filepath.Dir(relPath); dir != "." {
		return filepath.ToSlash(filepath.Clean(dir + "/" + urlPath))
	}
	return filepath.ToSlash(filepath.Clean(urlPath))
}

// outputTreeTarget resolves an output-tree link the way a browser resolves it
// against the rendered page's URL: relative to the page's output directory,
// with ".." segments that climb above the output root clamped there (#648).
func outputTreeTarget(relPath, urlPath string, meta *content.FileMeta) string {
	raw := urlPath
	if strings.HasPrefix(raw, "/") {
		raw = strings.TrimPrefix(raw, "/")
	} else if dir := pageOutputDir(relPath, meta); dir != "." {
		raw = dir + "/" + raw
	}
	cleaned := path.Clean(raw)
	for strings.HasPrefix(cleaned, "../") {
		cleaned = strings.TrimPrefix(cleaned, "../")
	}
	if cleaned == ".." {
		cleaned = "."
	}
	return cleaned
}

// pageOutputPath returns the output-relative path a page is written to,
// honouring the same slug/render rules the renderer applies.
func pageOutputPath(relPath string, meta *content.FileMeta) string {
	render := meta.Render == nil || *meta.Render
	slug := meta.Slug
	if slug != "" && !transform.IsUsableSlug(slug) {
		slug = ""
	}
	return filepath.ToSlash(filepath.Clean(transform.GetOutputURL(relPath, slug, render)))
}

// pageOutputDir returns the URL directory a page is served from — the base for
// resolving relative output-tree links.
func pageOutputDir(relPath string, meta *content.FileMeta) string {
	return path.Dir(pageOutputPath(relPath, meta))
}

// outputOwners maps each page's output path to the page id that owns it, so
// output-style links can be credited as inbound references (#652).
func outputOwners(fileMap map[string]*content.FileMeta) map[string]string {
	owners := make(map[string]string, len(fileMap))
	for relPath, meta := range fileMap {
		if meta == nil {
			continue
		}
		id := strings.TrimSuffix(relPath, ".md")
		if meta.Render != nil && !*meta.Render {
			id = relPath
		}
		owners[pageOutputPath(relPath, meta)] = id
	}
	return owners
}

// stubLinkFinding reports a link whose missing target a build satisfies with a
// generated "Missing Page" stub — a warning matching publish-check semantics
// (#647). When the target's output path is already owned by real content (the
// stub's claim would be denied) the link resolves to that content and there is
// nothing to report.
func stubLinkFinding(relPath string, line int, dest, targetRelPath string, stubOnly map[string]bool) (Finding, bool) {
	stubOut := filepath.ToSlash(filepath.Clean(transform.GetOutputURL(targetRelPath, "", true)))
	if !stubOnly[stubOut] {
		return Finding{}, false
	}
	return Finding{
		File:     relPath,
		Line:     line,
		Level:    LevelWarn,
		Category: CategoryBrokenLink,
		Message:  fmt.Sprintf("internal link %q -> %q resolves to a generated \"Missing Page\" stub", dest, targetRelPath),
	}, true
}

// collectStubTargets returns the content-tree paths a build satisfies with
// generated "Missing Page" stubs: local .md link targets absent from the file
// map, and unresolved wiki targets. It mirrors LinkTransformer's missing-file
// collection, which is what feeds stub.GenerateStubs.
func collectStubTargets(published, fileMap map[string]*content.FileMeta) map[string]bool {
	mdEngine := markdown.NewEngine(nil)
	targets := make(map[string]bool)
	for relPath, meta := range published {
		if meta == nil || len(meta.Rest) == 0 {
			continue
		}
		doc := mdEngine.Parser().Parse(text.NewReader(meta.Rest))
		_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}
			link, ok := n.(*ast.Link)
			if !ok {
				return ast.WalkContinue, nil
			}
			dest := string(link.Destination)
			if wikiTarget, _, isWiki := transform.ParseWikiLinkDestination(dest); isWiki {
				if _, _, resolved := transform.ResolveWikiTarget(relPath, wikiTarget, fileMap); !resolved {
					targets[transform.UnresolvedWikiTargetPath(relPath, wikiTarget)] = true
				}
				return ast.WalkContinue, nil
			}
			u, err := url.Parse(dest)
			if err != nil || u.IsAbs() || strings.HasPrefix(dest, "//") || !strings.HasSuffix(u.Path, ".md") {
				return ast.WalkContinue, nil
			}
			targetRelPath := sourceTreeTarget(relPath, u.Path)
			if !filepath.IsLocal(filepath.FromSlash(targetRelPath)) {
				return ast.WalkContinue, nil
			}
			if _, exists := fileMap[targetRelPath]; !exists {
				targets[targetRelPath] = true
			}
			return ast.WalkContinue, nil
		})
	}
	return targets
}

// normalizeOutputCandidate canonicalizes an already-clean output-tree path to
// its index.html form, mirroring how the publisher resolves references inside
// public/.
func normalizeOutputCandidate(cleaned string) string {
	if cleaned == "" || cleaned == "." {
		return "index.html"
	}
	cleaned = strings.TrimSuffix(cleaned, "/")
	if !strings.HasSuffix(cleaned, ".html") {
		cleaned = cleaned + "/index.html"
	}
	return cleaned
}

// expectedOutputFor reports whether a normalized candidate (optionally joined
// onto fromDir for relative links) resolves to an expected output file.
func expectedOutputFor(expected map[string]bool, candidate string) bool {
	if expected[candidate] {
		return true
	}
	// The generator may emit content/foo.md as foo/index.html; when a link
	// says foo.html, also accept foo/index.html — same contract as publisher.
	if strings.HasSuffix(candidate, ".html") {
		dirForm := strings.TrimSuffix(candidate, ".html") + "/index.html"
		return expected[dirForm]
	}
	return false
}

// slugSuggestion returns a renamed filename suggestion when the filename-derived
// slug of relPath contains URL-unsafe characters, or "" when the name is safe.
func slugSuggestion(relPath string) string {
	base := strings.TrimSuffix(path.Base(filepath.ToSlash(relPath)), ".md")
	if base == "" || base == "index" {
		return ""
	}
	if isValidSlugName(base) {
		return ""
	}
	return normalizeSlugName(base) + ".md"
}

func isValidSlugName(name string) bool {
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '.':
		default:
			return false
		}
	}
	return name != ""
}

func normalizeSlugName(name string) string {
	var sb strings.Builder
	lastHyphen := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			sb.WriteRune(r)
			lastHyphen = false
		case r == '-' || r == '_' || r == '.':
			sb.WriteRune(r)
			lastHyphen = r == '-'
		default:
			// Collapse runs of unsafe characters (spaces and friends) into a
			// single hyphen so "My Post" normalizes rather than disappearing.
			if !lastHyphen {
				sb.WriteRune('-')
				lastHyphen = true
			}
		}
	}
	normalized := strings.Trim(sb.String(), "-")
	if normalized == "" {
		return "untitled"
	}
	return normalized
}
