package checker

import (
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"

	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/markdown"
	"github.com/tbuddy/la-famille/internal/sitedata"
)

// ExtractManifestReferences returns the internal links and local asset
// references found in the supplied source pages. Link resolution follows the
// same source/output rules as Validate; graph targets follow LinkTransformer's
// .md-only graph contract.
func ExtractManifestReferences(
	fileMap map[string]*content.FileMeta,
	graphExplorer bool,
) (map[string][]sitedata.ManifestLink, map[string][]string) {
	linksByPage := make(map[string][]sitedata.ManifestLink, len(fileMap))
	assetsByPage := make(map[string][]string, len(fileMap))
	expectedOutputs := buildExpectedOutputs(fileMap, graphExplorer)
	engine := markdown.NewEngine(nil)

	for relPath, meta := range fileMap {
		if meta == nil {
			continue
		}
		if assetRef, ok := manifestAssetReference(relPath, meta.Image); ok {
			assetsByPage[relPath] = append(assetsByPage[relPath], assetRef)
		}
		if len(meta.Rest) == 0 {
			continue
		}
		doc := engine.Parser().Parse(text.NewReader(meta.Rest))
		_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}

			switch current := node.(type) {
			case *ast.Link:
				dest := string(current.Destination)
				if ref, ok := manifestLinkReference(relPath, meta, fileMap, expectedOutputs, dest, current); ok {
					linksByPage[relPath] = append(linksByPage[relPath], ref)
				}
				if isAssetLink(dest) {
					if assetRef, ok := manifestAssetReference(relPath, dest); ok {
						assetsByPage[relPath] = append(assetsByPage[relPath], assetRef)
					}
				}
			case *ast.Image:
				if assetRef, ok := manifestAssetReference(relPath, string(current.Destination)); ok {
					assetsByPage[relPath] = append(assetsByPage[relPath], assetRef)
				}
			}

			return ast.WalkContinue, nil
		})
	}

	return linksByPage, assetsByPage
}

func manifestLinkReference(
	relPath string,
	meta *content.FileMeta,
	fileMap map[string]*content.FileMeta,
	expectedOutputs map[string]bool,
	dest string,
	node ast.Node,
) (sitedata.ManifestLink, bool) {
	u, err := url.Parse(dest)
	if err != nil || u.IsAbs() || strings.HasPrefix(dest, "//") || u.Path == "" {
		return sitedata.ManifestLink{}, false
	}

	ext := strings.ToLower(path.Ext(u.Path))
	line := findLinkLine(meta.Content, meta.Rest, node, dest)
	if ext == ".md" {
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
			return sitedata.ManifestLink{}, false
		}

		targetID := strings.TrimSuffix(targetRelPath, ".md")
		target, resolved := fileMap[targetRelPath]
		if resolved && target.Render != nil && !*target.Render {
			targetID = targetRelPath
		}
		graphTarget := ""
		if strings.HasSuffix(u.Path, ".md") {
			graphTarget = targetID
		}
		return sitedata.ManifestLink{
			Destination: dest,
			Target:      targetRelPath,
			GraphTarget: graphTarget,
			Line:        line,
			Resolved:    resolved,
		}, true
	}

	if ext != "" && ext != ".html" {
		return sitedata.ManifestLink{}, false
	}

	raw := u.Path
	if strings.HasPrefix(raw, "/") {
		raw = strings.TrimPrefix(raw, "/")
	} else if dir := filepath.ToSlash(filepath.Dir(relPath)); dir != "." {
		raw = dir + "/" + raw
	}
	cleaned := path.Clean(raw)
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.Contains(dest, "%2E%2E") {
		return sitedata.ManifestLink{}, false
	}
	target := normalizeOutputCandidate(cleaned)
	return sitedata.ManifestLink{
		Destination: dest,
		Target:      target,
		Line:        line,
		Resolved:    expectedOutputFor(expectedOutputs, target),
	}, true
}

func isAssetLink(dest string) bool {
	ext := strings.ToLower(filepath.Ext(dest))
	return strings.HasPrefix(dest, "/assets/") ||
		strings.HasPrefix(dest, "assets/") ||
		rasterExts[ext] ||
		suspiciousImageExts[ext] ||
		ext == ".svg"
}

func manifestAssetReference(relPath, dest string) (string, bool) {
	u, err := url.Parse(dest)
	if err != nil || u.IsAbs() || strings.HasPrefix(dest, "//") || u.Path == "" {
		return "", false
	}

	refPath := u.Path
	var assetRel string
	switch {
	case strings.HasPrefix(refPath, "/assets/"):
		assetRel = strings.TrimPrefix(refPath, "/assets/")
	case strings.HasPrefix(refPath, "assets/"):
		assetRel = strings.TrimPrefix(refPath, "assets/")
	case strings.HasPrefix(refPath, "/"):
		assetRel = strings.TrimPrefix(refPath, "/")
	default:
		dir := filepath.Dir(relPath)
		assetRel = refPath
		if dir != "." {
			assetRel = dir + "/" + refPath
		}
		assetRel = strings.TrimPrefix(assetRel, "assets/")
	}
	return filepath.ToSlash(filepath.Clean(assetRel)), true
}
