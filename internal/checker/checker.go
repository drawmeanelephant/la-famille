package checker

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/adrg/frontmatter"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v2"

	"github.com/tbuddy/la-famille/internal/asset"
	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/markdown"
	"github.com/tbuddy/la-famille/internal/pathutil"
	"github.com/tbuddy/la-famille/internal/transform"
)

var validTagRegex = regexp.MustCompile(`^[a-z0-9-]+$`)

type Level string

const (
	LevelError Level = "ERROR"
	LevelWarn  Level = "WARN"
)

type Finding struct {
	File    string
	Line    int
	Level   Level
	Message string
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

// Validate checks content files for frontmatter errors, invalid dates, malformed tags,
// invalid render/slug combinations, path collisions, and broken internal links.
func Validate(cfg config.Config) (*Result, error) {
	fileMap, err := content.GatherMetadata(cfg.ContentDir)
	if err != nil {
		return nil, fmt.Errorf("failed to gather metadata: %w", err)
	}

	var findings []Finding

	// Sort file keys for deterministic evaluation order
	keys := make([]string, 0, len(fileMap))
	for k := range fileMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	mdEngine := markdown.NewEngine(nil)

	for _, relPath := range keys {
		meta := fileMap[relPath]

		// 1. Frontmatter syntax check
		var rawMatter map[string]interface{}
		_, fmErr := frontmatter.Parse(bytes.NewReader(meta.Content), &rawMatter)
		if fmErr != nil {
			findings = append(findings, Finding{
				File:    relPath,
				Line:    1,
				Level:   LevelError,
				Message: fmt.Sprintf("invalid frontmatter: %v", fmErr),
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
					Date   string   `yaml:"date"`
					Tags   []string `yaml:"tags"`
					Slug   string   `yaml:"slug"`
					Render *bool    `yaml:"render"`
				}
				_ = yaml.Unmarshal(yamlBytes, &matter)

				// Date validation
				if matter.Date != "" {
					if _, err := time.Parse(time.DateOnly, matter.Date); err != nil {
						line := findFieldLine(meta.Content, "date")
						findings = append(findings, Finding{
							File:    relPath,
							Line:    line,
							Level:   LevelError,
							Message: fmt.Sprintf("invalid date format %q: must be YYYY-MM-DD", matter.Date),
						})
					}
				}

				// Tags validation
				for _, tag := range matter.Tags {
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
							File:    relPath,
							Line:    line,
							Level:   LevelWarn,
							Message: fmt.Sprintf("malformed tag %q (normalized to %q)", tag, norm),
						})
					}
				}

				// Render & Slug combination check
				if matter.Render != nil && !*matter.Render && matter.Slug != "" {
					line := findFieldLine(meta.Content, "slug")
					findings = append(findings, Finding{
						File:    relPath,
						Line:    line,
						Level:   LevelError,
						Message: fmt.Sprintf("invalid render/slug combination: slug %q specified when render is false", matter.Slug),
					})
				}

				// Slug validity check
				if matter.Slug != "" {
					slug := matter.Slug
					if !filepath.IsLocal(slug) || strings.Contains(slug, ".") || strings.Contains(slug, string(filepath.Separator)) || strings.Contains(slug, "/") {
						line := findFieldLine(meta.Content, "slug")
						findings = append(findings, Finding{
							File:    relPath,
							Line:    line,
							Level:   LevelError,
							Message: fmt.Sprintf("invalid slug %q: slug must be a simple local name without slashes or dots", slug),
						})
					}
				}
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

				if _, exists := fileMap[targetRelPath]; !exists {
					lineNo := findLinkLine(meta.Content, meta.Rest, n, dest)
					findings = append(findings, Finding{
						File:    relPath,
						Line:    lineNo,
						Level:   LevelError,
						Message: fmt.Sprintf("broken internal link %q -> %q", dest, targetRelPath),
					})
				}

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
				File:    relPath,
				Line:    0,
				Level:   LevelError,
				Message: fmt.Sprintf("output path collision: %q and %q both map to %q", prev, relPath, relOut),
			})
		} else {
			owners[relOut] = relPath
		}
	}

	// 4. Asset health diagnostics (optional)
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
					File:    relSlash,
					Line:    0,
					Level:   LevelWarn,
					Message: fmt.Sprintf("symlink in asset directory skipped: %s", relSlash),
				})
				return nil
			}

			// Boundary breakout check for asset paths
			if !filepath.IsLocal(relPath) || strings.HasPrefix(relSlash, "..") || !pathutil.IsSafePath(cfg.AssetDir, path) {
				findings = append(findings, Finding{
					File:    relSlash,
					Line:    0,
					Level:   LevelWarn,
					Message: fmt.Sprintf("asset path %q escapes configured asset root %q", relSlash, cfg.AssetDir),
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
					File:    relSlash,
					Line:    0,
					Level:   LevelWarn,
					Message: fmt.Sprintf("asset case-collision / duplicate destination risk: %q and %q map to the same destination %q", prev, relSlash, lowerRel),
				})
			} else {
				assetCaseMap[lowerRel] = relSlash
			}

			// Unsupported or suspicious extension check
			ext := strings.ToLower(filepath.Ext(relSlash))
			if suspiciousImageExts[ext] {
				findings = append(findings, Finding{
					File:    relSlash,
					Line:    0,
					Level:   LevelWarn,
					Message: fmt.Sprintf("unsupported or suspicious image extension %q: prefer web-optimized formats (.png, .jpg, .webp, .svg, .avif)", ext),
				})
			}

			// Large raster asset check
			info, infoErr := d.Info()
			if infoErr == nil && rasterExts[ext] {
				if info.Size() > maxSize {
					findings = append(findings, Finding{
						File:    relSlash,
						Line:    0,
						Level:   LevelWarn,
						Message: fmt.Sprintf("unusually large raster asset (%s > %s threshold)", formatBytes(info.Size()), formatBytes(maxSize)),
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
					File:    relPath,
					Line:    lineNo,
					Level:   LevelWarn,
					Message: fmt.Sprintf("referenced asset path %q escapes asset root", dest),
				})
				return ast.WalkContinue, nil
			}

			// Check existence in AssetDir
			if !validAssets[assetRel] {
				lineNo := findLinkLine(meta.Content, meta.Rest, n, dest)
				if actual, caseMismatch := assetCaseMap[strings.ToLower(assetRel)]; caseMismatch {
					findings = append(findings, Finding{
						File:    relPath,
						Line:    lineNo,
						Level:   LevelWarn,
						Message: fmt.Sprintf("referenced asset %q has case mismatch with existing asset %q", dest, actual),
					})
				} else {
					findings = append(findings, Finding{
						File:    relPath,
						Line:    lineNo,
						Level:   LevelWarn,
						Message: fmt.Sprintf("missing referenced asset %q", dest),
					})
				}
			}

			return ast.WalkContinue, nil
		})
	}

	return findings, nil
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
