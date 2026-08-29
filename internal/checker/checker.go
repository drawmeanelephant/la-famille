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
	"github.com/tbuddy/la-famille/internal/transform"
)

var validTagRegex = regexp.MustCompile(`^[a-z0-9-]+$`)

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

// Validate checks content files for frontmatter errors, invalid dates, malformed tags/categories,
// missing metadata (title/description), invalid render/slug combinations, path collisions,
// broken internal links, orphaned pages, and optional asset health.
func Validate(cfg config.Config) (*Result, error) {
	fileMap, err := content.GatherMetadata(cfg.ContentDir)
	if err != nil {
		return nil, fmt.Errorf("failed to gather metadata: %w", err)
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
	// a build will actually write, so compute that once up front (#506).
	expectedOutputs := buildExpectedOutputs(fileMap, cfg.GraphExplorer)

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

				// Tags validation
				for _, tag := range []string(matter.Tags) {
					if !validTagRegex.MatchString(tag) {
						lower := strings.ToLower(tag)
						var sb strings.Builder
						for _, r := range lower {
							if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
								sb.WriteRune(r)
							}
						}
						norm := sb.String()
						line := findFieldLine(meta.Content, "tags")
						findings = append(findings, Finding{
							File:     relPath,
							Line:     line,
							Level:    LevelWarn,
							Category: CategoryMissingMetadata,
							Message:  fmt.Sprintf("malformed tag %q (normalized to %q)", tag, norm),
						})
					}
				}

				// Categories validation — mirrors tags (^[a-z0-9-]+$)
				allCategories := append([]string(nil), []string(matter.Categories)...)
				allCategories = append(allCategories, []string(matter.Category)...)
				for _, cat := range allCategories {
					if !validTagRegex.MatchString(cat) {
						lower := strings.ToLower(cat)
						var sb strings.Builder
						for _, r := range lower {
							if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
								sb.WriteRune(r)
							}
						}
						norm := sb.String()
						line := findFieldLine(meta.Content, "categories")
						if findFieldLine(meta.Content, "category") != 1 || strings.Contains(strings.ToLower(string(meta.Content)), "category:") {
							// Prefer the line of the key that actually appears; fall back to categories
							if strings.Contains(strings.ToLower(string(meta.Content)), "\ncategory:") || strings.HasPrefix(strings.ToLower(strings.TrimSpace(string(meta.Content))), "category:") {
								line = findFieldLine(meta.Content, "category")
							}
						}
						findings = append(findings, Finding{
							File:     relPath,
							Line:     line,
							Level:    LevelWarn,
							Category: CategoryMissingMetadata,
							Message:  fmt.Sprintf("malformed category %q (normalized to %q)", cat, norm),
						})
					}
				}

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
		if len(meta.Rest) > 0 {
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
					if strings.HasPrefix(u.Path, "/") {
						targetRelPath = filepath.ToSlash(filepath.Clean(strings.TrimPrefix(u.Path, "/")))
					} else {
						dir := filepath.Dir(relPath)
						if dir == "." {
							targetRelPath = filepath.ToSlash(filepath.Clean(u.Path))
						} else {
							targetRelPath = filepath.ToSlash(filepath.Clean(dir + "/" + u.Path))
						}
					}

					if !filepath.IsLocal(filepath.FromSlash(targetRelPath)) || strings.Contains(dest, "%2E%2E") {
						return ast.WalkContinue, nil
					}

					if _, exists := fileMap[targetRelPath]; exists {
						return ast.WalkContinue, nil
					}
				} else {
					// Resolve against the output tree: join onto the page's
					// directory for relative links before cleaning, so ../
					// spellings are validated against where a build writes.
					raw := u.Path
					if strings.HasPrefix(raw, "/") {
						raw = strings.TrimPrefix(raw, "/")
					} else if dir := filepath.ToSlash(filepath.Dir(relPath)); dir != "." {
						raw = dir + "/" + raw
					}
					cleaned := path.Clean(raw)
					if cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.Contains(dest, "%2E%2E") {
						return ast.WalkContinue, nil
					}
					candidate := normalizeOutputCandidate(cleaned)
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
	orphanFindings := detectOrphans(fileMap)
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

// detectOrphans reuses the markdown link graph to find rendered pages with zero inbound links.
// It mirrors the logic in generator.ComputeContentHealth but without requiring a full build.
// The rendered homepage (id "index") is exempt per content/docs/publishing.md Explorer Orphan Rule.
func detectOrphans(fileMap map[string]*content.FileMeta) []Finding {
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
			u, err := url.Parse(dest)
			if err != nil || u.IsAbs() || strings.HasPrefix(dest, "//") || !strings.HasSuffix(u.Path, ".md") {
				return ast.WalkContinue, nil
			}
			var targetRelPath string
			if strings.HasPrefix(u.Path, "/") {
				targetRelPath = filepath.ToSlash(filepath.Clean(strings.TrimPrefix(u.Path, "/")))
			} else {
				dir := filepath.Dir(relPath)
				if dir == "." {
					targetRelPath = filepath.ToSlash(filepath.Clean(u.Path))
				} else {
					targetRelPath = filepath.ToSlash(filepath.Clean(dir + "/" + u.Path))
				}
			}
			if !filepath.IsLocal(filepath.FromSlash(targetRelPath)) || strings.Contains(dest, "%2E%2E") {
				return ast.WalkContinue, nil
			}
			targetMeta, exists := fileMap[targetRelPath]
			if !exists {
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
// terms actually present, the graph explorer when enabled, and the homepage.
// It mirrors transform.GetOutputURL so a link that will resolve after build is
// never flagged as broken.
func buildExpectedOutputs(fileMap map[string]*content.FileMeta, graphExplorer bool) map[string]bool {
	outputs := make(map[string]bool)
	tags := make(map[string]bool)
	categories := make(map[string]bool)

	outputs["index.html"] = true

	addTerm := func(kind string, term string) {
		if term == "" {
			return
		}
		outputs[path.Join(kind, term, "index.html")] = true
	}

	for relPath, meta := range fileMap {
		render := meta.Render == nil || *meta.Render
		slug := meta.Slug
		if slug != "" && !transform.IsUsableSlug(slug) {
			slug = ""
		}
		out := filepath.ToSlash(filepath.Clean(transform.GetOutputURL(relPath, slug, render)))
		outputs[out] = true

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
	return outputs
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
